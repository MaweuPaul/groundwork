# Groundwork

Groundwork is an independent verification layer for real-world land asset claims. It replaces trust in paperwork with trust in reproducible satellite data by tracking physical land boundaries and calculating vegetation health indices (like NDVI) over time.

It ingests multi-source satellite imagery (e.g., Sentinel-2 via Google Earth Engine), computes spectral indices using versioned methodology, and serves auditable time-series data over a fast Go + PostGIS backend.

See [docs/architecture.md](docs/architecture.md) for diagrams of how the parts talk to each other and how to add a methodology.

## Architecture

Two backend services, and they never call each other. PostgreSQL is the only thing they share.

| Service | Owns |
|---|---|
| **Go API** (`backend/go-api`) | Users, sign-in and roles; parcels and ownership; requesting snapshots; serving results; the database schema (migrations) |
| **Python worker** (`backend/gee-service`) | All satellite calculations (NDVI, EVI, land cover, ...) through Google Earth Engine; writing measurements and imagery sources |

```text
Frontend ──▶ Go API ──▶ PostgreSQL ◀── Python worker ──▶ Google Earth Engine
```

The Go API queues work by inserting rows; the Python worker picks them up, computes, and writes results back. Either service can be down without stopping the other: if the worker is down, snapshot requests wait in the queue; if the API is down, the worker finishes the jobs already queued. Python is used because Earth Engine's official client libraries are Python and JavaScript.

### System design

How the parts talk to each other today. The Go API and the Python worker only meet in PostgreSQL.

![Current components and how they talk](docs/diagrams/architecture-1.svg)

## Core Concepts & Schema

Groundwork relies on a strict, auditable geospatial database using PostgreSQL and the PostGIS extension.

- **Parcels**: Physical land boundaries owned by one user, stored as `GEOMETRY(MULTIPOLYGON, 4326)` so one parcel can have several pieces (e.g. split by a river or road). A single polygon is stored as a one-piece MultiPolygon. The system automatically calculates the exact real-world area (in hectares, summed over all pieces) using PostGIS functions that factor in the curvature of the Earth. Invalid boundaries (self-intersecting, overlapping pieces) are rejected. Parcels can be edited.
- **Methodologies**: Immutable, version-controlled specifications that define *how* a satellite measurement is calculated (e.g., which satellite bands to use, cloud masking thresholds, and how to composite multiple images).
- **Snapshots**: A single measurement run. It acts as the bridge connecting a specific Parcel, a specific Methodology, and a specific Date Range. A snapshot stores its own copy of the parcel geometry (plus a hash of it), so editing a parcel never changes past snapshots.
- **Measurements**: The actual computed index values (e.g., the mean NDVI value of the parcel).
- **Imagery Sources**: A permanent audit trail logging the exact raw satellite imagery used to calculate a specific Snapshot.
- **Users**: Accounts with a role, either `auditor` (verifies claims) or `asset_holder` (registers parcels).

Database triggers enforce the audit guarantees: snapshot inputs can never change, completed snapshots cannot be deleted or reopened, and their measurements and imagery records are frozen.

## Project Structure

This repository is a monorepo containing the frontend and two separate backend services.

```text
/
├── frontend/                    # Next.js / React application
│   ├── app/(auth)/              # Sign-in and create-account screens
│   └── app/(app)/               # Overview, parcels, register parcel, snapshot pages
├── docs/                        # Architecture, design proposals, diagrams
└── backend/
    ├── go-api/                  # Go + PostGIS API
    │   ├── cmd/
    │   │   ├── api/             # HTTP server
    │   │   └── migrate/         # Applies pending SQL migrations
    │   ├── internal/
    │   │   ├── app/             # Wires repositories, services and handlers together
    │   │   ├── config/          # Environment variables
    │   │   ├── database/        # Database connection
    │   │   ├── handler/         # HTTP handlers
    │   │   ├── httpx/           # Shared JSON request/response helpers
    │   │   ├── middleware/      # Authentication, roles, request logging, panic recovery
    │   │   ├── models/          # GORM entities mapped to PostGIS tables
    │   │   ├── repository/      # DB operations
    │   │   ├── routes/          # One file per feature; auth applied once to the protected group
    │   │   └── service/         # Business logic (auth, tokens, parcels)
    │   ├── migrations/          # Numbered SQL migrations (001_, 002_, ...); the only place the schema changes
    │   └── .env.example         # Example environment variables
    └── gee-service/             # Python Earth Engine worker: all satellite calculations
```

