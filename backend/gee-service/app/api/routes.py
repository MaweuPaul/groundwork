import logging
from fastapi import APIRouter, HTTPException
import ee

from app.schemas.snapshot import SnapshotRequest, SnapshotResponse
from app.services.snapshot import process_snapshot

logger = logging.getLogger(__name__)

router = APIRouter()

@router.get("/health")
def health_check():
    return {"status": "healthy", "gee_initialized": ee.data.is_initialized()}

@router.post("/calculate-snapshot", response_model=SnapshotResponse)
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
