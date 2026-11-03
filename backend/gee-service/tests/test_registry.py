import pytest

from app.errors import JobError
from app.registry import PLUGINS, plugin_for


def test_ndvi_is_registered_with_a_map_image():
    plugin = plugin_for("ndvi-s2-median")
    assert plugin.compute is not None
    assert plugin.render is not None


def test_unknown_methodology_fails_permanently():
    with pytest.raises(JobError) as err:
        plugin_for("water-history")
    assert err.value.code == "UNSUPPORTED_METHODOLOGY"
    assert not err.value.retryable


def test_every_plugin_has_a_compute_function():
    assert all(callable(p.compute) for p in PLUGINS.values())