## Authentication

- `POST /auth/register`: creates a user (`email`, `password`, `first_name`, `last_name`, `username`, `role`) and returns the user plus a token pair.
- `POST /auth/login`: returns the user plus a token pair. Unknown email and wrong password give the same error.
- `POST /auth/refresh`: swaps a refresh token for a new pair. Each refresh token works once; a token rotated in the last 10 seconds may be reused, so parallel requests and tabs don't sign the user out.
- `POST /auth/logout`: `refresh_token`; ends that session at once (no reuse window). Always 204.
- `GET /me`: the signed-in user.

Access tokens are short-lived HS256 JWTs (default 15 minutes). Refresh tokens are random 32-byte values (default 30 days), stored only as SHA-256 hashes and rotated on every use. Passwords are hashed with bcrypt.

The frontend calls these endpoints from Next.js server actions and keeps both tokens in `httpOnly` cookies, so browser JavaScript never sees them.

Every route outside `/auth/*` requires `Authorization: Bearer <access token>` (401 otherwise). Role checks return 403. Resources owned by someone else return 404, so their existence is not revealed.

## Parcels

Wherever a URL has `{id}`, it accepts the UUID or the 8-character `short_id`. Frontend URLs use the short ID: `/parcels/nyeri-block-a-k7x2p9qd`, `/snapshots/s4m8q2wz`. The name part is cosmetic; a stale name or a UUID link redirects to the current URL.

- `GET /parcels`: the signed-in user's parcels, newest first. Each includes `centroid_lon`/`centroid_lat`, `latest_snapshot` (most recent, any status) and `latest_result` (measurements of the most recent completed snapshot); both are `null` when there are none.
- `GET /parcels/{id}`: one parcel. 404 if it doesn't exist or belongs to someone else.
- `POST /parcels` (`asset_holder` only): `name`, `geometry` (GeoJSON `Polygon` or `MultiPolygon`), optional `description` and `metadata`. The owner is always the signed-in user; `id`, `owner_id`, `area_ha` and timestamps in the body are ignored. Returns 201 with the saved parcel, including the computed `area_ha`. Invalid geometry returns 400.

Geometry is always returned as a GeoJSON `MultiPolygon`.

## Snapshots

