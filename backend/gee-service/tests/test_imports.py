import importlib

import pytest


@pytest.mark.parametrize("module", ["app.config", "app.earth_engine", "app.errors", "app.contract", "app.queue", "app.plugins.ndvi_s2_median", "app.plugins.chirps_rain_season", "app.plugins.landcover_dynamic_world", "app.windows", "app.registry", "app.worker"])
def test_module_imports(module):
    importlib.import_module(module)
