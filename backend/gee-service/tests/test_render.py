import json
import struct
import zlib

import pytest

from app.render import MAX_DIMENSION, MIN_DIMENSION, Bounds, bounds_of, dimension_for, grid_for, mercator, png_size


def test_bounds_of_polygon_and_multipolygon():
    polygon = {"type": "Polygon", "coordinates": [[[36.94, -0.42], [36.96, -0.42], [36.962, -0.405], [36.94, -0.42]]]}
    assert bounds_of(json.dumps(polygon)) == Bounds(36.94, -0.42, 36.962, -0.405)

    multi = {"type": "MultiPolygon", "coordinates": [polygon["coordinates"], [[[37.0, -1.0], [37.1, -1.0], [37.1, -0.9], [37.0, -1.0]]]]}
    assert bounds_of(json.dumps(multi)) == Bounds(36.94, -1.0, 37.1, -0.405)


def test_dimension_is_native_resolution_within_limits():
    # ~2.2 km across at the equator: 10 m pixels give ~222.
    small = Bounds(36.94, -0.01, 36.96, 0.01)
    assert dimension_for(small) == MIN_DIMENSION
    # ~5.5 km: native resolution fits.
    medium = Bounds(36.9, -0.025, 36.95, 0.025)
    assert MIN_DIMENSION < dimension_for(medium) < MAX_DIMENSION
    # A county-sized parcel is capped.
    assert dimension_for(Bounds(36.0, -1.0, 38.0, 1.0)) == MAX_DIMENSION


def tiny_png(width, height):
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)
    chunk = struct.pack(">I", len(ihdr)) + b"IHDR" + ihdr + struct.pack(">I", zlib.crc32(b"IHDR" + ihdr))
    return b"\x89PNG\r\n\x1a\n" + chunk


def test_png_size_reads_the_header():
    assert png_size(tiny_png(640, 480)) == (640, 480)


def test_non_png_is_rejected():
    with pytest.raises(ValueError):
        png_size(b'{"error": "nope"}' + b"\x00" * 16)


def test_grid_covers_the_bounds_in_web_mercator():
    b = Bounds(36.94, -0.42, 36.962, -0.405)
    grid = grid_for(b, 600)
    width, height = grid["dimensions"]["width"], grid["dimensions"]["height"]
    t = grid["affineTransform"]
    assert max(width, height) == 600
    assert grid["crsCode"] == "EPSG:3857"

    x0, y0 = mercator(b.west, b.south)
    x1, y1 = mercator(b.east, b.north)
    # Top-left pixel corner is (west, north); bottom-right lands on (east, south).
    assert (t["translateX"], t["translateY"]) == pytest.approx((x0, y1))
    assert t["translateX"] + t["scaleX"] * width == pytest.approx(x1)
    assert t["translateY"] + t["scaleY"] * height == pytest.approx(y0)
