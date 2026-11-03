import importlib

import pytest


@pytest.mark.parametrize("module", ["app.config", "app.earth_engine", "app.errors", "app.compute", "app.queue", "app.render", "app.images", "app.worker"])
def test_module_imports(module):
    importlib.import_module(module)
