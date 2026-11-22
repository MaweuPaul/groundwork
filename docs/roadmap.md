# Roadmap

## Phases

| Phase | Delivers |
|---|---|
| **0. Foundations** | ✅ Ownership, MultiPolygon parcels, GeoJSON, auth, roles, methodology seed |
| **1. One snapshot end to end** | ✅ Queue, hash chain, worker on Earth Engine, snapshot endpoints |
| **2. Hardening** | ✅ Retries with backoff, stuck-job recovery, integration tests and CI on PostGIS, rate-limited logins and sign-ups, per-user snapshot limits, duplicate requests answered with the queued snapshot, one hash chain per parcel. Remaining: mock Earth Engine, chain checkpoints |
| **3. Frontend** | ✅ Overview, parcels, register parcel, parcel detail, snapshot audit view, NDVI map, time series, methodologies |
| **4. Analysis plugins** | ✅ Plugin folders, manifests, generic parameters, record formats v4 (manifest hash, worker code version) and v5 (sealed artifacts), a second plugin (CHIRPS rainfall), combined runs |
| **5. Change analysis** | Per-pixel difference maps, area-based measurements, same-season comparisons |
| **6. Independent verification** | In progress: ✅ chain check (`GET /parcels/{id}/verify` and the "Recheck chain" button). Next: an audit export with a standalone script that rechecks every hash without the Groundwork server, a GeoJSON boundary export, read-only sharing with an auditor, then claims and verdicts |
| **7. Breadth** | Scheduled monitoring, PDF audit export, passkeys/SSO |

## Open decisions

- **Permissions**: proposed: asset holders run snapshots on their own parcels; auditors view and re-run parcels shared with them.
- **Artifact storage** (needed from Phase 4): Google Cloud Storage or local disk in development.
- **Earth Engine licence**: free for non-commercial use only; a commercial product needs a paid licence.

## Planned schema

The schema as built is in [backend/go-api/migrations](../backend/go-api/migrations), one numbered file per change.

| Change | Purpose |
|---|---|
| `parcel_access` | Sharing parcels with auditors |
| `snapshots.requested_by` | Who ran each snapshot |
| `audit_events` | Append-only log of who did what and when |
| Chain checkpoints | A periodic hash over every parcel's chain head, published outside the database |

## Analysis plugins

**Status: partly built** (see [Build order](#build-order)). Goal: add analyses (rainfall, phenology, water history, terrain, EUDR) without touching the core each time. The proposed design (components, evidence layers, data model, the plugin contract and the planned tables) is in [design/multi-analysis.md](design/multi-analysis.md); today's system is in [architecture.md](architecture.md).

Built so far: plugin folders with auto-discovery, manifests checked against the database at startup, generic parameters through the API and worker, source versions and quality flags in record format v3, and the worker's code version and the manifest hash in record format v4. The image table and the frontend still assume NDVI.

### Build order

1. ✅ Plugin folders, auto-discovery and manifests; NDVI moved onto them with no change in behaviour.
2. ✅ Generic parameters, source versions, quality flags; record format v3. Code version and manifest hash sealed in record format v4.
3. ✅ `artifacts`, sealed in record format v5. Remaining: `series_points`, methodology-aware frontend views.
4. ✅ Second plugin, `chirps-rain-season`, end to end: seasonal rainfall from CHIRPS against the 1991–2020 normal. Each methodology now states where its data starts (`data_start`), and the frontend shows any methodology's measurements by their published names and units.
5. ✅ Third plugin, `landcover-dynamic-world`: the share of a parcel in crops, trees, grass and shrub, built-up, water and bare ground, from each pixel's most common Dynamic World class in the window. The first of the farm-lending datasets; soil, terrain and flooding follow.
   - ✅ `ndvi-anomaly-s2`: the window's NDVI against the same calendar window in up to five earlier years from 2019, measured exactly like NDVI. Only earlier years count, so a sealed result doesn't change later.
6. Prototype water history and rainfall-adjusted residuals on parcels with known ground truth. Built so far: combined runs (NDVI and rainfall for the same window, sealed as a group) and both on one parcel chart.
7. Analyses and batch exports, then reports.
