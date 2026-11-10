from app.contract import Manifest, Parameter

MANIFEST = Manifest(
    slug="ndvi-s2-median",
    version="1.0.0",
    parameters=[
        Parameter("max_scene_cloud_pct", "Max scene cloud cover (%)", "number", 0.01, 100, 20),
        Parameter("cloud_score_threshold", "Clear pixel threshold", "number", 0.40, 0.80, 0.60),
    ],
    measurements=["ndvi_mean", "ndvi_max", "valid_pixel_pct"],
)
