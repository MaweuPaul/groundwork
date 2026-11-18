"""NDVI for one snapshot with Earth Engine (methodology ndvi-s2-median@1.0.0).

Sentinel-2 surface reflectance scenes in the window, under the snapshot's scene cloud limit,
are cloud-masked per pixel with Cloud Score+ at the snapshot's clear threshold, turned into
NDVI, and median-composited. The parcel's mean and max NDVI and its share of clear pixels
are the measurements. Both cloud settings come from the snapshot and are sealed into its record.
A snapshot with less than min_clear_pct clear pixels still completes, flagged low_coverage.
"""

import json
from dataclasses import replace
from datetime import date, datetime, timezone
from typing import Any

import ee

from app.contract import Measurement, Result, Scene
from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.ndvi_s2_median import render
from app.plugins.ndvi_s2_median.manifest import MANIFEST, SCALE_M
from app.windows import end_exclusive

S2_COLLECTION = "COPERNICUS/S2_SR_HARMONIZED"
CLOUD_SCORE_COLLECTION = "GOOGLE/CLOUD_SCORE_PLUS/V1/S2_HARMONIZED"
PROVIDER = "Copernicus Sentinel-2 via Google Earth Engine"
PROCESSING_LEVEL = "L2A"

SCENE_PROPERTIES = [
    "system:index", "system:time_start", "CLOUDY_PIXEL_PERCENTAGE", "SPACECRAFT_NAME", "PRODUCT_ID", "system:version",
]


def setting(parameters: dict[str, Any], name: str) -> float:
    """A parameter's value, or its manifest default for snapshots created before it existed."""
    if name in parameters:
        return float(parameters[name])
    return next(p.default for p in MANIFEST.parameters if p.name == name)


def check_supported(m: Methodology) -> None:
    supported = (
        m.index_type == "ndvi"
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


def build_composite(
    geometry: ee.Geometry,
    date_start: date,
    date_end: date,
    max_scene_cloud_pct: float,
    cloud_score_threshold: float,
) -> tuple[ee.ImageCollection, ee.Image]:
    """The scenes and the median NDVI composite of a snapshot. Measuring and the map image
    both build it here, so the picture always shows exactly what was measured."""
    scenes = (
        ee.ImageCollection(S2_COLLECTION)
        .filterBounds(geometry)
        .filterDate(date_start.isoformat(), end_exclusive(date_end))
        .filter(ee.Filter.lte("CLOUDY_PIXEL_PERCENTAGE", max_scene_cloud_pct))
    )

    def clear_ndvi(image: ee.Image) -> ee.Image:
        clear = image.select("cs_cdf").gte(cloud_score_threshold)
        # NDVI is a ratio, so the reflectance scale factor (1/10000) cancels out.
        return image.updateMask(clear).normalizedDifference(["B8", "B4"]).rename("ndvi")

    composite = (
        scenes.linkCollection(ee.ImageCollection(CLOUD_SCORE_COLLECTION), ["cs_cdf"])
        .map(clear_ndvi)
        .median()
    )
    return scenes, composite


def run(job: Job, methodology: Methodology) -> Result:
    check_supported(methodology)

    geometry = ee.Geometry(json.loads(job.geometry_geojson))
    scenes, composite = build_composite(
        geometry,
        job.date_start,
        job.date_end,
        setting(job.parameters, "max_scene_cloud_pct"),
        setting(job.parameters, "cloud_score_threshold"),
    )

    reducer = (
        ee.Reducer.mean()
        .combine(ee.Reducer.max(), sharedInputs=True)
        .combine(ee.Reducer.count(), sharedInputs=True)
    )
    region = {"geometry": geometry, "scale": SCALE_M, "maxPixels": 1e10, "tileScale": 4}
    stats = composite.reduceRegion(reducer=reducer, **region)
    parcel_pixels = ee.Image.constant(1).rename("px").reduceRegion(reducer=ee.Reducer.count(), **region)

    # One round trip for everything.
    info = ee.Dictionary(
        {
            "stats": stats,
            "parcel_pixels": parcel_pixels.get("px"),
            "scenes": scenes.reduceColumns(ee.Reducer.toList(len(SCENE_PROPERTIES)), SCENE_PROPERTIES).get("list"),
        }
    ).getInfo()

    return with_map_image(summarize(info, job), geometry, job.geometry_geojson, composite)


def with_map_image(result: Result, geometry: ee.Geometry, geojson: str, composite: ee.Image) -> Result:
    """Adds the NDVI map image, which is part of the evidence but not of the measurement: if
    drawing it fails, the snapshot still completes, with a sealed flag saying so."""
    try:
        image = render.map_image(geometry, geojson, composite)
    except Exception as exc:
        return replace(result, flags={**result.flags, "map_image_failed": str(exc)[:500]})
    return replace(result, artifacts=[*result.artifacts, image])


def summarize(info: dict[str, Any], job: Job) -> Result:
    """Turns Earth Engine's response into measurements. Never invents a value for missing data."""
    max_cloud = setting(job.parameters, "max_scene_cloud_pct")
    threshold = setting(job.parameters, "cloud_score_threshold")
    min_clear = setting(job.parameters, "min_clear_pct")

    rows = info.get("scenes") or []
    if not rows:
        raise JobError(
            "NO_IMAGERY",
            f"No Sentinel-2 scenes with at most {max_cloud:g}% cloud cover in this window.",
            retryable=False,
        )

    stats = info.get("stats") or {}
    clear_pixels = stats.get("ndvi_count") or 0
    mean, maximum = stats.get("ndvi_mean"), stats.get("ndvi_max")
    if clear_pixels == 0 or mean is None or maximum is None:
        raise JobError(
            "NO_CLEAR_PIXELS",
            f"All {len(rows)} scenes were cloudy over the parcel at clear threshold "
            f"{threshold:g}; no clear pixels to measure.",
            retryable=False,
        )

    parcel_pixels = info.get("parcel_pixels") or 0
    valid_pct = min(100.0, clear_pixels / parcel_pixels * 100) if parcel_pixels else 0.0

    shared = {"scale_m": SCALE_M, "composite": "median", "scenes": len(rows)}
    measurements = {
        "ndvi_mean": Measurement(mean, shared),
        "ndvi_max": Measurement(maximum, shared),
        "valid_pixel_pct": Measurement(
            valid_pct,
            {
                "clear_pixels": clear_pixels,
                "parcel_pixels": parcel_pixels,
                "scale_m": SCALE_M,
                "cloud_score_plus_threshold": threshold,
            },
        ),
    }

    scenes = [
        Scene(
            scene_id=scene_id,
            satellite=(spacecraft or "Sentinel-2").lower(),
            acquisition_time=datetime.fromtimestamp(time_start / 1000, tz=timezone.utc),
            cloud_cover_pct=cloud,
            product_id=product_id,
            provider=PROVIDER,
            processing_level=PROCESSING_LEVEL,
            asset_uri=f"{S2_COLLECTION}/{scene_id}",
            source_version=None if version is None else str(version),
        )
        for scene_id, time_start, cloud, spacecraft, product_id, version in rows
    ]

    flags = {}
    if valid_pct < min_clear:
        flags["low_coverage"] = f"{valid_pct:.1f}% of the parcel was clear, below the {min_clear:g}% minimum."
    return Result(measurements=measurements, scenes=scenes, flags=flags)
