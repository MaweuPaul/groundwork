"""Records how far each methodology's source data reaches, so the API can refuse a window
that ends after it instead of queuing a job that is bound to fail."""

import logging
from datetime import date

import psycopg

from app.contract import Plugin

log = logging.getLogger("coverage")

REFRESH_SECONDS = 6 * 60 * 60


def refresh(conn: psycopg.Connection, plugins: dict[str, Plugin]) -> None:
    """A failed check is logged and skipped: the API then keeps the last known end, and the
    job itself still catches a window the data doesn't cover."""
    for slug, plugin in sorted(plugins.items()):
        if plugin.latest_data is None:
            continue
        try:
            data_end = plugin.latest_data()
            record(conn, slug, data_end)
        except Exception as exc:
            log.warning("could not update how far %s's data reaches: %s", slug, exc)
            continue
        log.info("%s has data to %s", slug, data_end)


def record(conn: psycopg.Connection, slug: str, data_end: date) -> None:
    conn.execute(
        """
        INSERT INTO methodology_coverage (methodology_id, data_end, checked_at)
        SELECT id, %s, now() FROM methodologies WHERE slug = %s
        ON CONFLICT (methodology_id) DO UPDATE SET data_end = EXCLUDED.data_end, checked_at = EXCLUDED.checked_at
        """,
        (data_end, slug),
    )
