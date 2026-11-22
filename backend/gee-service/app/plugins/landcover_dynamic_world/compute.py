"""Land cover for one snapshot with Earth Engine (methodology landcover-dynamic-world@1.0.0).

Each pixel takes its most common Dynamic World class over the window's images; the classes are
counted over the parcel and grouped into shares of the classified pixels. Pixels with no
cloud-free view have no class. A snapshot with less than min_valid_pct of the parcel classified
still completes, flagged low_coverage.
"""

import json
from datetime import datetime, timezone
from typing import Any

import ee

from app.contract import Measurement, Result, Scene, version_text
from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.landcover_dynamic_world.manifest import (
    BAND,
    COLLECTION,
    MANIFEST,
    SCALE_M,
    SHARE_OF_CLASS,
    SHARES,
)
from app.windows import end_exclusive

PROVIDER = "Dynamic World V1 (Google and World Resources Institute) via Google Earth Engine"
PROCESSING_LEVEL = "land cover classification"
SCENE_PROPERTIES = ["system:index", "system:time_start", "system:version"]


def min_valid_pct(parameters: dict[str, Any]) -> float:
    if "min_valid_pct" in parameters:
        return float(parameters["min_valid_pct"])
    return next(p.default for p in MANIFEST.parameters if p.name == "min_valid_pct")


def check_supported(m: Methodology) -> None:
    supported = (
        m.index_type == "landcover"
        and m.satellite == "sentinel-2"
        and m.bands == [BAND]
        and m.composite_method == "mode"
    )
    if not supported:
        raise JobError(
            "UNSUPPORTED_METHODOLOGY",
            f"This worker can't compute {m.slug}@{m.version}.",
            retryable=False,
        )


def run(job: Job, methodology: Methodology) -> Result:
    check_supported(methodology)

    geometry = ee.Geometry(json.loads(job.geometry_geojson))
    images = (
        ee.ImageCollection(COLLECTION)
        .filterBounds(geometry)
        .filterDate(job.date_start.isoformat(), end_exclusive(job.date_end))
    )
    region = {"geometry": geometry, "scale": SCALE_M, "maxPixels": 1e10, "tileScale": 4}
    histogram = images.select(BAND).mode().reduceRegion(
        reducer=ee.Reducer.frequencyHistogram().unweighted(), **region
    )
    parcel_pixels = ee.Image.constant(1).rename("px").reduceRegion(reducer=ee.Reducer.count(), **region)

    # One round trip for everything.
    info = ee.Dictionary(
        {
            "histogram": histogram.get(BAND),
            "parcel_pixels": parcel_pixels.get("px"),
            "scenes": images.reduceColumns(ee.Reducer.toList(len(SCENE_PROPERTIES)), SCENE_PROPERTIES).get("list"),
        }
    ).getInfo()
    return summarize(info, job)


def summarize(info: dict[str, Any], job: Job) -> Result:
    """Turns Earth Engine's response into measurements. Never invents a class for an
    unclassified pixel."""
    rows = info.get("scenes") or []
    if not rows:
        raise JobError(
            "NO_IMAGERY",
            "No Dynamic World images of this parcel in this window; try a longer one.",
            retryable=False,
        )

    # Earth Engine returns the class indexes as strings.
    counts = {int(float(label)): count for label, count in (info.get("histogram") or {}).items()}
    classified = sum(counts.values())
    if classified == 0:
        raise JobError(
            "NO_CLEAR_PIXELS",
            f"All {len(rows)} images were cloudy over the parcel; no pixel has a class.",
            retryable=False,
        )
    unknown = sorted(set(counts) - set(SHARE_OF_CLASS))
    if unknown:
        raise JobError("UNEXPECTED_DATA", f"Dynamic World returned unknown classes {unknown}.", retryable=False)

    parcel_pixels = info.get("parcel_pixels") or 0
    valid_pct = min(100.0, classified / parcel_pixels * 100) if parcel_pixels else 0.0

    totals = dict.fromkeys(SHARES, 0)
    for label, count in counts.items():
        totals[SHARE_OF_CLASS[label]] += count

    shared = {"scale_m": SCALE_M, "composite": "mode", "images": len(rows), "classified_pixels": classified}
    measurements = {share: Measurement(total / classified * 100, shared) for share, total in totals.items()}
    measurements["landcover_valid_pct"] = Measurement(
        valid_pct,
        {"classified_pixels": classified, "parcel_pixels": parcel_pixels, "scale_m": SCALE_M},
    )

    scenes = [
        Scene(
            scene_id=index,
            satellite="sentinel-2",
            acquisition_time=datetime.fromtimestamp(time_start / 1000, tz=timezone.utc),
            cloud_cover_pct=None,
            product_id=None,
            provider=PROVIDER,
            processing_level=PROCESSING_LEVEL,
            asset_uri=f"{COLLECTION}/{index}",
            source_version=version_text(version),
        )
        for index, time_start, version in rows
    ]

    flags = {}
    minimum = min_valid_pct(job.parameters)
    if valid_pct < minimum:
        flags["low_coverage"] = f"{valid_pct:.1f}% of the parcel was classified, below the {minimum:g}% minimum."
    return Result(measurements=measurements, scenes=scenes, flags=flags)
