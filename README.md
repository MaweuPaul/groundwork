# Groundwork

Groundwork is an independent verification layer for real-world land asset claims. It replaces trust in paperwork with trust in reproducible satellite data by tracking physical land boundaries and calculating vegetation health indices (like NDVI) over time.

It ingests multi-source satellite imagery (e.g., Sentinel-2 via Google Earth Engine), computes spectral indices using versioned methodology, and serves auditable time-series data over a fast Go + PostGIS backend.

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
│   └── app/(auth)/              # Sign-in and create-account screens
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
    └── gee-service/             # Python Earth Engine worker: all satellite calculations (WIP)
```

## Authentication

- `POST /auth/register`: creates a user (`email`, `password`, `first_name`, `last_name`, `username`, `role`) and returns the user plus a token pair.
- `POST /auth/login`: returns the user plus a token pair. Unknown email and wrong password give the same error.
- `POST /auth/refresh`: swaps a refresh token for a new pair. Each refresh token works once.

Access tokens are short-lived HS256 JWTs (default 15 minutes). Refresh tokens are random 32-byte values (default 30 days), stored only as SHA-256 hashes and rotated on every use. Passwords are hashed with bcrypt.

The frontend calls these endpoints from Next.js server actions and keeps both tokens in `httpOnly` cookies, so browser JavaScript never sees them.

Every route outside `/auth/*` requires `Authorization: Bearer <access token>` (401 otherwise). Role checks return 403. Resources owned by someone else return 404, so their existence is not revealed.

## Parcels

- `GET /parcels`: the signed-in user's parcels, newest first.
- `POST /parcels` (`asset_holder` only): `name`, `geometry` (GeoJSON `Polygon` or `MultiPolygon`), optional `description` and `metadata`. The owner is always the signed-in user; `id`, `owner_id`, `area_ha` and timestamps in the body are ignored. Returns 201 with the saved parcel, including the computed `area_ha`. Invalid geometry returns 400.

Geometry is always returned as a GeoJSON `MultiPolygon`.

## Prerequisites

- **Go** (1.26+)
- **PostgreSQL** with the **PostGIS** extension enabled
- **Node.js** (for the frontend)
- **Python** (for the Google Earth Engine microservice)

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
| `GEE_SERVICE_URL` | no | Unused. Left over from the earlier HTTP design; to be removed. |

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

## Roadmap: Snapshot Pipeline

This is the planned design for running snapshots. It is built so that new indices, satellites and analyses can be added as data and plug-ins, without rewriting the pipeline.

### Design principles

1. **Snapshots are evidence, not cache.** A completed snapshot is never recomputed or overwritten. A better method means a new methodology version and new snapshots.
2. **A methodology is a versioned spec, held as data.** The Python service implements *algorithms*; a methodology row says which algorithm and which parameters to use.
3. **The database is the only link between the services.** Go requests work and serves results; Python does all calculations. Neither calls the other.
4. **Everything needed to reproduce a result is stored with it**: inputs, methodology version, engine version and the exact scene IDs.
5. **The schema is the contract.** Schema changes go through `backend/go-api/migrations` only, in backward-compatible steps (add, update both services, then remove).

### Flow

```text
Frontend ──▶ Go API: POST /parcels/{id}/snapshots
               └─ one transaction: INSERT snapshot (pending, frozen geometry + hash)
                  + INSERT job (queued); trigger sends NOTIFY snapshot_jobs
               ◀─ 202 Accepted { snapshot id }

Python worker ──▶ LISTEN snapshot_jobs (plus a 30–60 s poll as a fallback)
               ├─ claims job (FOR UPDATE SKIP LOCKED); snapshot → processing
               ├─ computes with Google Earth Engine
               └─ one transaction: INSERT measurements + imagery sources,
                  snapshot → completed (record_hash set by a Postgres function)
                  or failed with a reason, or job retried with backoff

Frontend ──▶ Go API: GET /snapshots/{id}   (status, measurements, sources)
```

The job queue lives in Postgres, so there is no extra infrastructure, and a snapshot can never exist without its job. Jobs left `processing` past their lock expiry (a crashed worker) are claimed again.

### Integrity: hash chain

When a snapshot completes, it gets a `record_hash`: a SHA-256 fingerprint of its inputs and outputs in a fixed order (snapshot and parcel IDs, geometry hash, methodology and version, dates, sorted measurements, sorted scene IDs, engine version, completion time). The hash also includes the **previous** snapshot's hash (`prev_record_hash`), forming a chain like git commits.

- The hash is computed by a Postgres function when a snapshot becomes `completed`, so there is one implementation, independent of which service wrote the row. Verification uses the same function.
- The full data is still stored. The hash adds tamper evidence, not compression.
- Changing any stored value no longer matches its hash, and breaks every hash after it.
- A verify command or endpoint recomputes the chain and reports the first record that does not match.
- Optional later step: publish the latest chain hash externally (e.g. OpenTimestamps) to prove the history existed at a point in time. No blockchain or Web3 is required.

### Planned schema changes

| Change | Purpose |
|---|---|
| `parcel_access` | Sharing parcels with auditors (`parcels.owner_id` is done, migration 004) |
| `methodologies.algorithm`, `algorithm_version`, `params` (JSONB) | The executable methodology spec |
| `snapshots.requested_by`, `input_hash`, `engine_version` | Who ran it, request fingerprint (same request returns the existing completed snapshot), which engine computed it |
| `snapshots.record_hash`, `prev_record_hash` | Hash chain, write-once |
| `snapshot_jobs` | Queue: status, attempts, `run_after`, lock owner and expiry, last error; an insert trigger sends `NOTIFY snapshot_jobs` |
| `gee_worker` database role | Least-privilege login for the Python worker (see below) |
| `snapshot_artifacts` | Rasters and files (index maps, change maps, thumbnails) with URI and checksum |
| `analyses` + `analysis_snapshots` | Change analysis across snapshots; only allowed when they share methodology version and geometry hash |
| `audit_events` | Append-only log of who did what and when |

### Go ↔ Python contract: the database

There is no HTTP API between the services. The contract is which tables each one reads and writes.

| | Go API | Python worker (`gee_worker` role) |
|---|---|---|
| `users`, `parcels`, `parcel_access` | read / write | no access |
| `methodologies`, `measurement_types` | read / write | read |
| `snapshots` | insert (pending), read | read; update status, `started_at`, `completed_at`, `failure_message`, `engine_version` |
| `snapshot_jobs` | insert | claim, update attempts / lock / errors |
| `measurements`, `imagery_sources`, `snapshot_artifacts` | read | insert |

The worker reads its inputs from the snapshot row itself: the frozen `parcel_geometry` (not the live parcel), the date range, and the methodology (`algorithm`, `algorithm_version`, `params`). The database triggers that freeze snapshot inputs and completed snapshots apply to both services.

Failures are recorded on the job and snapshot with a code. Terminal: `NO_VALID_IMAGERY`, `INVALID_GEOMETRY`, `UNKNOWN_ALGORITHM` (snapshot → `failed`). Retryable: `GEE_QUOTA`, `GEE_TIMEOUT`, `UPSTREAM_ERROR` (job re-queued with backoff).

The Python worker keeps a registry of algorithms keyed by `(algorithm, algorithm_version)`. Each algorithm validates its own `params`.

### Correctness rules

1. Never report a value without enough clear pixels. Every result includes `valid_pixel_pct`; below the methodology's threshold the snapshot fails with `NO_VALID_IMAGERY`.
2. Record **every** scene that went into the composite.
3. Store all timestamps as UTC ISO-8601.
4. Measure area on the Earth's actual surface (geography), matching PostGIS `area_ha`.
5. Normalize geometry (fixed precision, consistent winding) before hashing.

### Known issues in the current GEE service

- It is still a FastAPI HTTP service; it becomes a database worker (`python -m app.worker`).
- `ee.Initialize()` is called without a Cloud project or credentials, and a failure is only logged, so the service runs without Earth Engine.
- `acquisition_time` passes Earth Engine's millisecond timestamp (a number) into a string field, which Pydantic v2 rejects.
- Only the first 10 scenes are recorded (`toList(10)`), but the composite uses all of them.
- When no clear imagery exists, the result defaults to NDVI `0.0` instead of failing.
- The `QA60` cloud mask is empty for Sentinel-2 scenes from 2022–2024, so clouds pass through. Switch to Cloud Score+ (`GOOGLE/CLOUD_SCORE_PLUS/V1/S2_HARMONIZED`).
- Reflectance is scaled (`/10000`) only when masking is on, and the division drops image properties.
- `satellite` and `bands` from the methodology are ignored; Sentinel-2 B8/B4 are hard-coded.
- The Go GEE client (`service/gee_client.go`) and `GEE_SERVICE_URL` are left over from the HTTP design and will be removed.

### Phases

| Phase | Delivers |
|---|---|
| **0. Foundations** | ✅ Parcel ownership, MultiPolygon parcels, GeoJSON in/out, auth middleware, role permissions. Remaining: `GET /parcels/{id}`, seed methodology `ndvi-s2-median@1.0.0` and measurement types |
| **1. One snapshot end to end** | Migrations (job queue, notify trigger, hash-chain function, `gee_worker` role), Python worker with GEE fixes and algorithm registry, `POST /parcels/{id}/snapshots`, `GET /snapshots/{id}` |
| **2. Hardening** | Retries with backoff, stuck-job recovery, idempotency, mock GEE for tests, contract tests |
| **3. Frontend** | Parcel detail chart, snapshot audit view |
| **4. Change analysis** | `analyses`, per-pixel difference maps, area-based measurements (e.g. `area_ndvi_loss_ha`), same-season comparisons |
| **5. Breadth** | Scheduled monitoring, more indices and satellites, PDF audit export, passkeys/SSO |

### Open decisions

- **Earth Engine access**: Google Cloud project with Earth Engine enabled and a service account key, or start against a mock.
- **Permissions**: proposed: asset holders run snapshots on their own parcels; auditors view and re-run parcels shared with them.
- **Artifact storage** (needed from Phase 4): Google Cloud Storage or local disk in development.
