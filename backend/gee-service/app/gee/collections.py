import ee
from typing import List

from app.schemas.snapshot import ImagerySource

def build_s2_collection(geom: ee.Geometry, date_start: str, date_end: str, max_cloud_pct: float) -> ee.ImageCollection:
    """Filters the Sentinel-2 SR collection to the parcel, date range and scene cloud limit."""
    # For MVP, we hardcode Sentinel-2 SR processing. This can be made dynamic based on `methodology.satellite`.
    return ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED") \
        .filterBounds(geom) \
        .filterDate(date_start, date_end) \
        .filter(ee.Filter.lt('CLOUDY_PIXEL_PERCENTAGE', max_cloud_pct))

def extract_imagery_sources(collection: ee.ImageCollection, satellite: str) -> List[ImagerySource]:
    """Lists the satellite scenes that contributed to the composite."""
    # Note: getInfo() is a blocking call to Earth Engine servers.
    col_list = collection.toList(10) # limit to 10 for metadata extraction speed
    col_info = col_list.getInfo()

    sources = []
    for img_dict in col_info:
        props = img_dict.get('properties', {})
        sources.append(ImagerySource(
            satellite=satellite,
            scene_id=img_dict.get('id', 'unknown'),
            acquisition_time=props.get('system:time_start', 0), # Will convert to string in actual prod
            cloud_cover_pct=props.get('CLOUDY_PIXEL_PERCENTAGE'),
            metadata={'granule_id': props.get('GRANULE_ID')}
        ))
    return sources
