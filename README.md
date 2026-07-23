# Groundwork

Groundwork is an independent verification layer for real-world land asset claims. It replaces trust in paperwork with trust in reproducible satellite data by tracking physical land boundaries and calculating vegetation health indices (like NDVI) over time.

It ingests multi-source satellite imagery (e.g., Sentinel-2 via Google Earth Engine), computes spectral indices using versioned methodology, and serves auditable time-series data over a fast Go + PostGIS backend. 

## Core Concepts & Schema

Groundwork relies on a strict, auditable geospatial database using PostgreSQL and the PostGIS extension.

- **Parcels**: Physical land boundaries defined as standard GPS polygons (`GEOMETRY(POLYGON, 4326)`). The system automatically calculates the exact real-world acreage (in hectares) using PostGIS functions that factor in the curvature of the Earth.
- **Methodologies**: Immutable, version-controlled specifications that define *how* a satellite measurement is calculated (e.g., which satellite bands to use, cloud masking thresholds, and how to composite multiple images).
- **Snapshots**: A single measurement run. It acts as the bridge connecting a specific Parcel, a specific Methodology, and a specific Date Range.
- **Measurements**: The actual computed index values (e.g., the mean NDVI value of the parcel).
- **Imagery Sources**: A permanent audit trail logging the exact raw satellite imagery used to calculate a specific Snapshot.

## Project Structure

This repository is a monorepo containing both the Go backend and the frontend.

```text
/
├── frontend/                    # Next.js / React application
└── backend/                     # Go + PostGIS API
    ├── cmd/api/main.go          # Entry point for the Go API
    ├── internal/
    │   ├── config/              # Environment variables & DB connection config
    │   ├── models/              # GORM entities mapped to PostGIS tables
    │   ├── repository/          # DB operations
    │   ├── service/             # Business logic
    │   ├── handler/             # HTTP handlers
    │   └── database/            # Database connection
    ├── migrations/              # Core SQL scripts defining the PostGIS schema
    ├── gee-service/             # Python microservice for Google Earth Engine (WIP)
    ├── go.mod                   # Go module definitions
    └── .env.example             # Example environment variables
```

## Prerequisites

- **Go** (1.20+)
- **PostgreSQL** with the **PostGIS** extension enabled
- **Node.js** (for the frontend)
- **Python** (for the Google Earth Engine microservice)

## Setup Instructions

### 1. Database Setup
Ensure you have PostgreSQL running with PostGIS installed. Connect to your database and run the initial migration script found at `backend/migrations/001_initial_schema.sql` to generate the strict geospatial tables.

### 2. Backend API
Navigate to the `backend` directory, set up your `.env` file from the example, and run the server:
```bash
cd backend
cp .env.example .env
# Edit .env with your PostgreSQL DATABASE_URL
go mod tidy
go run cmd/api/main.go
```

### 3. Frontend
Navigate to the `frontend` directory, install dependencies, and run the development server:
```bash
cd frontend
npm install
npm run dev
```
