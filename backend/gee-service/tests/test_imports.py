import importlib

import pytest


@pytest.mark.parametrize("module", ["app.config", "app.earth_engine", "app.errors", "app.contract", "app.queue", "app.plugins.ndvi_s2_median", "app.images", "app.registry", "app.worker"])
def test_module_imports(module):
    importlib.import_module(module)