- `POST /parcels/{id}/snapshots` (`asset_holder`, own parcels only): `methodology` (slug), `methodology_version` (required; never defaults to the latest), `date_start`, `date_end` (`YYYY-MM-DD`), optional `max_scene_cloud_pct` (above 0, up to 100; defaults to the methodology's value) and `cloud_score_threshold` (0.40–0.80; defaults to 0.60). Both are sealed into the record. Returns **202** with `{"id", "status": "pending"}` and a `Location: /snapshots/{id}` header; the Python worker computes it in the background.
  - In one transaction, the snapshot copies the parcel's current geometry, stores a `geometry_hash` of a normalised copy (coordinates snapped to ~1 cm, points in a standard order, so the same boundary always hashes the same), and queues its job.
  - 400 if a date is malformed, `date_end` is before `date_start` or in the future, `date_start` is before 2015-06-23 (first Sentinel-2 imagery), the range is longer than a year, or the methodology version doesn't exist. 404 if the parcel doesn't exist or isn't yours.
- `GET /parcels/{id}/snapshots`: the parcel's snapshot history, newest first, with each snapshot's measurements and `record_hash`. 404 if the parcel doesn't exist or isn't yours.
- `POST /parcels/{id}/series` (`asset_holder`): a time series. The same fields, where the dates are the whole range, plus `window_months` and `interval_months` (1–12). Creates one snapshot per window in one transaction (at most 24; a last window that overruns is cut short at `date_end`). Returns **202** with `{"id", "short_id", "windows"}`.
- `GET /parcels/{id}/series`: the parcel's series with progress and, once every window has finished, `record_hash` and `sealed_at`.
- `GET /snapshots/{id}/ndvi.png`: the NDVI map image of a completed snapshot. `ndvi_image` on snapshot responses gives its bounds and colour scale. 404 if there is none yet or it isn't yours.
- `GET /methodologies`: the methodologies a snapshot can be requested with.
- `GET /snapshots/{id}`: one snapshot of your own. Poll it after creating a snapshot until `status` is `completed` or `failed`. Returns the status (and `failure_message`), inputs (methodology, dates, `geometry_hash`), timings, `measurements` as a `{type: value}` map, `imagery_sources` (scene, satellite, acquisition time, cloud %), and once completed, `chain_seq`, `record_hash` and `prev_record_hash`. `job` shows queue progress: attempts, `next_attempt_at` while waiting to retry, and the last error code (the worker's raw message stays internal). The frozen geometry is not included. 404 if it doesn't exist or isn't yours.

## Prerequisites

- **Go** (1.26+)
- **PostgreSQL** with the **PostGIS** extension enabled
- **Node.js** (for the frontend)
- **Python** 3.12 (for the Earth Engine worker)
- A **Google Cloud service account** with Earth Engine access (an API key won't work)

## Setup Instructions

### 1. Database
Create a database with PostGIS enabled:
```sql
CREATE DATABASE groundwork;
\c groundwork
CREATE EXTENSION IF NOT EXISTS postgis;
```

### 2. Backend API
```bash
cd backend/go-api
cp .env.example .env
```

Edit `.env`:

| Variable | Required | Notes |
|---|---|---|
| `DATABASE_URL` | yes | e.g. `postgres://postgres:password@localhost:5432/groundwork?sslmode=disable`. URL-encode special characters in the password (`#` → `%23`). |
| `JWT_SECRET` | yes | At least 32 characters. Generate with `openssl rand -hex 64`. |
| `PORT` | no | Defaults to `8080`. |
| `ACCESS_TOKEN_TTL` | no | Defaults to `15m`. |
| `REFRESH_TOKEN_TTL` | no | Defaults to `720h` (30 days). |

Apply migrations, then start the server:
```bash
go run ./cmd/migrate   # applies only migrations not yet recorded in schema_migrations
go run ./cmd/api
```

If `parcels` already exists from running `001_initial_schema.sql` by hand, the migrate command records 001 as applied instead of re-running it.

### 3. Frontend
```bash
cd frontend
npm install
npm run dev
```

If the API is not on `http://localhost:8080`, create `frontend/.env.local`:
```
API_URL=http://localhost:8081
```

Sign-in is at `/login` and create-account at `/register`.

### 4. Earth Engine worker
See [backend/gee-service/README.md](backend/gee-service/README.md) for the service account, `.env` and database role, then:
```bash
cd backend/gee-service
python -m app.worker
```

## Snapshot Pipeline

How snapshots run today. The proposal for adding more analyses is under [Proposed: analysis plugins](#proposed-analysis-plugins).

### Design principles

1. **Snapshots are evidence, not cache.** A completed snapshot is never recomputed or overwritten. A better method means a new methodology version and new snapshots.
2. **A methodology is a versioned spec.** A methodology row is immutable; the worker's registry maps its slug to the code that computes it.
3. **The database is the only link between the services.** Go requests work and serves results; Python does all calculations. Neither calls the other.
4. **Everything needed to reproduce a result is stored with it**: inputs, cloud settings, methodology version and the exact scene IDs. Dataset versions are not stored yet (see the proposal).
5. **The schema is the contract.** Schema changes go through `backend/go-api/migrations` only, in backward-compatible steps (add, update both services, then remove).

### Flow

```text
Frontend ──▶ Go API: POST /parcels/{id}/snapshots
               └─ one transaction: INSERT snapshot (pending, frozen geometry + hash)
                  + INSERT job (queued); trigger sends NOTIFY snapshot_jobs
               ◀─ 202 Accepted { snapshot id }

Python worker ──▶ LISTEN snapshot_jobs (plus a 30–60 s poll as a fallback)
               ├─ claims job (FOR UPDATE SKIP LOCKED); snapshot → processing
               ├─ computes with Google Earth Engine (methodology looked up in app/registry.py)
               └─ one transaction: INSERT measurements + imagery sources,
                  snapshot → completed (record_hash set by a Postgres function)
                  or failed with a reason, or job retried with backoff
               └─ when idle: renders the NDVI map image of completed snapshots

Frontend ──▶ Go API: GET /snapshots/{id}   (status, measurements, sources)
```

The job queue lives in Postgres, so there is no extra infrastructure, and a snapshot can never exist without its job. Jobs left `processing` past their lock expiry (a crashed worker) are claimed again.

### Life of a snapshot

![Sequence of a snapshot from request to sealed result](docs/diagrams/architecture-2.svg)

### Integrity: hash chain

When a snapshot completes, it gets a `record_hash`: a SHA-256 fingerprint of its inputs and outputs in a fixed order (snapshot and parcel IDs, methodology and version, cloud settings, dates, geometry hashes, completion time, sorted measurements, sorted scene IDs). The hash also includes the **previous** snapshot's hash (`prev_record_hash`), forming a chain like git commits.

- The hash is computed by a Postgres function when a snapshot becomes `completed`, so there is one implementation, independent of which service wrote the row. Verification uses the same function.
- The full data is still stored. The hash adds tamper evidence, not compression.
- Changing any stored value no longer matches its hash, and breaks every hash after it.
- `verify_snapshot_chain()` recomputes the chain and reports every record that does not match. There is no API endpoint for it yet.
- Records keep the format they were sealed with (`record_format`), so old hashes still verify when the format changes. New records use format v2.
- **Series:** each window is sealed as its own snapshot. When the last window finishes, a trigger seals the series with one hash over its settings and every window's result in date order, so a window can't be dropped later. Series hashes are stored on the series, not in the snapshot chain.
- The NDVI map image is a picture of the evidence and is not part of the hash.
- Optional later step: publish the latest chain hash externally (e.g. OpenTimestamps) to prove the history existed at a point in time. No blockchain or Web3 is required.

### Schema status

| Change | Purpose |
|---|---|
| ✅ `snapshots.chain_seq`, `record_hash`, `prev_record_hash` | Hash chain, write-once (migration 005) |
| ✅ `snapshot_jobs`, `gee_worker` role | Queue and least-privilege worker login (migration 006) |
| ✅ `parcels.short_id`, `snapshots.short_id` | Clean URLs (migration 008) |
| ✅ `snapshots.max_scene_cloud_pct`, `cloud_score_threshold`, `record_format` | Per-snapshot cloud settings, sealed from format v2 (migration 009) |
| ✅ `snapshot_ndvi_images` | NDVI map images, outside the hash (migration 010) |
| ✅ `snapshot_series`, `snapshots.series_id` | Time series and their seal (migration 011) |
| `parcel_access` | Sharing parcels with auditors |
| `snapshots.requested_by`, `input_hash` | Who ran it; same request returns the existing completed snapshot |
| `audit_events` | Append-only log of who did what and when |

### Go ↔ Python contract: the database

There is no HTTP API between the services. The contract is which tables each one reads and writes.

| | Go API | Python worker (`gee_worker` role) |
|---|---|---|
| `users`, `parcels`, `parcel_access` | read / write | no access |
| `methodologies`, `measurement_types` | read / write | read |
| `snapshots` | insert (pending), read | read; changed only through the queue functions |
| `snapshot_jobs` | insert (with the snapshot, same transaction) | read; changed only through the queue functions |
| `measurements`, `imagery_sources`, `snapshot_ndvi_images` | read | read, insert |
| `snapshot_series` | insert, read | no direct access; sealed by a trigger |

The worker has no direct `UPDATE` rights. It drives jobs through four functions, each of which checks that it still holds the job's lock:

| Function | Does |
|---|---|
| `claim_snapshot_job(worker, lock_for)` | Claims the next due job (or one abandoned by a dead worker) and returns its inputs: job and snapshot IDs, attempt, methodology, dates, cloud settings, GeoJSON geometry. No row means nothing to do. |
| `extend_snapshot_job_lock(job, worker, lock_for)` | Heartbeat during long computations. |
| `complete_snapshot_job(job, worker)` | Called last, in the transaction that inserted the results. Completes the snapshot (the hash trigger stamps it) and returns `record_hash`. |
| `fail_snapshot_job(job, worker, code, message, retryable)` | Retryable: re-queued with backoff (1, 2, 4… minutes, at most 1 hour) until `max_attempts`. Otherwise the job and snapshot fail with the code and message. |

`worker` must be unique per running process. A worker whose lock expired and was taken over by another is refused, so two workers can never both save a result. The `gee_worker` role is created without a password; set one by hand with `ALTER ROLE gee_worker PASSWORD '...'`.

The worker reads its inputs from the snapshot row itself: the frozen `parcel_geometry` (not the live parcel), the date range, the cloud settings and the methodology. The database triggers that freeze snapshot inputs and completed snapshots apply to both services.

Failures are recorded on the job and snapshot with a code. Terminal: `NO_IMAGERY`, `NO_CLEAR_PIXELS`, `COMPUTATION_TOO_LARGE` (over Earth Engine's limit for one request), `UNSUPPORTED_METHODOLOGY`, `GEE_ERROR` (snapshot → `failed`). Retryable: `GEE_QUOTA`, `GEE_TIMEOUT`, `UPSTREAM_ERROR` (job re-queued with backoff).

The worker finds each methodology's code in `app/registry.py`, keyed by slug.

### Correctness rules

1. Never report a value without clear pixels: the snapshot fails with `NO_IMAGERY` or `NO_CLEAR_PIXELS`, never `0`. Every result includes `valid_pixel_pct`. There is no minimum percentage yet.
2. Record **every** scene that went into the composite.
3. Store all timestamps as UTC ISO-8601.
4. Measure area on the Earth's actual surface (geography), matching PostGIS `area_ha`.
5. Normalize geometry (fixed precision, consistent winding) before hashing.

### Phases

| Phase | Delivers |
|---|---|
| **0. Foundations** | ✅ Ownership, MultiPolygon parcels, GeoJSON, auth, roles, methodology seed |
| **1. One snapshot end to end** | ✅ Queue, hash chain, worker on Earth Engine, snapshot endpoints |
| **2. Hardening** | ✅ Retries with backoff, stuck-job recovery. Remaining: idempotency, mock Earth Engine, contract tests |
| **3. Frontend** | ✅ Overview, parcels, register parcel, parcel detail, snapshot audit view, NDVI map, time series |
| **4. Analysis plugins** | Proposed below |
| **5. Change analysis** | Per-pixel difference maps, area-based measurements, same-season comparisons |
| **6. Breadth** | Scheduled monitoring, PDF audit export, auditor sharing, passkeys/SSO |

### Open decisions

- **Permissions**: proposed: asset holders run snapshots on their own parcels; auditors view and re-run parcels shared with them.
- **Artifact storage** (needed from Phase 4): Google Cloud Storage or local disk in development.
- **Earth Engine licence**: free for non-commercial use only; a commercial product needs a paid licence.

## Proposed: analysis plugins

**Status: proposal, not built.** Goal: add analyses (rainfall, phenology, water history, terrain, EUDR) without touching the core each time. Diagrams and the data model are in [docs/design/multi-analysis.md](docs/design/multi-analysis.md); today's system is in [docs/architecture.md](docs/architecture.md).

Built so far: the worker's methodology registry (`app/registry.py`). The job fields, image table and frontend still assume NDVI.

### What a plugin is

```text
backend/gee-service/app/plugins/
├── ndvi_s2_median/
│   ├── manifest.py    name, version, parameters, outputs, datasets, data start date
│   ├── compute.py     the calculation
│   ├── render.py      map image (optional)
│   └── test_plugin.py
└── chirps_rain_season/
```

The manifest is the single source of truth. On startup the worker publishes each manifest to `methodologies`; the Go API validates requests against it, and the frontend builds its form and help text from it. Go and Python still only meet in Postgres. A plugin that uses an existing output type (value, time series, map, events) needs no Go or frontend change.

Plugins live in this repo and are reviewed like other code. They are not installed at runtime, because sealed records are only trustworthy if the code behind them is known.

### Versions and hashes

Versions and hashes do different jobs, so both stay. A version **names** what ran; a hash **proves** a record hasn't changed.

| Thing | Version | Hash | Why |
|---|---|---|---|
| Methodology / plugin | ✅ `1.0.0` | ✅ manifest hash, sealed into each record | A version is only a label; the hash catches a recipe changed without a new version |
| Plugin code | ✅ with the plugin | Git commit recorded with each record | Shows exactly which code ran |
| Datasets | ✅ collection ID and version | ❌ | Too large to hash; they can be reprocessed, so the version must be stored |
| Parameters | ❌ | ✅ inside the record hash | Chosen per snapshot |
| Snapshot, series, analysis, report | ❌ | ✅ | These are the evidence |
| Record format | ✅ `record_format` | ❌ | Old records must keep verifying under their own format |
| Map images and other artifacts | ❌ | ✅ SHA-256, once they become `artifacts` | Today's NDVI image isn't hashed |

The worker refuses to start if a manifest differs from what is already published under the same version: changing a calculation means a new version, and old records stay reproducible.

### Proposed system design

**Components and how they talk.** Each analysis is a plugin; analysis plugins only read records that are already sealed.

![Proposed components with plugins, batch exports and reports](docs/diagrams/multi-analysis-1.svg)

**Layers of sealed evidence.** Each layer's hash covers the hashes below it.

![Scenes, snapshot, series, analysis and report layers](docs/diagrams/multi-analysis-2.svg)

**Data model.** New and changed tables.

![Proposed data model](docs/diagrams/multi-analysis-3.svg)

**How a combined analysis runs.** Rainfall-adjusted productivity as the example.

![Sequence of a combined analysis](docs/diagrams/multi-analysis-4.svg)

**The plugin contract.** What every plugin declares and returns.

![Plugin and Result contract](docs/diagrams/multi-analysis-5.svg)

### Proposed changes

| Change | Replaces or adds |
|---|---|
| `methodologies`: `kind`, `parameter_spec`, `data_start`, `needs_window`, `datasets`, `manifest_hash` | Adds the plugin's declaration |
| `methodology_outputs` | Declared outputs: key, kind, unit, native resolution |
| `snapshots`: `parameters`, `dataset_versions`, `code_version`, `quality`; record format v3 | Cloud columns become NDVI's parameters. `quality` can be `insufficient_resolution` |
| `series_points` | Dated values; a missing observation is null, never 0 |
| `artifacts` | Replaces `snapshot_ndvi_images`; adds media type and SHA-256 |
| `analyses`, `analysis_inputs` | Sealed results built over other sealed records, e.g. RESTREND |
| `reports`, `report_items` | Bundles for buyers, lenders and exporters, worded as evidence, not certification |
| `reference_areas` | Frozen, hashed areas for peer benchmarks |
| Earth Engine batch exports | Jobs over the 5-minute limit for one request |

### Adding a methodology

What exists today, and what the proposal adds.

![Built and planned extension points](docs/diagrams/architecture-3.svg)

### Build order

1. Plugin folders, auto-discovery and manifests; NDVI moved onto them with no change in behaviour.
2. Generic parameters, dataset and code versions, manifest hash, quality flags; record format v3.
3. `series_points` and `artifacts`; methodology-aware frontend views.
4. Second plugin, `chirps-rain-season`, end to end.
5. Prototype water history and rainfall-adjusted residuals on parcels with known ground truth.
6. Analyses and batch exports, then reports.
