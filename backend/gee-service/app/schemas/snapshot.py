from typing import List, Dict, Any, Optional
from pydantic import BaseModel, Field

class MethodologyConfig(BaseModel):
    satellite: str = Field(..., description="e.g., 'sentinel-2'")
    index_type: str = Field(..., description="e.g., 'ndvi'")
    bands: List[str] = Field(default=["B4", "B8"])
    cloud_mask: str = Field(default="qa60")
    max_cloud_pct: float = Field(default=20.0)
    composite_method: str = Field(default="median", description="'median', 'mean', etc.")

class SnapshotRequest(BaseModel):
    # Earth Engine natively accepts GeoJSON, so we'll expect a GeoJSON geometry dict.
    geometry_geojson: Dict[str, Any] = Field(..., description="GeoJSON geometry object of the parcel")
    date_start: str = Field(..., description="YYYY-MM-DD format")
    date_end: str = Field(..., description="YYYY-MM-DD format")
    methodology: MethodologyConfig

class ImagerySource(BaseModel):
    provider: str = "earthengine"
    satellite: str
    scene_id: str
    acquisition_time: str
    cloud_cover_pct: Optional[float] = None
    metadata: Dict[str, Any] = {}

class SnapshotResponse(BaseModel):
    measurements: Dict[str, float] = Field(description="Key-value pairs of calculated metrics like 'ndvi_mean'")
    imagery_sources: List[ImagerySource] = Field(default_factory=list)
