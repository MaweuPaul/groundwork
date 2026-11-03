"""NDVI for one snapshot with Earth Engine (methodology ndvi-s2-median@1.0.0).

Sentinel-2 surface reflectance scenes in the window, under the snapshot's scene cloud limit,
are cloud-masked per pixel with Cloud Score+ at the snapshot's clear threshold, turned into
NDVI, and median-composited. The parcel's mean and max NDVI and its share of clear pixels
are the measurements. Both cloud settings come from the snapshot and are sealed into its record.
"""

import json
from dataclasses import dataclass, field
from datetime import date, datetime, timedelta, timezone
from typing import Any

import ee

from app.errors import JobError
from app.models import Job, Methodology

S2_COLLECTION = "COPERNICUS/S2_SR_HARMONIZED"
CLOUD_SCORE_COLLECTION = "GOOGLE/CLOUD_SCORE_PLUS/V1/S2_HARMONIZED"
SCALE_M = 10
PROVIDER = "Copernicus Sentinel-2 via Google Earth Engine"

SCENE_PROPERTIES = ["system:index", "system:time_start", "CLOUDY_PIXEL_PERCENTAGE", "SPACECRAFT_NAME", "PRODUCT_ID"]


@dataclass(frozen=True)
class Measurement:
    value: float
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True)
class Scene:
    scene_id: str
    satellite: str
    acquisition_time: datetime
    cloud_cover_pct: float | None
    product_id: str | None


@dataclass(frozen=True)
class Result:
    measurements: dict[str, Measurement]
    scenes: list[Scene]


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


def end_exclusive(day: date) -> str:
    """Snapshot windows include their end day; Earth Engine's filterDate excludes it."""
    return (day + timedelta(days=1)).isoformat()


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
        geometry, job.date_start, job.date_end, job.max_scene_cloud_pct, job.cloud_score_threshold
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

    return summarize(info, job)


def summarize(info: dict[str, Any], job: Job) -> Result:
    """Turns Earth Engine's response into measurements. Never invents a value for missing data."""
    rows = info.get("scenes") or []
    if not rows:
        raise JobError(
            "NO_IMAGERY",
            f"No Sentinel-2 scenes with at most {job.max_scene_cloud_pct:g}% cloud cover in this window.",
            retryable=False,
        )

    stats = info.get("stats") or {}
    clear_pixels = stats.get("ndvi_count") or 0
    mean, maximum = stats.get("ndvi_mean"), stats.get("ndvi_max")
    if clear_pixels == 0 or mean is None or maximum is None:
        raise JobError(
            "NO_CLEAR_PIXELS",
            f"All {len(rows)} scenes were cloudy over the parcel at clear threshold "
            f"{job.cloud_score_threshold:g}; no clear pixels to measure.",
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
                "cloud_score_plus_threshold": job.cloud_score_threshold,
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
        )
        for scene_id, time_start, cloud, spacecraft, product_id in rows
    ]
    return Result(measurements=measurements, scenes=scenes)
