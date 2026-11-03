# Groundwork architecture

Diagrams are [Mermaid](https://mermaid.js.org/). GitHub renders them; in VS Code install
**Markdown Preview Mermaid Support** (`bierner.markdown-mermaid`) and open the Markdown preview.

## How the parts talk to each other

The Go API and the Python worker never call each other. PostgreSQL is the only link between
them, so either can be down or redeployed without breaking the other.

```mermaid
flowchart LR
    Browser["Browser<br/>(map, pages)"]

    subgraph Frontend["Next.js frontend :3000"]
        Pages["Server components<br/>and server actions"]
        Proxy["proxy.ts<br/>refreshes the session"]
        ImgRoute["/api/snapshots/[ref]/ndvi<br/>adds the bearer token"]
    end

    subgraph API["Go API :8081"]
        Handlers["Handlers → services → repositories"]
    end

    subgraph DB["PostgreSQL + PostGIS"]
        Tables[("parcels, snapshots,<br/>measurements, imagery,<br/>series, NDVI images")]
        Queue[("snapshot_jobs<br/>queue")]
        Chain["Triggers: hash chain,<br/>series seal, freeze"]
    end

    subgraph Worker["Python worker"]
        Loop["Claim → compute → save"]
        Registry["Methodology registry"]
    end

    EE["Google Earth Engine<br/>(Sentinel-2, Cloud Score+)"]
    Tiles["Esri / OSM<br/>map tiles"]

    Browser -- "pages, forms" --> Proxy --> Pages
    Browser -- "NDVI image" --> ImgRoute
    Browser -- "basemap tiles" --> Tiles
    Pages -- "REST + JWT" --> Handlers
    ImgRoute -- "REST + JWT" --> Handlers
    Handlers -- "SQL" --> Tables
    Handlers -- "insert job" --> Queue
    Queue -- "LISTEN/NOTIFY" --> Loop
    Loop -- "queue functions,<br/>results (gee_worker role)" --> DB
    Loop --> Registry -- "compute, render" --> EE
    Tables --- Chain
```

## Life of a snapshot

```mermaid
sequenceDiagram
    actor U as User
    participant FE as Next.js
    participant API as Go API
    participant DB as PostgreSQL
    participant W as Worker
    participant EE as Earth Engine

    U->>FE: Run snapshot (window, cloud settings)
    FE->>API: POST /parcels/{id}/snapshots
    API->>DB: insert snapshot (boundary frozen) + job, one transaction
    API-->>FE: 202 queued
    DB-->>W: NOTIFY snapshot_jobs
    W->>DB: claim_snapshot_job() → status processing
    W->>EE: composite, reduce over parcel
    EE-->>W: NDVI mean/max, clear pixels, scenes
    W->>DB: measurements + imagery + complete_snapshot_job(), one transaction
    Note over DB: trigger seals the record hash into the chain<br/>(and the series, if it was its last window)
    W->>EE: render the same composite as a PNG
    W->>DB: store the NDVI image
    FE->>API: page refreshes itself while work is pending
    API-->>FE: completed, record hash, image bounds
```

Failures go through `fail_snapshot_job()`: quota and network errors retry with backoff;
missing data or requests too big for Earth Engine fail at once with a readable reason.

## Adding a new analysis (methodology)

A methodology is a versioned, immutable recipe. Everything about a snapshot (inputs,
measurements, scenes) is sealed into its record hash, so a methodology never changes once
used: a change is a new version.

```mermaid
flowchart TB
    subgraph Built["Built"]
        M["Migration: seed the methodology row<br/>and its measurement types"]
        P["Worker: a module with compute()<br/>and optionally render()"]
        R["Worker: register it in app/registry.py"]
    end
    subgraph Planned["Planned, once the first new methodology is chosen"]
        Params["Generic parameters: each methodology declares its inputs<br/>(type, range, default, help); API validates them,<br/>the dialog renders them, record format v3 seals them"]
        Outputs["Richer outputs: time series and artifacts<br/>(images, event polygons) hashed into the record"]
        Prov["Dataset provenance: collection IDs and versions<br/>sealed with each record"]
        Analyses["Analyses: sealed records built over other sealed<br/>records (series is the first; RESTREND next)"]
        Batch["Batch exports for jobs over<br/>Earth Engine's 5-minute interactive limit"]
    end
    M --> P --> R
    R -.-> Params & Outputs & Prov
    Params -.-> Analyses
    Outputs -.-> Batch
```

Today the cloud settings are NDVI-specific columns on `snapshots`; under the planned design
they become NDVI's declared parameters. Candidate methodologies from the agricultural
assessment idea: `chirps-rain-season`, `s2-phenology`, `water-history`, `drainage-baseline`,
`eudr-forest-check`.

## Sealed records at a glance

| Record | Sealed when | Hash covers |
|---|---|---|
| Snapshot (format v2) | It completes | Inputs incl. cloud settings, frozen boundary, every measurement and scene, previous record hash |
| Series (format v1) | Its last window completes or fails | Series settings and each window's record hash (or "failed"), in date order |

`verify_snapshot_chain()` recomputes every snapshot hash; `series_record_hash()` recomputes a series.
