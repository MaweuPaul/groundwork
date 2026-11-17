# API

The Go API's endpoints. Every route outside `/auth/*` needs `Authorization: Bearer <access token>`.

## Authentication

- `POST /auth/register`: creates a user (`email`, `password`, `first_name`, `last_name`, `username`, `role`) and returns the user plus a token pair.
- `POST /auth/login`: returns the user plus a token pair. Unknown email and wrong password give the same error.
- Logins and sign-ups are rate-limited (429 with `Retry-After`); the limits and `PROXY_SECRET` are in [development.md](development.md#backend-api).
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

- `POST /parcels/{id}/snapshots` (`asset_holder`, own parcels only): `methodology` (slug), `methodology_version` (required; never defaults to the latest), `date_start`, `date_end` (`YYYY-MM-DD`), and optional `parameters`, e.g. `{"min_clear_pct": 40}`. Each methodology lists its parameters with their ranges and defaults on `GET /methodologies`; any left out get their default, and the full set is sealed into the record. For NDVI these are `max_scene_cloud_pct` (0.01–100, default 20), `cloud_score_threshold` (0.40–0.80, default 0.60) and `min_clear_pct` (0–100, default 50). The older top-level `max_scene_cloud_pct` and `cloud_score_threshold` fields still work when `parameters` doesn't set them. Returns **202** with `{"id", "status": "pending"}` and a `Location: /snapshots/{id}` header; the Python worker computes it in the background.
  - In one transaction, the snapshot copies the parcel's current geometry, stores a `geometry_hash` of a normalised copy (coordinates snapped to ~1 cm, points in a standard order, so the same boundary always hashes the same), and queues its job.
  - A request identical to one still pending or processing (same parcel, methodology, window and parameters) returns that snapshot instead of queuing another.
  - 429 when the user's daily snapshot allowance is used up, 422 when the parcel is larger than the area limit (see `SNAPSHOTS_PER_DAY` and `MAX_SNAPSHOT_AREA_HA`).
  - 400 if a date is malformed, `date_end` is before `date_start` or in the future, `date_start` is before 2015-06-23 (first Sentinel-2 imagery), the range is longer than a year, the methodology version doesn't exist, or a parameter is unknown or out of range. 404 if the parcel doesn't exist or isn't yours.
- `GET /parcels/{id}/snapshots`: the parcel's snapshot history, newest first, with each snapshot's `parameters`, `flags`, measurements and `record_hash`. 404 if the parcel doesn't exist or isn't yours.
- `POST /parcels/{id}/series` (`asset_holder`): a time series. The same fields, where the dates are the whole range, plus `window_months` and `interval_months` (1–12). Creates one snapshot per window in one transaction (at most 24; a last window that overruns is cut short at `date_end`). Every window counts against the daily allowance, and an identical series still running is returned instead of queued again. Returns **202** with `{"id", "short_id", "windows"}`.
- `GET /parcels/{id}/series`: the parcel's series with progress and, once every window has finished, `record_hash` and `sealed_at`.
- `GET /snapshots/{id}/ndvi.png`: the NDVI map image of a completed snapshot. `ndvi_image` on snapshot responses gives its bounds and colour scale. 404 if there is none yet or it isn't yours.
- `GET /methodologies`: the methodologies a snapshot can be requested with, each with its `parameters` (name, label, type, min, max, default).
- `GET /snapshots/{id}`: one snapshot of your own. Poll it after creating a snapshot until `status` is `completed` or `failed`. Returns the status (and `failure_message`), inputs (methodology, dates, `parameters`, `geometry_hash`), quality `flags` (e.g. `low_coverage`, which doesn't fail the snapshot), timings, `measurements` as a `{type: value}` map, `imagery_sources` (scene, satellite, acquisition time, cloud %), and once completed, `parcel_seq` (its position in the parcel's chain), `record_hash`, `prev_record_hash`, and from record format v4 `code_version` and `manifest_sha256`. `job` shows queue progress: attempts, `next_attempt_at` while waiting to retry, and the last error code (the worker's raw message stays internal). The frozen geometry is not included. 404 if it doesn't exist or isn't yours.
