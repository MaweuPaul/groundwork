from app.contract import Manifest
from app.plugins.ndvi_s2_median.manifest import MANIFEST as NDVI_MANIFEST

# Sentinel-2 surface reflectance covers the world from 2019, so baselines start there.
BASELINE_FIRST_YEAR = 2019
BASELINE_MAX_YEARS = 5
BASELINE_MIN_YEARS = 3

MANIFEST = Manifest(
    slug="ndvi-anomaly-s2",
    version="1.0.0",
    # The same cloud settings as NDVI: every window is measured exactly like an NDVI snapshot.
    parameters=NDVI_MANIFEST.parameters,
    measurements=["ndvi_anomaly", "ndvi_mean", "ndvi_baseline_mean", "ndvi_baseline_years", "valid_pixel_pct"],
)
