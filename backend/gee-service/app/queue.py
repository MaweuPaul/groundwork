"""The worker's side of the snapshot_jobs queue (migration 006).

Jobs and snapshots change only through the queue functions, which check that this
worker still holds the job's lock. Measurements and imagery are inserted directly, in
the same transaction as complete_snapshot_job, so a snapshot is never left half-saved.
"""

from uuid import UUID

import psycopg
from psycopg.rows import class_row
from psycopg.types.json import Jsonb

from app.compute import PROVIDER, S2_COLLECTION, Result
from app.errors import JobError
from app.models import Job, Methodology


def claim(conn: psycopg.Connection, worker: str, lock_minutes: int) -> Job | None:
    with conn.cursor(row_factory=class_row(Job)) as cur:
        cur.execute(
            """
            SELECT job_id, snapshot_id, attempt, methodology_id, methodology_version,
                   date_start, date_end,
                   max_scene_cloud_pct::float8 AS max_scene_cloud_pct,
                   cloud_score_threshold::float8 AS cloud_score_threshold,
                   geometry_geojson
            FROM claim_snapshot_job(%s, make_interval(mins => %s))
            """,
            (worker, lock_minutes),
        )
        return cur.fetchone()


def load_methodology(conn: psycopg.Connection, methodology_id: UUID) -> Methodology:
    with conn.cursor(row_factory=class_row(Methodology)) as cur:
        cur.execute(
            """
            SELECT slug, version, index_type, satellite, bands, cloud_mask,
                   max_cloud_pct::float8 AS max_cloud_pct, composite_method
            FROM methodologies
            WHERE id = %s
            """,
            (methodology_id,),
        )
        methodology = cur.fetchone()
    if methodology is None:
        raise JobError("UNKNOWN_METHODOLOGY", f"Methodology {methodology_id} not found.", retryable=False)
    return methodology


def save_and_complete(conn: psycopg.Connection, job: Job, worker: str, result: Result) -> str:
    """Saves the results and completes the job in one transaction. Returns the record hash."""
    with conn.transaction():
        type_ids = dict(
            conn.execute(
                "SELECT slug, id FROM measurement_types WHERE slug = ANY(%s)",
                (list(result.measurements),),
            ).fetchall()
        )
        unknown = set(result.measurements) - set(type_ids)
        if unknown:
            raise JobError("UNKNOWN_MEASUREMENT", f"No measurement types: {sorted(unknown)}", retryable=False)

        with conn.cursor() as cur:
            cur.executemany(
                """
                INSERT INTO measurements (snapshot_id, measurement_type_id, value, metadata)
                VALUES (%s, %s, %s, %s)
                """,
                [
                    (job.snapshot_id, type_ids[slug], m.value, Jsonb(m.metadata))
                    for slug, m in result.measurements.items()
                ],
            )
            cur.executemany(
                """
                INSERT INTO imagery_sources (snapshot_id, provider, satellite, scene_id,
                                             acquisition_time, cloud_cover_pct,
                                             processing_level, asset_uri, metadata)
                VALUES (%s, %s, %s, %s, %s, %s, 'L2A', %s, %s)
                """,
                [
                    (
                        job.snapshot_id,
                        PROVIDER,
                        s.satellite,
                        s.scene_id,
                        s.acquisition_time,
                        s.cloud_cover_pct,
                        f"{S2_COLLECTION}/{s.scene_id}",
                        Jsonb({"product_id": s.product_id} if s.product_id else {}),
                    )
                    for s in result.scenes
                ],
            )

        # Raises if another worker took the job over; the transaction then rolls back.
        (record_hash,) = conn.execute(
            "SELECT complete_snapshot_job(%s, %s)", (job.job_id, worker)
        ).fetchone()
    return record_hash


def fail(conn: psycopg.Connection, job: Job, worker: str, error: JobError) -> str:
    """Records the failure. Returns the job's new status: 'queued' (will retry) or 'failed'."""
    (status,) = conn.execute(
        "SELECT fail_snapshot_job(%s, %s, %s, %s, %s)::text",
        (job.job_id, worker, error.code, error.message, error.retryable),
    ).fetchone()
    return status
