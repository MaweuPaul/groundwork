# How verification works

For auditors, buyers, lenders and anyone else asked to trust a Groundwork result. It explains what a sealed record is, how to check one, and what a passing check does and doesn't prove.

## A record is a fingerprinted statement

Each time Groundwork measures a parcel, the result is written down in a fixed text form when the measurement finishes, and the database computes a **SHA-256 fingerprint** (the *record hash*) of that text. Changing a single character, or a single digit of a single value, gives a completely different fingerprint.

A record (format v5) contains:

- **Where:** the parcel's boundary exactly as it was measured. The snapshot keeps its own frozen copy, so editing the parcel later never changes old records.
- **When:** the start and end dates of the period measured.
- **How:**
  - the methodology and its version;
  - the fingerprint of its published specification (the *manifest hash*);
  - the version of the worker code that computed it;
  - every setting used, including defaults.
- **From what:** every satellite image or daily dataset read, with the version its provider reported at the time.
- **The result:** every measured value, such as mean NDVI or rainfall as a percentage of normal, plus quality flags (for example, too few clear pixels) and the fingerprint of every file produced, such as the NDVI map image.
- **Sealing:** the exact time the record was sealed.
- **The link:** the fingerprint of the parcel's previous record.

Only finished measurements are sealed. A failed run leaves no record in the chain.

## Records form a chain

Each parcel has its own numbered chain: record #1, #2, #3 and so on. Every record includes the previous record's fingerprint, and the database remembers the chain's last position and fingerprint (the *head*). As a result:

- **Editing** a record changes its fingerprint, so it no longer matches its contents, and the next record no longer links to it.
- **Deleting** a record leaves a gap in the numbering and a broken link. Deleting the latest records leaves the head pointing past the end of the chain.
- **Inserting or reordering** records breaks the numbering and the links.

**Combined runs**, several methodologies measured for the same period (for example NDVI and rainfall for one season), are sealed once more as a set. That seal is one fingerprint over the period and every member's record, so an unfavourable member can't be dropped quietly later.

The database writes all of this itself, in triggers that run when a measurement finishes. Completed records, their values and their sources are frozen by further triggers. Neither the API nor the worker computes a fingerprint.

## How to check

On a parcel's page, the **Chain check** card has a **Recheck chain** button. It recomputes every record's fingerprint from what is stored now, follows the links from #1 to the head, and recomputes the seal of every combined run.

- **"All 12 records verify"**: nothing stored has changed since it was sealed, and no record is missing from the chain. The card also shows the head's position and fingerprint, which you can copy.
- **"N problems found"**: each problem names what failed. Click a record to open it.

| Problem | What it means |
|---|---|
| `record_hash does not match the stored contents` | Something in that record changed after it was sealed |
| `prev_record_hash does not match the parcel's previous record_hash` | The record before it was changed or removed, or records were reordered |
| `parcel_seq is X, expected Y (missing or extra record)` | A record is missing from the chain, or one was inserted |
| `parcel chain head does not match the parcel's last completed snapshot` | Records were removed from the end, or the head itself was altered |
| `group record_hash does not match its members` | A member of a combined run changed, or the run's seal was altered |

Each record's own page shows its number, fingerprint and the previous fingerprint under **Integrity**.

## What a passing check proves

**It proves:** every record is exactly as it was when sealed, and none has been removed from or added to the parcel's chain, according to the data stored now.

**It does not prove:**

- **That the measurement reflects the ground.** NDVI is a satellite estimate of vegetation, and CHIRPS rainfall describes an area of about 5.5 km around the parcel, not the parcel alone. The methodology and its sources are listed on every record so you can judge them.
- **That the boundary is the legal boundary.** Groundwork measures the boundary it was given and seals it. It doesn't check land titles.
- **That the database operator rewrote nothing.** The check runs on Groundwork's server against Groundwork's database. Someone with full control of that database could, in principle, rewrite an entire chain and recompute every fingerprint consistently, and the check would pass.

**Protecting yourself against that last case is simple: keep the head fingerprint.**
1. Copy it when you review a parcel, and put it in your report or an email.
2. Later, open that record number and compare its fingerprint with the one you kept.

If anything at or before that point was rewritten, the fingerprint won't match. You don't need to trust Groundwork for this; you only need the fingerprint you kept.

**Planned:** an audit export containing every record of a parcel, with a small standalone script that recomputes every fingerprint on your own computer, without the Groundwork server.

## For engineers: recomputing a record

- **Record text:** `snapshot_record_text(snapshot_id, completed_at, prev_record_hash)` in [migrations/016_artifacts.sql](../backend/go-api/migrations/016_artifacts.sql) defines the text, one `key=value` per line in a fixed order. Lists are sorted bytewise and numbers are rounded to 9 decimal places. The record hash is the lowercase hex SHA-256 of that text in UTF-8.
- **Older formats:** records keep the format they were sealed with (`record_format`), and the same function handles formats 1 to 5.
- **Group seals:** these are defined by `group_record_text` in [migrations/019_snapshot_groups.sql](../backend/go-api/migrations/019_snapshot_groups.sql).
- **Checks:**
  - `verify_parcel_chain(parcel_id)` checks one parcel's chain, and `verify_snapshot_group(group_id)` checks one group. `GET /parcels/{id}/verify` runs both, read-only.
  - `verify_snapshot_chain()` checks every parcel at once, for operators.
- **Records sealed before per-parcel chains** (migration 015) were linked in one global chain across all parcels. The parcel check still recomputes their contents, but their links are checked by `verify_snapshot_chain()`.
