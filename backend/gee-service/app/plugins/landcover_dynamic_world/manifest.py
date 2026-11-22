from app.contract import Manifest, Parameter

COLLECTION = "GOOGLE/DYNAMICWORLD/V1"
BAND = "label"

# Dynamic World's resolution; classes are counted per pixel at it.
SCALE_M = 10

# Dynamic World's class index -> the share it counts toward. Changing this is a new version.
SHARE_OF_CLASS = {
    0: "water_pct",  # water
    1: "trees_pct",  # trees
    2: "grass_shrub_pct",  # grass
    3: "water_pct",  # flooded vegetation
    4: "crops_pct",  # crops
    5: "grass_shrub_pct",  # shrub and scrub
    6: "built_pct",  # built
    7: "bare_pct",  # bare
    8: "bare_pct",  # snow and ice
}

SHARES = ["crops_pct", "trees_pct", "grass_shrub_pct", "built_pct", "water_pct", "bare_pct"]

MANIFEST = Manifest(
    slug="landcover-dynamic-world",
    version="1.0.0",
    parameters=[
        Parameter("min_valid_pct", "Minimum classified pixels (%)", "number", 0, 100, 50),
    ],
    measurements=[*SHARES, "landcover_valid_pct"],
)
