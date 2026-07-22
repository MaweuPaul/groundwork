"""
Google Earth Engine (GEE) Service
"""

import os
import ee

def init_gee():
    # Initialize the Earth Engine module.
    # Assumes authentication is already configured or using service account.
    try:
        ee.Initialize()
        print("Earth Engine initialized successfully.")
    except Exception as e:
        print(f"Failed to initialize Earth Engine: {e}")

if __name__ == '__main__':
    init_gee()
    # TODO: Add API endpoint (e.g. FastAPI/Flask) or CLI to trigger GEE calculations
