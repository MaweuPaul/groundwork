# Groundwork

Groundwork is an independent verification layer for real-world land asset claims. It replaces trust in paperwork with trust in reproducible satellite data by tracking physical land boundaries and calculating vegetation health indices (like NDVI) over time.

It ingests multi-source satellite imagery (e.g., Sentinel-2 via Google Earth Engine), computes spectral indices using versioned methodology, and serves auditable time-series data over a fast Go + PostGIS backend.

## Core Concepts & Schema

Groundwork relies on a strict, auditable geospatial database using PostgreSQL and the PostGIS extension.

- **Parcels**: Physical land boundaries defined as standard GPS polygons (`GEOMETRY(POLYGON, 4326)`). The system automatically calculates the exact real-world acreage (in hectares) using PostGIS functions that factor in the curvature of the Earth. Parcels can be edited.
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
    │   │   ├── config/          # Environment variables
    │   │   ├── models/          # GORM entities mapped to PostGIS tables
    │   │   ├── repository/      # DB operations
    │   │   ├── service/         # Business logic (auth, tokens, parcels, GEE client)
    │   │   ├── handler/         # HTTP handlers
    │   │   └── database/        # Database connection
    │   ├── migrations/          # Numbered SQL migrations (001_, 002_, ...)
    │   └── .env.example         # Example environment variables
    └── gee-service/             # Python microservice for Google Earth Engine (WIP)
```

## Authentication

- `POST /auth/register`: creates a user (`email`, `password`, `first_name`, `last_name`, `username`, `role`) and returns the user plus a token pair.
- `POST /auth/login`: returns the user plus a token pair. Unknown email and wrong password give the same error.
- `POST /auth/refresh`: swaps a refresh token for a new pair. Each refresh token works once.

Access tokens are short-lived HS256 JWTs (default 15 minutes). Refresh tokens are random 32-byte values (default 30 days), stored only as SHA-256 hashes and rotated on every use. Passwords are hashed with bcrypt.

The frontend calls these endpoints from Next.js server actions and keeps both tokens in `httpOnly` cookies, so browser JavaScript never sees them.

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
| `GEE_SERVICE_URL` | no | Defaults to `http://localhost:8000`. |

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
3. **Computing a result and recording it are separate jobs.** The Python service only computes. The Go side owns the database, statuses and retries.
4. **Everything needed to reproduce a result is stored with it**: inputs, methodology version, engine version and the exact scene IDs.
5. **Contracts are versioned.** The Go ↔ Python API lives under `/v1/` and uses generic shapes.

### Flow

```text
Frontend ──▶ Go API: POST /parcels/{id}/snapshots
               └─ one transaction: INSERT snapshot (pending) + INSERT job (queued)

Go worker ──▶ claims job from Postgres (FOR UPDATE SKIP LOCKED)
               ├─ snapshot → processing
               ├─ POST gee-service /v1/compute ──▶ Google Earth Engine
               └─ one transaction: INSERT measurements + imagery sources,
                  compute record_hash, snapshot → completed
                  (or failed, or retried with backoff)
```

The job queue lives in Postgres, so there is no extra infrastructure, and a snapshot can never exist without its job.

### Integrity: hash chain

When a snapshot completes, it gets a `record_hash`: a SHA-256 fingerprint of its inputs and outputs in a fixed order (snapshot and parcel IDs, geometry hash, methodology and version, dates, sorted measurements, sorted scene IDs, engine version, completion time). The hash also includes the **previous** snapshot's hash (`prev_record_hash`), forming a chain like git commits.

- The full data is still stored. The hash adds tamper evidence, not compression.
- Changing any stored value no longer matches its hash, and breaks every hash after it.
- A verify command or endpoint recomputes the chain and reports the first record that does not match.
- Optional later step: publish the latest chain hash externally (e.g. OpenTimestamps) to prove the history existed at a point in time. No blockchain or Web3 is required.

### Planned schema changes

