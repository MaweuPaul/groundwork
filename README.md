# Groundwork

Groundwork is an independent verification layer for land claims. It replaces trust in paperwork with trust in reproducible satellite data: it tracks parcel boundaries, measures vegetation (NDVI) from Sentinel-2 imagery through Google Earth Engine with versioned methodologies, and seals every result into a tamper-evident hash chain per parcel.

## How it fits together

```text
Frontend ──▶ Go API ──▶ PostgreSQL ◀── Python worker ──▶ Google Earth Engine
```

| Part | Where | Does |
|---|---|---|
| Frontend | `frontend/` | Next.js app: parcels, snapshots, NDVI maps, audit views |
| Go API | `backend/go-api/` | Accounts, parcels, snapshot requests, results; owns the schema (migrations) |
| Python worker | `backend/gee-service/` | All satellite calculations, as plugins |
| PostgreSQL + PostGIS | | The only link between the two services: a job queue, and the triggers that seal records |

The API and the worker never call each other, so either can be down without stopping the other.

## Quick start

You need Go 1.26+, Node.js, Python 3.12, PostgreSQL with PostGIS, and a Google Cloud service account with Earth Engine access.

```bash
# Database
createdb groundwork && psql groundwork -c 'CREATE EXTENSION postgis'

# API: set DATABASE_URL and JWT_SECRET in .env
cd backend/go-api && cp .env.example .env
go run ./cmd/migrate && go run ./cmd/api

# Worker: see backend/gee-service/README.md for the key and database role
cd backend/gee-service && python -m app.worker

# Frontend: set API_URL in .env.local if the API isn't on http://localhost:8080
cd frontend && npm install && npm run dev
```

Open http://localhost:3000/register to create an account.

## Tests

```bash
cd backend/go-api && go test ./...
cd backend/gee-service && python -m pytest
cd frontend && npx tsc --noEmit && npm run lint
```

The Go integration tests also need a `groundwork_test` database and `TEST_DATABASE_URL`; see [docs/development.md](docs/development.md#tests). CI runs everything on every pull request.

## Docs

| Doc | Covers |
|---|---|
| [docs/development.md](docs/development.md) | Full setup, every setting, tests, project structure |
| [docs/api.md](docs/api.md) | Endpoints, limits and errors |
| [docs/architecture.md](docs/architecture.md) | Concepts, the snapshot pipeline, the hash chain, the Go ↔ Python contract, plugins |
| [docs/deploy.md](docs/deploy.md) | Free-tier deployment on Vercel, Render and Supabase |
| [docs/roadmap.md](docs/roadmap.md) | Phases, open decisions, planned work |
| [docs/design/multi-analysis.md](docs/design/multi-analysis.md) | Design for many analyses as plugins (partly built) |

## Licence note

The free tiers used here, and Earth Engine's free quota, are for noncommercial use. A commercial product needs a paid Earth Engine licence.
