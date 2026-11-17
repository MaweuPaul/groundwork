# Development

Running Groundwork locally, its settings, and the tests.

## Prerequisites

- **Go** (1.26+)
- **PostgreSQL** with the **PostGIS** extension enabled
- **Node.js** (for the frontend)
- **Python** 3.12 (for the Earth Engine worker)
- A **Google Cloud service account** with Earth Engine access (an API key won't work)

## Database

Create a database with PostGIS enabled:
```sql
CREATE DATABASE groundwork;
\c groundwork
CREATE EXTENSION IF NOT EXISTS postgis;
```

## Backend API

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
| `PROXY_SECRET` | no | At least 32 characters, the same value as the frontend's `PROXY_SECRET`. With it, logins and sign-ups are rate-limited per visitor instead of per frontend server. |
| `BEHIND_PROXY` | no | `true` when a proxy appends the caller to `X-Forwarded-For` (Render). |
| `SNAPSHOTS_PER_DAY` | no | Snapshots one user may queue in any 24 hours, each series window counted. Defaults to `30`; `0` turns it off. |
| `MAX_SNAPSHOT_AREA_HA` | no | The largest parcel a snapshot may cover. Defaults to `50000` (500 km²); `0` turns it off. |

Every snapshot spends the Earth Engine project's compute quota, hence the two limits above. A request identical to one still pending or processing (same parcel, methodology, window and parameters) returns that snapshot instead of queuing another, so a double click costs nothing.

Logins are limited to 10 at once then 1 a minute per visitor, and 5 at once then 1 every 3 minutes per account; sign-ups to 3 at once then 1 every 20 minutes per visitor. Limits are kept in memory, so they reset when the API restarts.

Apply migrations, then start the server:
```bash
go run ./cmd/migrate   # applies only migrations not yet recorded in schema_migrations
go run ./cmd/api
```

If `parcels` already exists from running `001_initial_schema.sql` by hand, the migrate command records 001 as applied instead of re-running it.

## Frontend

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

## Earth Engine worker

See [backend/gee-service/README.md](../backend/gee-service/README.md) for the service account, `.env` and database role, then:
```bash
cd backend/gee-service
python -m app.worker
```

## Tests

Unit tests need nothing set up:
```bash
cd backend/go-api && go test ./...
cd backend/gee-service && python -m pytest
cd frontend && npx tsc --noEmit && npm run lint
```

The Go integration tests in `internal/integration` run the repositories and the database's own rules against a real PostGIS database: parameter checks, the job queue, sealing, the hash chain, the triggers that freeze completed snapshots, series sealing and the activity feed. They need a separate database whose name ends in `_test`; they refuse any other name, run the migrations themselves, and roll back every test, so it stays empty. Create it once as a superuser:
```sql
CREATE DATABASE groundwork_test;
\c groundwork_test
CREATE EXTENSION postgis;
CREATE EXTENSION pgcrypto;
```
Then run them with `TEST_DATABASE_URL` set (without it they are skipped):
```bash
cd backend/go-api
TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/groundwork_test?sslmode=disable" go test ./internal/integration/
```

CI ([.github/workflows/ci.yml](../.github/workflows/ci.yml)) runs all of this on every pull request and push to `main`, with the integration tests against a `postgis/postgis:16-3.4` service.

## Project structure

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
    │   │   ├── integration/     # Tests against a real PostGIS database (see Tests)
    │   │   ├── middleware/      # Authentication, roles, request logging, panic recovery
    │   │   ├── migrate/         # Applies migrations; used by cmd/migrate and the integration tests
    │   │   ├── models/          # GORM entities mapped to PostGIS tables
    │   │   ├── ratelimit/       # In-memory token buckets for login and sign-up limits
    │   │   ├── repository/      # DB operations
    │   │   ├── routes/          # One file per feature; auth applied once to the protected group
    │   │   └── service/         # Business logic (auth, tokens, parcels)
    │   ├── migrations/          # Numbered SQL migrations (001_, 002_, ...); the only place the schema changes
    │   └── .env.example         # Example environment variables
    └── gee-service/             # Python Earth Engine worker: all satellite calculations
```