| Change | Purpose |
|---|---|
| `parcels.owner_id` + `parcel_access` | Parcel ownership, and sharing parcels with auditors |
| `methodologies.algorithm`, `algorithm_version`, `params` (JSONB) | The executable methodology spec |
| `snapshots.requested_by`, `input_hash`, `engine_version` | Who ran it, request fingerprint (same request returns the existing completed snapshot), which engine computed it |
| `snapshots.record_hash`, `prev_record_hash` | Hash chain, write-once |
| `snapshot_jobs` | Queue: status, attempts, `run_after`, lock owner and expiry, last error |
| `snapshot_artifacts` | Rasters and files (index maps, change maps, thumbnails) with URI and checksum |
| `analyses` + `analysis_snapshots` | Change analysis across snapshots; only allowed when they share methodology version and geometry hash |
| `audit_events` | Append-only log of who did what and when |

### Go ↔ Python contract (v1)

```jsonc
// POST /v1/compute
{
  "request_id": "<snapshot id>",
  "algorithm": "optical_index_composite",
  "algorithm_version": "1",
  "params": { /* methodology.params */ },
  "geometry": { /* GeoJSON */ },
  "period": { "start": "2025-03-01", "end": "2025-04-30" }
}

// 200
{
  "engine": { "version": "0.3.0" },
  "measurements": [{ "type": "ndvi_mean", "value": 0.62, "unit": null, "metadata": {} }],
  "sources": [{ "scene_id": "...", "acquired_at": "2025-03-14T08:21:09Z", "cloud_pct": 3.1 }],
  "artifacts": [{ "kind": "index_map", "uri": "gs://...", "checksum": "..." }],
  "warnings": []
}

// error
{ "error": { "code": "NO_VALID_IMAGERY", "retryable": false, "message": "..." } }
```

Terminal errors: `NO_VALID_IMAGERY`, `INVALID_GEOMETRY`, `UNKNOWN_ALGORITHM`. Retryable errors: `GEE_QUOTA`, `GEE_TIMEOUT`, `UPSTREAM_ERROR`.

The Python service keeps a registry of algorithms keyed by `(algorithm, algorithm_version)`. Each algorithm validates its own `params`.

### Correctness rules

1. Never report a value without enough clear pixels. Every result includes `valid_pixel_pct`; below the methodology's threshold the snapshot fails with `NO_VALID_IMAGERY`.
2. Record **every** scene that went into the composite.
3. Store all timestamps as UTC ISO-8601.
4. Measure area on the Earth's actual surface (geography), matching PostGIS `area_ha`.
5. Normalize geometry (fixed precision, consistent winding) before hashing.

### Known issues in the current GEE service

- `acquisition_time` passes Earth Engine's millisecond timestamp (a number) into a string field, which Pydantic v2 rejects.
- Only the first 10 scenes are recorded (`toList(10)`), but the composite uses all of them.
- When no clear imagery exists, the result defaults to NDVI `0.0` instead of failing.
- `satellite` and `bands` from the methodology are ignored; Sentinel-2 B8/B4 are hard-coded.
- Nothing calls the Go GEE client yet.

### Phases

| Phase | Delivers |
|---|---|
| **0. Foundations** | Parcel ownership, auth middleware, role permissions, seed methodology `ndvi-s2-median@1.0.0` and measurement types |
| **1. One snapshot end to end** | Migrations, job queue, worker, `/v1/compute`, GEE service fixes and algorithm registry, hash chain, `POST /parcels/{id}/snapshots`, `GET /snapshots/{id}` |
| **2. Hardening** | Retries with backoff, stuck-job recovery, idempotency, mock GEE for tests, contract tests |
| **3. Frontend** | Parcel detail chart, snapshot audit view |
| **4. Change analysis** | `analyses`, per-pixel difference maps, area-based measurements (e.g. `area_ndvi_loss_ha`), same-season comparisons |
| **5. Breadth** | Scheduled monitoring, more indices and satellites, PDF audit export, passkeys/SSO |

### Open decisions

- **Earth Engine access**: Google Cloud project with Earth Engine enabled and a service account key, or start against a mock.
- **Permissions**: proposed: asset holders run snapshots on their own parcels; auditors view and re-run parcels shared with them.
- **Artifact storage** (needed from Phase 4): Google Cloud Storage or local disk in development.
