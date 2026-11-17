"""Snapshot worker: python -m app.worker

Waits for snapshot jobs (LISTEN snapshot_jobs, with a fallback poll), computes each with
Earth Engine, and saves the result, with any artifacts such as the map image, or records the
failure. Run several for more throughput; SKIP LOCKED in claim_snapshot_job keeps them off each other's jobs.
"""

import logging
import sys
import time

import psycopg

from app import earth_engine, manifests, queue, registry
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
        record_hash = queue.save_and_complete(conn, job, settings.worker_id, result, settings.code_version)
    except KeyboardInterrupt:
        # Hand the job back now instead of leaving it locked until the lock expires.
        record_failure(conn, settings, job, JobError("WORKER_STOPPED", "Worker was stopped mid-job.", retryable=True))
        raise
    except Exception as exc:
        record_failure(conn, settings, job, classify(exc))
        return

    log.info(
        "job %s: completed, %d measurements, %d scenes, %d artifacts, record %s",
        job.job_id, len(result.measurements), len(result.scenes), len(result.artifacts), record_hash,
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


def serve(settings: Settings) -> None:
    while True:
        try:
            with psycopg.connect(settings.database_url, autocommit=True) as conn:
                conn.execute("LISTEN snapshot_jobs")
                log.info("worker %s listening for jobs", settings.worker_id)
                while True:
                    drain(conn, settings)
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
        with psycopg.connect(settings.database_url) as conn:
            problems = manifests.check(conn, registry.PLUGINS)
    except psycopg.Error as exc:
        log.error("could not check plugin manifests: %s", exc)
        return 2
    if problems:
        for problem in problems:
            log.error("%s", problem)
        return 2
    log.info("plugins: %s; code %s", ", ".join(sorted(registry.PLUGINS)), settings.code_version)

    try:
        serve(settings)
    except KeyboardInterrupt:
        log.info("stopped")
    return 0


if __name__ == "__main__":
    sys.exit(main())
