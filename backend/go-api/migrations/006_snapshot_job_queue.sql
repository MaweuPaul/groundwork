-- ============================================================
-- Snapshot job queue
--
-- The Go API and the Python worker never call each other. Go
-- queues a job here when it creates a snapshot; the worker
-- claims it, computes with Earth Engine and saves the result.
--
-- Job state (attempts, locks, errors) lives here, not on
-- snapshots, so the evidence table only holds what was measured.
--
-- The worker changes jobs and snapshots only through the
-- functions below (claim, extend, complete, fail), which check
-- that it still holds the job's lock.
-- ============================================================


-- ============================================================
-- Enum types
-- ============================================================

-- queued    → waiting, or waiting to retry after run_after
-- running   → claimed by a worker until locked_until
-- succeeded → snapshot completed
-- failed    → snapshot failed; no more attempts
CREATE TYPE snapshot_job_status AS ENUM (
    'queued',
    'running',
    'succeeded',
    'failed'
);


-- ============================================================
-- Table: snapshot_jobs
--
-- One job per snapshot.
-- ============================================================
CREATE TABLE snapshot_jobs (

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    snapshot_id UUID NOT NULL UNIQUE
        REFERENCES snapshots(id),

    status snapshot_job_status NOT NULL DEFAULT 'queued',

    -- Incremented on every claim, including reclaims after a crash.
    attempts     INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,

    -- Not claimed before this time; used for retry backoff.
    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Which worker holds the job, and until when. A worker that
    -- stops responding loses the job once locked_until passes.
    locked_by    TEXT,
    locked_until TIMESTAMPTZ,

    -- Last failure, e.g. GEE_TIMEOUT / "Earth Engine timed out".
    last_error_code TEXT,
    last_error      TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT snapshot_jobs_attempts_check
        CHECK (attempts >= 0 AND attempts <= max_attempts),

    CONSTRAINT snapshot_jobs_max_attempts_check
        CHECK (max_attempts > 0),

    -- A running job always has an owner and an expiry; other
    -- states never do.
    CONSTRAINT snapshot_jobs_lock_check
        CHECK (
            (status = 'running')
            = (locked_by IS NOT NULL AND locked_until IS NOT NULL)
        )
);

-- Jobs ready to claim.
CREATE INDEX snapshot_jobs_ready_idx
    ON snapshot_jobs (run_after)
    WHERE status = 'queued';

-- Running jobs whose worker may have died.
CREATE INDEX snapshot_jobs_lock_expiry_idx
    ON snapshot_jobs (locked_until)
    WHERE status = 'running';


