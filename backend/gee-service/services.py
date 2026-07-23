import ee
import logging
from typing import Dict, Any, Tuple
from models import SnapshotRequest, SnapshotResponse, ImagerySource

logger = logging.getLogger(__name__)

def mask_s2_clouds(image: ee.Image) -> ee.Image:
    """Masks clouds in Sentinel-2 images using QA60 band."""
    qa = image.select('QA60')
    # Bits 10 and 11 are clouds and cirrus, respectively.
    cloud_bit_mask = 1 << 10
    cirrus_bit_mask = 1 << 11
    # Both flags should be set to zero, indicating clear conditions.
    mask = qa.bitwiseAnd(cloud_bit_mask).eq(0) \
        .And(qa.bitwiseAnd(cirrus_bit_mask).eq(0))
    return image.updateMask(mask).divide(10000)

def add_ndvi(image: ee.Image) -> ee.Image:
    """Computes NDVI and adds it as a band."""
    ndvi = image.normalizedDifference(['B8', 'B4']).rename('ndvi')
    return image.addBands(ndvi)

def process_snapshot(request: SnapshotRequest) -> SnapshotResponse:
    try:
        geom = ee.Geometry(request.geometry_geojson)
    except Exception as e:
        raise ValueError(f"Invalid GeoJSON geometry: {e}")

    # 1. Filter Collection
    # For MVP, we hardcode Sentinel-2 SR processing. This can be made dynamic based on `request.methodology.satellite`.
    collection = ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED") \
        .filterBounds(geom) \
        .filterDate(request.date_start, request.date_end) \
        .filter(ee.Filter.lt('CLOUDY_PIXEL_PERCENTAGE', request.methodology.max_cloud_pct))

    # Apply Cloud Mask
    if request.methodology.cloud_mask == "qa60":
        collection = collection.map(mask_s2_clouds)

    # 2. Extract Metadata about Imagery Used
    # We want to know which satellite scenes contributed to this composite.
    # Note: getInfo() is a blocking call to Earth Engine servers.
    col_list = collection.toList(10) # limit to 10 for metadata extraction speed
    col_info = col_list.getInfo()

    sources = []
    for img_dict in col_info:
        props = img_dict.get('properties', {})
        sources.append(ImagerySource(
            satellite=request.methodology.satellite,
            scene_id=img_dict.get('id', 'unknown'),
            acquisition_time=props.get('system:time_start', 0), # Will convert to string in actual prod
            cloud_cover_pct=props.get('CLOUDY_PIXEL_PERCENTAGE'),
            metadata={'granule_id': props.get('GRANULE_ID')}
        ))
        
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
