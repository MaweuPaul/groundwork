from dataclasses import dataclass
from datetime import date
from typing import Any
from uuid import UUID


@dataclass(frozen=True)
class Job:
    """A claimed job, as returned by claim_snapshot_job."""

    job_id: UUID
    snapshot_id: UUID
    attempt: int
    methodology_id: UUID
    methodology_version: str
    date_start: date
    date_end: date
    # The methodology's parameters for this snapshot, defaults filled in; sealed into its record.
    parameters: dict[str, Any]
    geometry_geojson: str


@dataclass(frozen=True)
class Methodology:
    slug: str
    version: str
    index_type: str
    satellite: str
    bands: list[str]
    cloud_mask: str | None
    max_cloud_pct: float
    composite_method: str
