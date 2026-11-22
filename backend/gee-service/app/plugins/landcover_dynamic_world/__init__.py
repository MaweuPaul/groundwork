from app.contract import Plugin
from app.plugins.landcover_dynamic_world import compute
from app.plugins.landcover_dynamic_world.manifest import MANIFEST

PLUGIN = Plugin(manifest=MANIFEST, compute=compute.run)
