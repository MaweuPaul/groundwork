from dataclasses import replace

from app.contract import Manifest, Parameter
from app.manifests import differences, measurement_differences
from app.plugins.ndvi_s2_median.manifest import MANIFEST

CLOUD, THRESHOLD, MIN_CLEAR = MANIFEST.parameters


def test_matching_parameters_have_no_differences():
    assert differences(MANIFEST, [MIN_CLEAR, THRESHOLD, CLOUD]) == []


def test_database_numbers_compare_equal_to_manifest_numbers():
    as_floats = [replace(p, min=float(p.min), max=float(p.max), default=float(p.default)) for p in MANIFEST.parameters]
    assert differences(MANIFEST, as_floats) == []


def test_changed_default_is_reported():
    problems = differences(MANIFEST, [replace(CLOUD, default=30), THRESHOLD, MIN_CLEAR])
    assert len(problems) == 1
    assert "max_scene_cloud_pct" in problems[0]


def test_parameter_on_one_side_only_is_reported():
    extra = Parameter("extra", "Extra", "number", 0, 1, 0)
    assert len(differences(MANIFEST, [CLOUD, THRESHOLD])) == 1
    assert len(differences(MANIFEST, [CLOUD, THRESHOLD, MIN_CLEAR, extra])) == 1


def test_unpublished_methodology_is_reported():
    problems = differences(Manifest("rain", "1.0.0", [], []), None)
    assert problems == ["rain@1.0.0 is not in methodologies; add a migration for it."]


def test_matching_measurements_have_no_differences():
    assert measurement_differences(MANIFEST, ["ndvi_mean", "ndvi_max", "valid_pixel_pct"]) == []


def test_missing_or_reordered_measurements_are_reported():
    assert len(measurement_differences(MANIFEST, ["ndvi_mean", "ndvi_max"])) == 1
    assert len(measurement_differences(MANIFEST, ["ndvi_max", "ndvi_mean", "valid_pixel_pct"])) == 1
