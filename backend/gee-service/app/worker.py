"""Snapshot worker: python -m app.worker

Waits for snapshot jobs (LISTEN snapshot_jobs, with a fallback poll), computes each with
Earth Engine, and saves the result or records the failure. When no job is due, it renders
the NDVI map image of any completed snapshot that lacks one. Run several for more
throughput; SKIP LOCKED in claim_snapshot_job keeps them off each other's jobs.
"""

import logging
import sys
import time

import psycopg

from app import earth_engine, images, queue, registry
from app.config import ConfigError, Settings, load_settings
from app.errors import JobError, classify
from app.models import Job

log = logging.getLogger("worker")

RECONNECT_SECONDS = 5


def process(conn: psycopg.Connection, settings: Settings, job: Job) -> None:
    log.info(
        "job %s: snapshot %s, attempt %d, %s to %s",
        job.job_id, job.snapshot_id, job.attempt, job.date_start, job.date_end,
    )
    try:
        methodology = queue.load_methodology(conn, job.methodology_id)
        result = registry.plugin_for(methodology.slug).compute(job, methodology)
        record_hash = queue.save_and_complete(conn, job, settings.worker_id, result)
    except KeyboardInterrupt:
        # Hand the job back now instead of leaving it locked until the lock expires.
        record_failure(conn, settings, job, JobError("WORKER_STOPPED", "Worker was stopped mid-job.", retryable=True))
        raise
    except Exception as exc:
        record_failure(conn, settings, job, classify(exc))
        return

    m = result.measurements
    log.info(
        "job %s: completed, NDVI mean %.3f, %.0f%% clear, %d scenes, record %s",
        job.job_id, m["ndvi_mean"].value, m["valid_pixel_pct"].value, len(result.scenes), record_hash,
    )


def record_failure(conn: psycopg.Connection, settings: Settings, job: Job, error: JobError) -> None:
    try:
        status = queue.fail(conn, job, settings.worker_id, error)
    except psycopg.Error as exc:
        # Most likely another worker took the job over after our lock expired.
        log.error("job %s: %s; could not record it: %s", job.job_id, error, exc)
        return
    outcome = "will retry" if status == "queued" else "gave up"
    log.warning("job %s: %s (%s)", job.job_id, error, outcome)


def drain(conn: psycopg.Connection, settings: Settings) -> None:
    """Processes jobs until none are due."""
    while job := queue.claim(conn, settings.worker_id, settings.lock_minutes):
        process(conn, settings, job)


def render_next_image(conn: psycopg.Connection) -> bool:
    """Renders one missing NDVI image. Returns False when there was none to render."""
    req = images.next_missing(conn)
    if req is None:
        return False
    try:
        plugin = registry.plugin_for(req.methodology_slug)
        if plugin.render is None:
            images.save_error(conn, req, "This methodology has no map image.")
            return True
        image = plugin.render(req)
    except Exception as exc:
        error = classify(exc)
        log.warning("snapshot %s: NDVI image failed: %s", req.snapshot_id, error)
        images.save_error(conn, req, str(error))
        return True
    images.save(conn, req, image)
    log.info("snapshot %s: NDVI image %dx%d, %d KB", req.snapshot_id, image.width, image.height, len(image.png) // 1024)
    return True


def serve(settings: Settings) -> None:
    while True:
        try:
            with psycopg.connect(settings.database_url, autocommit=True) as conn:
                conn.execute("LISTEN snapshot_jobs")
                log.info("worker %s listening for jobs", settings.worker_id)
                while True:
                    drain(conn, settings)
                    # One image at a time, so a newly queued job never waits behind a backlog.
                    if render_next_image(conn):
                        continue
                    # Returns on the first NOTIFY, or after the poll interval (for retries
                    # whose backoff has passed, and anything missed while busy).
                    for _ in conn.notifies(timeout=settings.poll_seconds, stop_after=1):
                        pass
        except psycopg.OperationalError as exc:
            log.error("database connection lost: %s; reconnecting in %ds", exc, RECONNECT_SECONDS)
            time.sleep(RECONNECT_SECONDS)


def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    try:
        settings = load_settings()
    except ConfigError as exc:
        log.error("%s", exc)
        return 2

    try:
        earth_engine.initialize(settings)
    except Exception as exc:
        log.error("could not sign in to Earth Engine (project %s): %s", settings.gee_project, exc)
        return 2
    log.info("signed in to Earth Engine, project %s", settings.gee_project)

    try:
        serve(settings)
    except KeyboardInterrupt:
        log.info("stopped")
    return 0


if __name__ == "__main__":
    sys.exit(main())
