from app.contract import Plugin
from app.plugins.ndvi_s2_median import compute, render
from app.plugins.ndvi_s2_median.manifest import MANIFEST

PLUGIN = Plugin(manifest=MANIFEST, compute=compute.run, render=render.render)