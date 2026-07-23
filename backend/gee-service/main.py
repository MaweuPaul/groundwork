import os
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
import ee

from models import SnapshotRequest, SnapshotResponse
from services import process_snapshot

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Initialize Earth Engine
    try:
        # ee.Initialize() will automatically look for GOOGLE_APPLICATION_CREDENTIALS environment variable
        logger.info("Initializing Google Earth Engine...")
        ee.Initialize()
        logger.info("Google Earth Engine initialized successfully.")
    except Exception as e:
        logger.error(f"Failed to initialize Earth Engine: {e}")
        logger.warning("Please ensure GOOGLE_APPLICATION_CREDENTIALS is set and the service account has GEE access.")
        # We don't raise here so the server can still start and show error endpoints if needed.
    
    yield
    # Shutdown
    logger.info("Shutting down GEE Service...")

app = FastAPI(title="Groundwork GEE Microservice", lifespan=lifespan)

@app.get("/health")
def health_check():
    return {"status": "healthy", "gee_initialized": ee.data.is_initialized()}

@app.post("/calculate-snapshot", response_model=SnapshotResponse)
def calculate_snapshot(request: SnapshotRequest):
    if not ee.data.is_initialized():
        raise HTTPException(status_code=500, detail="Earth Engine is not initialized. Check credentials.")
    
    try:
        response = process_snapshot(request)
        return response
    except ValueError as ve:
        raise HTTPException(status_code=400, detail=str(ve))
    except Exception as e:
        logger.error(f"GEE processing failed: {e}")
        raise HTTPException(status_code=500, detail=f"Earth Engine calculation failed: {e}")
