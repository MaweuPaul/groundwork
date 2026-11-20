# Codebase guide

A tour for someone new to the code. It follows one request through every layer, shows where common changes go, and lists the rules the code keeps.

- Setup, settings and the folder list: [development.md](development.md).
- Concepts and the reasons behind the design: [architecture.md](architecture.md).
- Endpoints: [api.md](api.md).

## The three parts

| Part | Language | Does | Never does |
|---|---|---|---|
| `frontend/` | Next.js, TypeScript | Pages; calls the Go API from the server with the user's token | Talk to the database or Earth Engine |
| `backend/go-api/` | Go | Accounts, parcels, snapshot requests, results; owns the schema | Satellite calculations |
| `backend/gee-service/` | Python | Every satellite calculation, as plugins | Serve HTTP or change the schema |

PostgreSQL connects the two backends. The API writes a job, the worker picks it up, and triggers in the database seal the result. The Go API and the Python worker never call each other.

## One request, end to end: running a snapshot

**1. The browser.**
- The parcel page is `frontend/app/(app)/parcels/[slug]/page.tsx`. It is a server component: it fetches the parcel, its snapshots, series and groups before rendering.
- "Run snapshot" opens `components/features/parcels/run-snapshot-dialog.tsx`, which posts a form to the server action `runSnapshotAction` in `app/(app)/parcels/[slug]/actions.ts`.

**2. The Next.js server.**
- The action reads the form and calls `createSnapshot` in `lib/api.ts`. If several methodologies were chosen it calls `createGroup` instead, and for a series it calls `createSeries`.
- `api()` adds the access token from the session cookie and calls the Go API. Browser code never sees the token or the API's address.

**3. The Go API.** It is layered, and each layer has one job:

| Layer | File | Job |
|---|---|---|
| Route | `internal/routes/snapshots.go` | Maps `POST /parcels/{id}/snapshots` to a handler; `RequireRole` limits it to asset holders |
| Middleware | `internal/middleware/` | Checks the token, logs the request, recovers from panics |
| Handler | `internal/handler/snapshot_handler.go` | HTTP only: resolves `{id}` (UUID or short ID), decodes the body, turns errors into status codes |
| Service | `internal/service/snapshot_service.go` | Rules: dates, the methodology's `data_start`, parameters and their ranges |
| Repository | `internal/repository/snapshot_repo.go` | SQL. `CreateWithJob` checks ownership and limits, copies the parcel's boundary into the snapshot and queues its job, all in one transaction |

**4. The database.** Inserting the job fires the `snapshot_jobs_notify` trigger, which sends `NOTIFY snapshot_jobs`.

**5. The worker** (`backend/gee-service/app/`).
- `worker.py` listens for that notification. `queue.claim` takes the job through the database function `claim_snapshot_job`, so two workers never take the same job.
- `registry.plugin_for(slug)` finds the plugin in `app/plugins/<slug>/`, and its `compute.py` asks Earth Engine for the result.
- The plugin returns a `Result` (see `contract.py`): measurements, sources, flags and artifacts such as the NDVI map.

**6. Sealing.**
- `queue.save_and_complete` writes the result and completes the job in one transaction.
- When the snapshot's status becomes `completed`, the `stamp_snapshot_hash` trigger writes the record text, hashes it, links it to the parcel's previous record and moves the parcel's chain head forward. If the snapshot belongs to a group and was the last member to finish, `seal_snapshot_group` seals the group.
- No application code computes these hashes, so neither service can get them wrong.
- If the job fails, `queue.fail` calls `fail_snapshot_job`. A temporary error is retried with backoff; a permanent one fails the snapshot with a readable message.

**7. Back in the browser.** While a snapshot is pending, `LiveRefresh` re-renders the page every few seconds. `SnapshotNotifier` polls `/api/activity` and shows a toast when a snapshot finishes.

## Where common changes go

**A new endpoint.** Work from the bottom up:
1. Add the repository method, with the SQL and the ownership check.
2. Add the service method, with the rules.
3. Add the handler and register the route in `internal/routes/<feature>.go`.
4. Add an integration test in `internal/integration/`.
5. Describe the endpoint in [api.md](api.md).
6. On the frontend, add the type to `lib/types.ts` and the call to `lib/api.ts`.

`GET /parcels/{id}/verify` is a small, complete example: `verify_repo.go`, `VerifyParcel` in the service and handler, `verify_test.go`, and `chain-check.tsx`.

**A schema change.**
- Add the next numbered file to `backend/go-api/migrations/` and apply it with `go run ./cmd/migrate`.
- Never edit a migration that has already been applied.
- Make changes in backward-compatible steps (add, update both services, then remove), because the API and the worker are deployed separately.

**A new methodology.** Follow [Adding a methodology](architecture.md#adding-a-methodology). In short:
1. A migration publishes the methodology, its measurement types, its parameters, and which measurements it produces (`methodology_measurements`, in the manifest's order).
2. A plugin folder in `app/plugins/` holds `manifest.py`, `compute.py` and `__init__.py`.
3. An entry in `frontend/lib/methodologies.ts` gives its title, icon, inputs, formula and help text.

The rest of the frontend picks it up from there.

**A new page.**
- Add `frontend/app/(app)/<route>/page.tsx`. Anything inside `(app)` requires a login and gets the sidebar.
- Put its components in `components/features/<area>/`, and any form actions in an `actions.ts` next to the page.
- Add the link in `components/shared/app-sidebar.tsx`.

## Rules the code keeps

- **Records are evidence.** A completed snapshot is never recomputed, edited or deleted. A better method is a new methodology version, which produces new snapshots.
- **The database enforces its own rules.** Triggers freeze inputs and completed results and seal records. Go and Python check things too, but only to give a readable error; the database guarantees them.
- **Someone else's data answers 404.** A parcel or snapshot that isn't yours looks the same as one that doesn't exist, so IDs can't be probed.
- **Integration tests run only against a `*_test` database.** They refuse any other name and roll back every test.
- **Comments explain why, not what.** Add one only when the reason isn't obvious from the code.
- **Secrets stay out of git.** Use `.env` files and `backend/gee-service/credentials/`, both ignored. Private agreements go in `legal/`, which is also ignored.

## Tests

The commands are in [development.md](development.md#tests).

| Suite | Covers |
|---|---|
| `go-api/internal/service` | Validation rules, with fake repositories |
| `go-api/internal/integration` | The repositories and the database's own rules against real PostGIS: queue, sealing, chains, triggers, groups, series, verification |
| `gee-service/tests` | Plugin calculations that don't need Earth Engine, the registry and the worker's contract |
| frontend | Type checking and lint |

CI runs all of them on every pull request.