-- ============================================================
-- Wake the worker
--
-- Sends the job id on channel snapshot_jobs. Postgres delivers
-- it only when the inserting transaction commits, so the worker
-- never sees a job that doesn't exist yet. The worker still
-- polls as a fallback in case it was disconnected.
-- ============================================================
CREATE OR REPLACE FUNCTION notify_snapshot_job()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('snapshot_jobs', NEW.id::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER snapshot_jobs_notify
AFTER INSERT ON snapshot_jobs
FOR EACH ROW
EXECUTE FUNCTION notify_snapshot_job();


-- ============================================================
-- claim_snapshot_job(worker, lock_for)
--
-- Claims the next job and returns everything needed to compute
-- it, or no row when there is nothing to do. Also takes back
-- jobs whose worker stopped responding, and fails those that
-- have no attempts left.
--
-- worker must be unique per running process (e.g. host-pid-uuid).
-- ============================================================
CREATE OR REPLACE FUNCTION claim_snapshot_job(
    p_worker   TEXT,
    p_lock_for INTERVAL DEFAULT INTERVAL '15 minutes'
)
RETURNS TABLE (
    job_id              UUID,
    snapshot_id         UUID,
    attempt             INTEGER,
    methodology_id      UUID,
    methodology_version TEXT,
    date_start          DATE,
    date_end            DATE,
    geometry_geojson    TEXT
) AS $$
#variable_conflict use_column
DECLARE
    v_job_id      UUID;
    v_snapshot_id UUID;
    v_attempt     INTEGER;
BEGIN
    IF p_worker IS NULL OR btrim(p_worker) = '' THEN
        RAISE EXCEPTION 'A worker id is required.';
    END IF;

    -- Abandoned jobs with no attempts left: fail them for good.
    WITH dead AS (
        UPDATE snapshot_jobs j
        SET status          = 'failed',
            locked_by       = NULL,
            locked_until    = NULL,
            last_error_code = 'LOCK_EXPIRED',
            last_error      = 'Worker stopped responding and no attempts are left.',
            updated_at      = NOW()
        WHERE j.status = 'running'
          AND j.locked_until < NOW()
          AND j.attempts >= j.max_attempts
        RETURNING j.snapshot_id
    )
    UPDATE snapshots s
    SET status = 'failed',
        failure_message = 'LOCK_EXPIRED: worker stopped responding and no attempts are left.'
    WHERE s.id IN (SELECT dead.snapshot_id FROM dead);

    -- Next job: queued and due, or abandoned by a dead worker.
    -- SKIP LOCKED lets several workers claim at once without
    -- waiting on, or taking, the same job.
    UPDATE snapshot_jobs j
    SET status       = 'running',
        attempts     = j.attempts + 1,
        locked_by    = p_worker,
        locked_until = NOW() + p_lock_for,
        updated_at   = NOW()
    WHERE j.id = (
        SELECT c.id
        FROM snapshot_jobs c
        WHERE (c.status = 'queued' AND c.run_after <= NOW())
           OR (c.status = 'running' AND c.locked_until < NOW())
        ORDER BY c.run_after, c.created_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING j.id, j.snapshot_id, j.attempts
    INTO v_job_id, v_snapshot_id, v_attempt;

    IF v_job_id IS NULL THEN
        RETURN;
    END IF;

    -- Keep the first start time when a job is retried.
    UPDATE snapshots s
    SET status = 'processing',
        started_at = COALESCE(s.started_at, NOW())
    WHERE s.id = v_snapshot_id;

    RETURN QUERY
    SELECT v_job_id,
           s.id,
           v_attempt,
           s.methodology_id,
           s.methodology_version,
           s.date_start,
           s.date_end,
           ST_AsGeoJSON(s.parcel_geometry)
    FROM snapshots s
    WHERE s.id = v_snapshot_id;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;


-- ============================================================
-- Lock check shared by extend, complete and fail
--
-- Locks the job row and raises unless this worker holds it.
-- A worker whose lock expired and was taken over by another
-- worker is refused, so two workers can never both save.
-- ============================================================
CREATE OR REPLACE FUNCTION lock_owned_snapshot_job(
    p_job_id UUID,
    p_worker TEXT
)
RETURNS snapshot_jobs AS $$
DECLARE
    job snapshot_jobs;
BEGIN
    SELECT *
    INTO job
    FROM snapshot_jobs
    WHERE id = p_job_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Snapshot job % does not exist.', p_job_id;
    END IF;

    IF job.status <> 'running' OR job.locked_by IS DISTINCT FROM p_worker THEN
        RAISE EXCEPTION
            'Snapshot job % is not held by worker %.', p_job_id, p_worker;
    END IF;

    RETURN job;
END;
$$ LANGUAGE plpgsql
SET search_path = public, pg_temp;


-- ============================================================
-- extend_snapshot_job_lock(job, worker, lock_for)
--
-- Heartbeat for long computations, so the job isn't taken back
-- while the worker is still busy with it.
-- ============================================================
CREATE OR REPLACE FUNCTION extend_snapshot_job_lock(
    p_job_id   UUID,
    p_worker   TEXT,
    p_lock_for INTERVAL DEFAULT INTERVAL '15 minutes'
)
RETURNS TIMESTAMPTZ AS $$
DECLARE
    new_until TIMESTAMPTZ := NOW() + p_lock_for;
BEGIN
    PERFORM lock_owned_snapshot_job(p_job_id, p_worker);

    UPDATE snapshot_jobs
    SET locked_until = new_until,
        updated_at = NOW()
    WHERE id = p_job_id;

    RETURN new_until;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;


-- ============================================================
-- complete_snapshot_job(job, worker)
--
-- Call last, in the same transaction that inserted the
-- measurements and imagery sources. Marks the snapshot completed
-- (the hash chain trigger stamps it) and returns its record_hash.
-- If anything fails, the whole transaction, inserts included,
-- rolls back.
-- ============================================================
CREATE OR REPLACE FUNCTION complete_snapshot_job(
    p_job_id UUID,
    p_worker TEXT
)
RETURNS TEXT AS $$
DECLARE
    job  snapshot_jobs;
    hash TEXT;
BEGIN
    job := lock_owned_snapshot_job(p_job_id, p_worker);

    UPDATE snapshots
    SET status = 'completed'
    WHERE id = job.snapshot_id
    RETURNING record_hash INTO hash;

    UPDATE snapshot_jobs
    SET status          = 'succeeded',
        locked_by       = NULL,
        locked_until    = NULL,
        last_error_code = NULL,
        last_error      = NULL,
        updated_at      = NOW()
    WHERE id = p_job_id;

    RETURN hash;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;


-- ============================================================
-- fail_snapshot_job(job, worker, code, message, retryable)
--
-- Retryable errors (GEE_QUOTA, GEE_TIMEOUT, UPSTREAM_ERROR) put
-- the job back in the queue with exponential backoff: 1, 2, 4,
-- 8 ... minutes, at most 1 hour. Terminal errors, or running out
-- of attempts, fail the job and the snapshot.
--
-- Returns the job's new status: 'queued' or 'failed'.
-- ============================================================
CREATE OR REPLACE FUNCTION fail_snapshot_job(
    p_job_id    UUID,
    p_worker    TEXT,
    p_code      TEXT,
    p_message   TEXT,
    p_retryable BOOLEAN
)
RETURNS snapshot_job_status AS $$
DECLARE
    job snapshot_jobs;
BEGIN
    IF p_code IS NULL OR btrim(p_code) = '' THEN
        RAISE EXCEPTION 'An error code is required.';
    END IF;

    job := lock_owned_snapshot_job(p_job_id, p_worker);

    IF p_retryable AND job.attempts < job.max_attempts THEN
        UPDATE snapshot_jobs
        SET status          = 'queued',
            run_after       = NOW() + LEAST(
                                  INTERVAL '1 minute' * power(2, job.attempts - 1),
                                  INTERVAL '1 hour'
                              ),
            locked_by       = NULL,
            locked_until    = NULL,
            last_error_code = p_code,
            last_error      = p_message,
            updated_at      = NOW()
        WHERE id = p_job_id;

        RETURN 'queued';
    END IF;

    UPDATE snapshot_jobs
    SET status          = 'failed',
        locked_by       = NULL,
        locked_until    = NULL,
        last_error_code = p_code,
        last_error      = p_message,
        updated_at      = NOW()
    WHERE id = p_job_id;

    UPDATE snapshots
    SET status = 'failed',
        failure_message = p_code || ': ' || COALESCE(p_message, '')
    WHERE id = job.snapshot_id;

    RETURN 'failed';
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp;


-- ============================================================
-- Role: gee_worker
--
-- The Python worker's login. It can read what it needs and
-- insert results, but changes jobs and snapshots only through
-- the functions above. It has no access to users, parcels or
-- tokens.
--
-- Created without a password, so it can't log in yet. Set one
-- by hand (never in a migration, which is committed):
--     ALTER ROLE gee_worker PASSWORD '...';
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gee_worker') THEN
        CREATE ROLE gee_worker LOGIN;
    END IF;

    EXECUTE format('GRANT CONNECT ON DATABASE %I TO gee_worker', current_database());
END;
$$;

GRANT USAGE ON SCHEMA public TO gee_worker;

-- Inputs.
GRANT SELECT ON methodologies, measurement_types, snapshots, snapshot_jobs
    TO gee_worker;

-- Results. SELECT lets the worker check what it already wrote.
GRANT SELECT, INSERT ON measurements, imagery_sources
    TO gee_worker;

-- Functions are executable by everyone by default; limit the
-- queue functions to the worker (and the owner).
REVOKE ALL ON FUNCTION claim_snapshot_job(TEXT, INTERVAL) FROM PUBLIC;
REVOKE ALL ON FUNCTION extend_snapshot_job_lock(UUID, TEXT, INTERVAL) FROM PUBLIC;
REVOKE ALL ON FUNCTION complete_snapshot_job(UUID, TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION fail_snapshot_job(UUID, TEXT, TEXT, TEXT, BOOLEAN) FROM PUBLIC;
REVOKE ALL ON FUNCTION lock_owned_snapshot_job(UUID, TEXT) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION claim_snapshot_job(TEXT, INTERVAL) TO gee_worker;
GRANT EXECUTE ON FUNCTION extend_snapshot_job_lock(UUID, TEXT, INTERVAL) TO gee_worker;
GRANT EXECUTE ON FUNCTION complete_snapshot_job(UUID, TEXT) TO gee_worker;
GRANT EXECUTE ON FUNCTION fail_snapshot_job(UUID, TEXT, TEXT, TEXT, BOOLEAN) TO gee_worker;
