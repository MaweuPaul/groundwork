# Groundwork

Groundwork is an independent verification layer for real-world land asset claims. It ingests multi-source satellite imagery (e.g., Sentinel-2 via Google Earth Engine), computes spectral indices like NDVI using versioned methodology, and serves auditable time-series data over a fast Go + PostGIS backend. This ensures claims about physical land conditions are backed by reproducible data.

## Project Structure

This repository is set up with the backend service structured as follows:

```text
backend/
├── cmd/
│   └── api/
│       └── main.go              # Entry point for the Go API
├── internal/
│   ├── config/                  # Environment variables & DB connection config
│   ├── models/                  # GORM entities (Parcel, Methodology, Snapshot, etc.)
│   ├── repository/              # DB operations
│   ├── service/                 # Business logic
│   ├── handler/                 # HTTP handlers
│   └── database/                # GORM connection + AutoMigrate
├── migrations/                  # SQL scripts (e.g., PostGIS extension setup)
├── gee-service/                 # Python microservice for Google Earth Engine (WIP)
├── go.mod                       # Go module definitions
└── .env.example                 # Example environment variables
```

## Prerequisites

- **Go** (1.20+)
- **PostgreSQL** with the **PostGIS** extension
- **Python** (for the Google Earth Engine microservice)

## Setup Instructions

1. **Database Setup**
   Ensure you have PostgreSQL running with PostGIS installed. Connect to your database and run the initial migration:
   ```sql
   CREATE EXTENSION IF NOT EXISTS postgis;
   ```
   *(This is also included in `backend/migrations/001_initial_schema.sql`)*

2. **Environment Variables**
   Navigate to the `backend` directory and copy the `.env.example` file:
   ```bash
   cd backend
   cp .env.example .env
   ```
   Update the `DATABASE_URL` inside `.env` to match your local PostgreSQL credentials.

3. **Run the Go Server**
   From the `backend` directory, download dependencies and start the API:
   ```bash
   go mod tidy
   go run cmd/api/main.go
   ```
   The server will auto-migrate the database tables via GORM on startup and run on the configured `PORT` (default is 8080).
