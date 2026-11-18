from app.contract import Manifest, Parameter

COLLECTION = "UCSB-CHG/CHIRPS/DAILY"
BAND = "precipitation"

# WMO standard climate normal. Not in the sealed manifest hash, so a different baseline is a new version.
BASELINE_START_YEAR = 1991
BASELINE_END_YEAR = 2020

# CHIRPS' native resolution (0.05 degrees): values describe the area around the parcel, not the field.
NATIVE_SCALE_M = 5566

# Reducing at the native scale gives no value for parcels that contain no pixel centre; at this
# scale every parcel gets a value. Changing it can shift results, so it is part of the version.
REDUCE_SCALE_M = 100

MANIFEST = Manifest(
    slug="chirps-rain-season",
    version="1.0.0",
    parameters=[
        Parameter("rainy_day_mm", "Rainy day threshold (mm)", "number", 0.1, 20, 1),
    ],
    measurements=[
        "rain_total_mm",
        "rain_normal_mm",
        "rain_pct_of_normal",
        "rainy_days",
        "longest_dry_spell_days",
    ],
)
