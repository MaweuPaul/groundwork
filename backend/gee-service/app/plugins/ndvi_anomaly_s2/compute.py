"""NDVI against the parcel's own history (methodology ndvi-anomaly-s2@1.0.0).

The window and the same calendar window in up to five earlier years (none before 2019) are
each composited exactly like an NDVI snapshot. An earlier year counts toward the baseline if at
least min_clear_pct of the parcel was clear; at least three must count. Later years never count,
so a sealed result doesn't depend on when it was computed.
"""

import json
from datetime import date
from statistics import fmean
from typing import Any

import ee

from app.contract import Measurement, Result
from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.ndvi_anomaly_s2.manifest import BASELINE_FIRST_YEAR, BASELINE_MAX_YEARS, BASELINE_MIN_YEARS
from app.plugins.ndvi_s2_median.compute import SCENE_PROPERTIES, build_composite, scene_from_row, setting
from app.plugins.ndvi_s2_median.manifest import SCALE_M
from app.windows import shift_years


def check_supported(m: Methodology) -> None:
    supported = (
        m.index_type == "ndvi_anomaly"
        and m.satellite == "sentinel-2"
        and m.cloud_mask == "cloud_score_plus"
        and m.composite_method == "median"
        and sorted(m.bands) == ["B4", "B8"]
    )
    if not supported:
        raise JobError(
            "UNSUPPORTED_METHODOLOGY",
            f"This worker can't compute {m.slug}@{m.version}.",
            retryable=False,
        )


def baseline_windows(date_start: date, date_end: date) -> list[tuple[date, date]]:
    """The same calendar window in each earlier year, most recent first."""
    windows = []
    for back in range(1, BASELINE_MAX_YEARS + 1):
        start = shift_years(date_start, -back)
        if start.year < BASELINE_FIRST_YEAR:
            break
        windows.append((start, shift_years(date_end, -back)))
    return windows


def window_stats(geometry: ee.Geometry, date_start: date, date_end: date, parameters: dict[str, Any]) -> ee.Dictionary:
    scenes, composite = build_composite(
        geometry,
        date_start,
        date_end,
        setting(parameters, "max_scene_cloud_pct"),
        setting(parameters, "cloud_score_threshold"),
    )
    stats = composite.reduceRegion(
        reducer=ee.Reducer.mean().combine(ee.Reducer.count(), sharedInputs=True),
        geometry=geometry,
        scale=SCALE_M,
        maxPixels=1e10,
        tileScale=4,
    )
    return ee.Dictionary(
        {
            "stats": stats,
            "scenes": scenes.reduceColumns(ee.Reducer.toList(len(SCENE_PROPERTIES)), SCENE_PROPERTIES).get("list"),
        }
    )


def run(job: Job, methodology: Methodology) -> Result:
    check_supported(methodology)

    geometry = ee.Geometry(json.loads(job.geometry_geojson))
    parcel_pixels = ee.Image.constant(1).rename("px").reduceRegion(
        reducer=ee.Reducer.count(), geometry=geometry, scale=SCALE_M, maxPixels=1e10, tileScale=4
    )

    # One round trip for everything.
    info = ee.Dictionary(
        {
            "parcel_pixels": parcel_pixels.get("px"),
            "window": window_stats(geometry, job.date_start, job.date_end, job.parameters),
            "baseline": ee.List(
                [
                    window_stats(geometry, start, end, job.parameters)
                    for start, end in baseline_windows(job.date_start, job.date_end)
                ]
            ),
        }
    ).getInfo()
    return summarize(info, job)


def clear_pct(stats: dict[str, Any], parcel_pixels: int) -> float:
    clear = stats.get("ndvi_count") or 0
    return min(100.0, clear / parcel_pixels * 100) if parcel_pixels else 0.0


def summarize(info: dict[str, Any], job: Job) -> Result:
    """Turns Earth Engine's response into measurements. A year without enough clear pixels is
    left out of the baseline, never filled in."""
    max_cloud = setting(job.parameters, "max_scene_cloud_pct")
    threshold = setting(job.parameters, "cloud_score_threshold")
    min_clear = setting(job.parameters, "min_clear_pct")
    parcel_pixels = info.get("parcel_pixels") or 0

    window = info.get("window") or {}
    rows = window.get("scenes") or []
    if not rows:
        raise JobError(
            "NO_IMAGERY",
            f"No Sentinel-2 scenes with at most {max_cloud:g}% cloud cover in this window.",
            retryable=False,
        )
    stats = window.get("stats") or {}
    mean = stats.get("ndvi_mean")
    if not stats.get("ndvi_count") or mean is None:
        raise JobError(
            "NO_CLEAR_PIXELS",
            f"All {len(rows)} scenes were cloudy over the parcel at clear threshold "
            f"{threshold:g}; no clear pixels to measure.",
            retryable=False,
        )
    valid_pct = clear_pct(stats, parcel_pixels)

    windows = baseline_windows(job.date_start, job.date_end)
    counted: dict[str, float] = {}
    skipped: dict[str, float] = {}
    for (start, _), year in zip(windows, info.get("baseline") or []):
        year_stats = year.get("stats") or {}
        year_clear = clear_pct(year_stats, parcel_pixels)
        if year_stats.get("ndvi_mean") is not None and year_clear >= min_clear:
            counted[str(start.year)] = year_stats["ndvi_mean"]
        else:
            skipped[str(start.year)] = round(year_clear, 1)
        rows = rows + (year.get("scenes") or [])

    if len(counted) < BASELINE_MIN_YEARS:
        raise JobError(
            "INSUFFICIENT_BASELINE",
            f"Only {len(counted)} of {len(windows)} earlier years had at least {min_clear:g}% of the parcel "
            f"clear in this window; {BASELINE_MIN_YEARS} are needed. Try a longer window or a lower minimum.",
            retryable=False,
        )

    baseline = fmean(counted.values())
    shared = {"scale_m": SCALE_M, "composite": "median"}
    baseline_meta = {
        **shared,
        "years": counted,
        "skipped_clear_pct": skipped,
        "first_year": BASELINE_FIRST_YEAR,
        "max_years": BASELINE_MAX_YEARS,
    }
    measurements = {
        "ndvi_anomaly": Measurement(mean - baseline, baseline_meta),
        "ndvi_mean": Measurement(mean, shared),
        "ndvi_baseline_mean": Measurement(baseline, baseline_meta),
        "ndvi_baseline_years": Measurement(len(counted), baseline_meta),
        "valid_pixel_pct": Measurement(
            valid_pct,
            {
                "clear_pixels": stats["ndvi_count"],
                "parcel_pixels": parcel_pixels,
                "scale_m": SCALE_M,
                "cloud_score_plus_threshold": threshold,
            },
        ),
    }

    # A window up to a year long can share a day with the year before, and so a scene.
    unique = {row[0]: row for row in rows}
    scenes = [scene_from_row(row) for row in unique.values()]

    flags = {}
    if valid_pct < min_clear:
        flags["low_coverage"] = f"{valid_pct:.1f}% of the parcel was clear, below the {min_clear:g}% minimum."
    return Result(measurements=measurements, scenes=scenes, flags=flags)
