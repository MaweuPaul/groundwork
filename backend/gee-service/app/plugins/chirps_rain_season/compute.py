"""Seasonal rainfall for one snapshot with Earth Engine (methodology chirps-rain-season@1.0.0).

Each CHIRPS daily image in the window is averaged over the parcel. Their sum is the window's
rainfall; the mean of the same calendar window's totals over the baseline years is its normal.
Rainy days and the longest dry spell use the snapshot's rainy-day threshold, sealed into its
record. A CHIRPS pixel is about 5.5 km, so values describe the area around the parcel.
"""

import json
from datetime import date, datetime, timedelta, timezone
from statistics import fmean
from typing import Any

import ee

from app.contract import Measurement, Result, Scene, version_text
from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.chirps_rain_season.manifest import (
    BAND,
    BASELINE_END_YEAR,
    BASELINE_START_YEAR,
    COLLECTION,
    MANIFEST,
    NATIVE_SCALE_M,
    REDUCE_SCALE_M,
)
from app.windows import end_exclusive

PROVIDER = "CHIRPS (UCSB Climate Hazards Center) via Google Earth Engine"
PROCESSING_LEVEL = "final"
BASELINE_YEARS = BASELINE_END_YEAR - BASELINE_START_YEAR + 1


def rainy_day_mm(parameters: dict[str, Any]) -> float:
    if "rainy_day_mm" in parameters:
        return float(parameters["rainy_day_mm"])
    return next(p.default for p in MANIFEST.parameters if p.name == "rainy_day_mm")


def check_supported(m: Methodology) -> None:
    supported = (
        m.index_type == "rainfall"
        and m.satellite == "chirps"
        and m.bands == [BAND]
        and m.composite_method == "sum"
    )
    if not supported:
        raise JobError(
            "UNSUPPORTED_METHODOLOGY",
            f"This worker can't compute {m.slug}@{m.version}.",
            retryable=False,
        )


def shift_years(day: date, years: int) -> date:
    try:
        return day.replace(year=day.year + years)
    except ValueError:  # 29 February in a year that has none
        return day.replace(year=day.year + years, day=28)


def baseline_windows(date_start: date, date_end: date) -> list[tuple[date, date]]:
    """The same calendar window in each baseline year. Both ends move by the same number of
    years, so a window that crosses a year end keeps its shape."""
    return [
        (shift_years(date_start, offset), shift_years(date_end, offset))
        for offset in (year - date_start.year for year in range(BASELINE_START_YEAR, BASELINE_END_YEAR + 1))
    ]


def longest_run_below(values: list[float], threshold: float) -> int:
    longest = current = 0
    for value in values:
        current = current + 1 if value < threshold else 0
        longest = max(longest, current)
    return longest


def parcel_mean(image: ee.Image, geometry: ee.Geometry) -> ee.ComputedObject:
    return image.reduceRegion(
        reducer=ee.Reducer.mean(), geometry=geometry, scale=REDUCE_SCALE_M, maxPixels=1e10, tileScale=4
    ).get(BAND)


def run(job: Job, methodology: Methodology) -> Result:
    check_supported(methodology)

    geometry = ee.Geometry(json.loads(job.geometry_geojson))
    daily = ee.ImageCollection(COLLECTION).select(BAND)

    # A list per day keeps a missing mean (None) in place; aggregating properties would drop it.
    days = (
        daily.filterDate(job.date_start.isoformat(), end_exclusive(job.date_end))
        .map(
            lambda image: ee.Feature(
                None,
                {
                    "row": ee.List(
                        [
                            image.get("system:index"),
                            image.get("system:time_start"),
                            image.get("system:version"),
                            parcel_mean(image, geometry),
                        ]
                    )
                },
            )
        )
        .aggregate_array("row")
    )
    baseline_totals = ee.List(
        [
            parcel_mean(daily.filterDate(start.isoformat(), end_exclusive(end)).sum(), geometry)
            for start, end in baseline_windows(job.date_start, job.date_end)
        ]
    )

    # One round trip for everything.
    info = ee.Dictionary({"days": days, "baseline_totals": baseline_totals}).getInfo()
    return summarize(info, job)


def latest_day() -> date:
    """The last day CHIRPS has published, looking back six months."""
    today = datetime.now(timezone.utc).date()
    latest = (
        ee.ImageCollection(COLLECTION)
        .filterDate((today - timedelta(days=183)).isoformat(), end_exclusive(today))
        .aggregate_max("system:time_start")
        .getInfo()
    )
    if latest is None:
        raise RuntimeError(f"{COLLECTION} has no images from the last six months")
    return utc_day(latest).date()


def utc_day(time_start_ms: int) -> datetime:
    return datetime.fromtimestamp(time_start_ms / 1000, tz=timezone.utc)


def summarize(info: dict[str, Any], job: Job) -> Result:
    """Turns Earth Engine's response into measurements. Never reports rainfall for days it
    doesn't have."""
    threshold = rainy_day_mm(job.parameters)
    rows = sorted(info.get("days") or [], key=lambda row: row[1])
    if not rows:
        raise JobError(
            "NO_IMAGERY",
            "No CHIRPS data in this window yet; it is published a few weeks after each month ends.",
            retryable=False,
        )

    expected_days = (job.date_end - job.date_start).days + 1
    if len(rows) < expected_days:
        latest = utc_day(rows[-1][1]).date().isoformat()
        raise JobError(
            "NO_IMAGERY",
            f"CHIRPS has {len(rows)} of the {expected_days} days in this window (latest {latest}); "
            "it is published a few weeks after each month ends. Choose an earlier end date.",
            retryable=False,
        )

    values = [row[3] for row in rows]
    baseline = info.get("baseline_totals") or []
    if any(v is None for v in values) or any(t is None for t in baseline):
        raise JobError(
            "NO_IMAGERY",
            "No CHIRPS data over this parcel; CHIRPS covers land between 50°S and 50°N.",
            retryable=False,
        )
    if len(baseline) != BASELINE_YEARS:
        raise JobError(
            "NO_IMAGERY",
            f"Only {len(baseline)} of the {BASELINE_YEARS} baseline years have CHIRPS data for this window.",
            retryable=False,
        )

    total = sum(values)
    normal = fmean(baseline)
    shared = {
        "days": len(rows),
        "native_scale_m": NATIVE_SCALE_M,
        "reduce_scale_m": REDUCE_SCALE_M,
        "baseline": f"{BASELINE_START_YEAR}-{BASELINE_END_YEAR}",
        "rainy_day_mm": threshold,
    }
    measurements = {
        "rain_total_mm": Measurement(total, shared),
        "rain_normal_mm": Measurement(normal, shared),
        "rainy_days": Measurement(sum(1 for v in values if v >= threshold), shared),
        "longest_dry_spell_days": Measurement(longest_run_below(values, threshold), shared),
    }

    flags = {}
    if normal > 0:
        measurements["rain_pct_of_normal"] = Measurement(total / normal * 100, shared)
    else:
        flags["zero_normal"] = "No rain fell in this window in any baseline year, so there is no % of normal."

    scenes = [
        Scene(
            scene_id=index,
            satellite="chirps",
            acquisition_time=utc_day(time_start),
            cloud_cover_pct=None,
            product_id=None,
            provider=PROVIDER,
            processing_level=PROCESSING_LEVEL,
            asset_uri=f"{COLLECTION}/{index}",
            source_version=version_text(version),
        )
        for index, time_start, version, _ in rows
    ]
    return Result(measurements=measurements, scenes=scenes, flags=flags)
