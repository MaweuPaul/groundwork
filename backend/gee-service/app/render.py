"""The NDVI map image for a completed snapshot (migration 010).

Re-builds the snapshot's composite from its stored inputs, colours it, clips it to the frozen
boundary and fetches it as a PNG on a Web Mercator grid covering the boundary's bounding box.
The frontend stretches it between those bounds, so the projection has to match the map's.

computePixels needs only the permission already used for measuring; thumbnail URLs would need
an extra Earth Engine role.
"""

import json
import math
import struct
from dataclasses import dataclass
from datetime import date
from uuid import UUID

import ee

from app.compute import SCALE_M, build_composite

# Sequential, light to dark: bare ground pale, dense vegetation dark. Lightness alone carries
# the order, so it reads without colour vision and in greyscale.
PALETTE = ["f3ead2", "d9eaa3", "9cc77c", "4f8a4b", "1b4332"]
NDVI_MIN = 0.0
NDVI_MAX = 0.9
MAX_DIMENSION = 1024
MIN_DIMENSION = 256
EARTH_RADIUS_M = 6_378_137


@dataclass(frozen=True)
class ImageRequest:
    snapshot_id: UUID
    geometry_geojson: str
    date_start: date
    date_end: date
    max_scene_cloud_pct: float
    cloud_score_threshold: float


@dataclass(frozen=True)
class Bounds:
    west: float
    south: float
    east: float
    north: float


@dataclass(frozen=True)
class Rendered:
    png: bytes
    width: int
    height: int
    bounds: Bounds
    visualization: dict


def bounds_of(geojson: str) -> Bounds:
    geometry = json.loads(geojson)
    points = [p for polygon in _polygons(geometry) for ring in polygon for p in ring]
    xs, ys = [p[0] for p in points], [p[1] for p in points]
    return Bounds(min(xs), min(ys), max(xs), max(ys))


def _polygons(geometry: dict) -> list:
    return geometry["coordinates"] if geometry["type"] == "MultiPolygon" else [geometry["coordinates"]]


def dimension_for(b: Bounds) -> int:
    """Longest side in pixels: the native 10 m where possible, between 256 and 1024."""
    mid_lat = math.radians((b.south + b.north) / 2)
    width_m = (b.east - b.west) * 111_320 * math.cos(mid_lat)
    height_m = (b.north - b.south) * 110_540
    native = math.ceil(max(width_m, height_m) / SCALE_M)
    return max(MIN_DIMENSION, min(MAX_DIMENSION, native))


def mercator(lon: float, lat: float) -> tuple[float, float]:
    return (
        EARTH_RADIUS_M * math.radians(lon),
        EARTH_RADIUS_M * math.log(math.tan(math.pi / 4 + math.radians(lat) / 2)),
    )


def grid_for(b: Bounds, longest: int) -> dict:
    """A Web Mercator pixel grid exactly covering b, longest side `longest` pixels."""
    x0, y0 = mercator(b.west, b.south)
    x1, y1 = mercator(b.east, b.north)
    span_x, span_y = x1 - x0, y1 - y0
    if span_x >= span_y:
        width, height = longest, max(1, round(longest * span_y / span_x))
    else:
        width, height = max(1, round(longest * span_x / span_y)), longest
    return {
        "dimensions": {"width": width, "height": height},
        "affineTransform": {
            "scaleX": span_x / width,
            "shearX": 0,
            "translateX": x0,
            "shearY": 0,
            "scaleY": -span_y / height,
            "translateY": y1,
        },
        "crsCode": "EPSG:3857",
    }


def png_size(png: bytes) -> tuple[int, int]:
    if png[:8] != b"\x89PNG\r\n\x1a\n":
        raise ValueError("Earth Engine did not return a PNG.")
    return struct.unpack(">II", png[16:24])


def render(req: ImageRequest) -> Rendered:
    geometry = ee.Geometry(json.loads(req.geometry_geojson))
    _, composite = build_composite(
        geometry, req.date_start, req.date_end, req.max_scene_cloud_pct, req.cloud_score_threshold
    )

    b = bounds_of(req.geometry_geojson)
    visualization = {"min": NDVI_MIN, "max": NDVI_MAX, "palette": [f"#{c}" for c in PALETTE]}
    png = ee.data.computePixels(
        {
            "expression": composite.visualize(min=NDVI_MIN, max=NDVI_MAX, palette=PALETTE).clip(geometry),
            "fileFormat": "PNG",
            "grid": grid_for(b, dimension_for(b)),
        }
    )

    width, height = png_size(png)
    return Rendered(png=png, width=width, height=height, bounds=b, visualization=visualization)
