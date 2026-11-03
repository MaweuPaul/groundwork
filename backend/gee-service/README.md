# Groundwork GEE worker

Computes Groundwork's satellite measurements (NDVI) with the Google Earth Engine Python API.

It does not talk to the Go API, and the Go API does not call it. PostgreSQL is the only link: the Go API queues snapshot jobs in the database; this worker claims them, computes the results with Earth Engine, and saves measurements and imagery sources back. See "Go ↔ Python contract" in the [main README](../../README.md).

## How a job runs

1. `claim_snapshot_job` hands the worker the next due job and marks the snapshot `processing`.
2. `ndvi-s2-median@1.0.0`: Sentinel-2 surface reflectance scenes in the window, under the methodology's scene cloud limit, are masked per pixel with Cloud Score+ (`cs_cdf >= 0.60`), turned into NDVI and median-composited at 10 m. The parcel's mean and max NDVI and its share of clear pixels are measured.
3. Measurements, every scene used, and `complete_snapshot_job` are saved in **one transaction**, which seals the snapshot into the hash chain. If anything fails, nothing is saved.
4. Failures go to `fail_snapshot_job`. Quotas, timeouts and network errors are retried with backoff; no imagery, no clear pixels, or an unsupported methodology fail the snapshot at once. Missing data is never reported as `0`.

The worker wakes on `NOTIFY snapshot_jobs` and also polls every 30 s. Several workers can run at once.

## Setup

1. Python 3.12, from this directory:
   ```powershell
   python -m venv venv
   venv\Scripts\activate
   pip install -r requirements.txt
   ```
2. Earth Engine signs in with a **service account** (an API key won't work). In Google Cloud:
   - register the project for Earth Engine and enable the Google Earth Engine API;
   - create a service account with **Earth Engine Resource Viewer** and **Service Usage Consumer**;
   - create a JSON key and save it as `credentials/earth-engine-key.json` (git-ignored).
3. Give the database role a password (it's created without one by migration 006):
   ```sql
   ALTER ROLE gee_worker PASSWORD 'choose-one';
   ```
4. Copy `.env.example` to `.env` and fill in `GEE_PROJECT` and `DATABASE_URL`.

## Run

```powershell
python -m app.worker
```

It signs in to Earth Engine first and exits with a clear message if the key, project or settings are wrong.

## Tests

```powershell
python -m pytest
```

The tests cover turning Earth Engine's response into measurements, error classification and settings; they don't call Earth Engine.
