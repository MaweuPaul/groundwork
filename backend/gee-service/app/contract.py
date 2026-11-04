"""What a methodology plugin hands the worker, and what the worker hands it."""

from collections.abc import Callable
from dataclasses import dataclass, field
from datetime import date, datetime
from typing import Any
from uuid import UUID

from app.models import Job, Methodology


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
    provider: str
    processing_level: str
    asset_uri: str


@dataclass(frozen=True)
class Result:
    measurements: dict[str, Measurement]
    scenes: list[Scene]


@dataclass(frozen=True)
class ImageRequest:
    snapshot_id: UUID
    methodology_slug: str
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


@dataclass(frozen=True)
class Plugin:
    compute: Callable[[Job, Methodology], Result]
    # None when the methodology has no map image.
    render: Callable[[ImageRequest], Rendered] | None = None
