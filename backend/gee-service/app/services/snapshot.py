import ee
import logging

from app.schemas.snapshot import SnapshotRequest, SnapshotResponse
from app.gee.collections import build_s2_collection, extract_imagery_sources
from app.gee.masks import mask_s2_clouds
from app.gee.indices import add_ndvi

logger = logging.getLogger(__name__)

def process_snapshot(request: SnapshotRequest) -> SnapshotResponse:
    try:
        geom = ee.Geometry(request.geometry_geojson)
    except Exception as e:
        raise ValueError(f"Invalid GeoJSON geometry: {e}")

    # 1. Filter Collection
    collection = build_s2_collection(
        geom,
        request.date_start,
        request.date_end,
        request.methodology.max_cloud_pct,
    )

    # Apply Cloud Mask
    if request.methodology.cloud_mask == "qa60":
        collection = collection.map(mask_s2_clouds)

    # 2. Extract Metadata about Imagery Used
    # We want to know which satellite scenes contributed to this composite.
    sources = extract_imagery_sources(collection, request.methodology.satellite)

    # 3. Calculate Index
    if request.methodology.index_type.lower() == "ndvi":
        collection = collection.map(add_ndvi)
        band_to_reduce = 'ndvi'
    else:
        raise ValueError(f"Unsupported index type: {request.methodology.index_type}")

    # 4. Composite
    if request.methodology.composite_method == "median":
        composite = collection.median()
    elif request.methodology.composite_method == "mean":
        composite = collection.mean()
    else:
        composite = collection.median()

    # 5. Reduce Region
    stats = composite.select(band_to_reduce).reduceRegion(
        reducer=ee.Reducer.mean().combine(reducer2=ee.Reducer.max(), sharedInputs=True),
        geometry=geom,
        scale=10, # 10m resolution for Sentinel-2
        maxPixels=1e9
    )

    stats_info = stats.getInfo()

    # Map GEE outputs back to our expected measurements
    measurements = {
        f"{request.methodology.index_type}_mean": stats_info.get(f'{band_to_reduce}_mean', 0.0),
        f"{request.methodology.index_type}_max": stats_info.get(f'{band_to_reduce}_max', 0.0)
    }

    return SnapshotResponse(
        measurements=measurements,
        imagery_sources=sources
    )
