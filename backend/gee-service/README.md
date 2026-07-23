# Groundwork GEE Microservice

This is a lightweight Python microservice built with FastAPI that talks directly to the Google Earth Engine Python API. 

The Go backend communicates with this service via HTTP to calculate remote sensing metrics (like NDVI) over specific geometries.

## Requirements

1. Python 3.9+
2. A Google Cloud Service Account JSON key with the Earth Engine API enabled.

## Setup

1. Navigate to this directory:
   ```bash
   cd backend/gee-service
   ```

2. Create a virtual environment (recommended):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows use `venv\Scripts\activate`
   ```

3. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

## Running the Service

You **must** set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the absolute path of your Service Account JSON file before running the service.

**Windows (PowerShell):**
```powershell
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\path\to\your\credentials.json"
uvicorn main:app --reload --host 0.0.0.0 --port 8000
```

**macOS/Linux:**
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/credentials.json"
uvicorn main:app --reload --host 0.0.0.0 --port 8000
```

## API Documentation

Once the server is running, navigate to:
[http://localhost:8000/docs](http://localhost:8000/docs) to see the interactive Swagger UI and test the API directly!
