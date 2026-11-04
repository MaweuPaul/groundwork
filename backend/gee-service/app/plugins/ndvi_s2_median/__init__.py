from app.contract import Plugin
from app.plugins.ndvi_s2_median import compute, render

PLUGIN = Plugin(compute=compute.run, render=render.render)
