import pytest

from app.errors import JobError
from app.registry import PLUGINS, plugin_for


def test_ndvi_is_registered():
    assert plugin_for("ndvi-s2-median").compute is not None


def test_chirps_is_registered():
    assert plugin_for("chirps-rain-season").compute is not None


def test_landcover_is_registered():
    assert plugin_for("landcover-dynamic-world").compute is not None


def test_unknown_methodology_fails_permanently():
    with pytest.raises(JobError) as err:
        plugin_for("water-history")
    assert err.value.code == "UNSUPPORTED_METHODOLOGY"
    assert not err.value.retryable


def test_every_plugin_has_a_compute_function():
    assert all(callable(p.compute) for p in PLUGINS.values())
