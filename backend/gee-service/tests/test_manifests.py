from dataclasses import replace

from app.contract import Manifest, Parameter
from app.manifests import differences
from app.plugins.ndvi_s2_median.manifest import MANIFEST

CLOUD, THRESHOLD = MANIFEST.parameters


def test_matching_parameters_have_no_differences():
    assert differences(MANIFEST, [THRESHOLD, CLOUD]) == []


def test_database_numbers_compare_equal_to_manifest_numbers():
    as_floats = [replace(p, min=float(p.min), max=float(p.max), default=float(p.default)) for p in MANIFEST.parameters]
    assert differences(MANIFEST, as_floats) == []


def test_changed_default_is_reported():
    problems = differences(MANIFEST, [replace(CLOUD, default=30), THRESHOLD])
    assert len(problems) == 1
    assert "max_scene_cloud_pct" in problems[0]


def test_parameter_on_one_side_only_is_reported():
    extra = Parameter("extra", "Extra", "number", 0, 1, 0)
    assert len(differences(MANIFEST, [CLOUD])) == 1
    assert len(differences(MANIFEST, [CLOUD, THRESHOLD, extra])) == 1


def test_unpublished_methodology_is_reported():
    problems = differences(Manifest("rain", "1.0.0", [], []), None)
    assert problems == ["rain@1.0.0 is not in methodologies; add a migration for it."]
