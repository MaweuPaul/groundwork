"""Which completed snapshots still need an NDVI map image, and saving it (migration 010)."""

import psycopg
from psycopg.rows import class_row
from psycopg.types.json import Jsonb

from app.render import ImageRequest, Rendered


def next_missing(conn: psycopg.Connection) -> ImageRequest | None:
    """The oldest completed snapshot with no image row, or None. Two workers may pick the
    same one; the primary key keeps only the first image."""
    with conn.cursor(row_factory=class_row(ImageRequest)) as cur:
        cur.execute(
            """
            SELECT s.id AS snapshot_id, m.slug AS methodology_slug,
                   ST_AsGeoJSON(s.parcel_geometry) AS geometry_geojson,
                   s.date_start, s.date_end,
                   s.max_scene_cloud_pct::float8 AS max_scene_cloud_pct,
                   s.cloud_score_threshold::float8 AS cloud_score_threshold
            FROM snapshots s
            JOIN methodologies m ON m.id = s.methodology_id
            WHERE s.status = 'completed'
              AND NOT EXISTS (SELECT 1 FROM snapshot_ndvi_images i WHERE i.snapshot_id = s.id)
            ORDER BY s.completed_at
            LIMIT 1
            """
        )
        return cur.fetchone()


def save(conn: psycopg.Connection, req: ImageRequest, image: Rendered) -> None:
    b = image.bounds
    conn.execute(
        """
        INSERT INTO snapshot_ndvi_images (snapshot_id, png, width, height,
                                          west, south, east, north, visualization)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON CONFLICT (snapshot_id) DO NOTHING
        """,
        (req.snapshot_id, image.png, image.width, image.height, b.west, b.south, b.east, b.north, Jsonb(image.visualization)),
    )


def save_error(conn: psycopg.Connection, req: ImageRequest, error: str) -> None:
    """Recorded so the worker doesn't retry it forever; delete the row to try again."""
    conn.execute(
        """
        INSERT INTO snapshot_ndvi_images (snapshot_id, error)
        VALUES (%s, %s)
        ON CONFLICT (snapshot_id) DO NOTHING
        """,
        (req.snapshot_id, error[:1000]),
    )
