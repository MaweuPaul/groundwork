from app.contract import Plugin
from app.plugins.ndvi_anomaly_s2 import compute
from app.plugins.ndvi_anomaly_s2.manifest import MANIFEST

PLUGIN = Plugin(manifest=MANIFEST, compute=compute.run)
