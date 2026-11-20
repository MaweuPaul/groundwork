from app.contract import Plugin
from app.plugins.chirps_rain_season import compute
from app.plugins.chirps_rain_season.manifest import MANIFEST

PLUGIN = Plugin(manifest=MANIFEST, compute=compute.run, latest_data=compute.latest_day)
