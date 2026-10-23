import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI
import ee

from app.api.routes import router

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
app.include_router(router)
