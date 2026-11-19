from dataclasses import replace
from datetime import date, datetime, timezone
from uuid import uuid4

import pytest

from app.contract import Artifact
from app.plugins.ndvi_s2_median import compute
from app.plugins.ndvi_s2_median.compute import check_supported, end_exclusive, summarize
from app.errors import JobError
from app.models import Job, Methodology

METHODOLOGY = Methodology(
    slug="ndvi-s2-median",
    version="1.0.0",
    index_type="ndvi",
    satellite="sentinel-2",
    bands=["B4", "B8"],
    cloud_mask="cloud_score_plus",
    max_cloud_pct=20.0,
    composite_method="median",
)

JOB = Job(
    job_id=uuid4(),
    snapshot_id=uuid4(),
    attempt=1,
    methodology_id=uuid4(),
    methodology_version="1.0.0",
    date_start=date(2025, 4, 1),
    date_end=date(2025, 4, 30),
    parameters={"max_scene_cloud_pct": 60.0, "cloud_score_threshold": 0.6, "min_clear_pct": 50},
    geometry_geojson="{}",
)

SCENE = ["20250412T073619_20250412T074502_T37MBU", 1744443379000, 1.4, "Sentinel-2B", "S2B_MSIL2A_20250412", 1744460000123456]


def info(**overrides):
    base = {
        "stats": {"ndvi_mean": 0.742, "ndvi_max": 0.913, "ndvi_count": 900},
        "parcel_pixels": 1000,
        "scenes": [SCENE],
    }
    return {**base, **overrides}


def test_summarize_builds_measurements_and_scenes():
    result = summarize(info(), JOB)

    assert result.measurements["ndvi_mean"].value == 0.742
    assert result.measurements["ndvi_max"].value == 0.913
    assert result.measurements["valid_pixel_pct"].value == pytest.approx(90.0)
    assert result.measurements["valid_pixel_pct"].metadata["clear_pixels"] == 900
    assert result.measurements["valid_pixel_pct"].metadata["cloud_score_plus_threshold"] == 0.6

    (scene,) = result.scenes
    assert scene.provider == "Copernicus Sentinel-2 via Google Earth Engine"
    assert scene.processing_level == "L2A"
    assert scene.asset_uri == f"COPERNICUS/S2_SR_HARMONIZED/{scene.scene_id}"
    assert scene.scene_id == SCENE[0]
    assert scene.satellite == "sentinel-2b"
    assert scene.acquisition_time == datetime(2025, 4, 12, 7, 36, 19, tzinfo=timezone.utc)
    assert scene.cloud_cover_pct == 1.4
    assert scene.source_version == "1744460000123456"
    assert result.flags == {}


def test_low_coverage_is_flagged_not_failed():
    result = summarize(info(stats={"ndvi_mean": 0.5, "ndvi_max": 0.6, "ndvi_count": 420}), JOB)
    assert result.measurements["valid_pixel_pct"].value == pytest.approx(42.0)
    assert list(result.flags) == ["low_coverage"]
    assert "42.0%" in result.flags["low_coverage"]


def test_snapshot_from_before_min_clear_pct_uses_the_manifest_default():
    older = replace(JOB, parameters={"max_scene_cloud_pct": 60.0, "cloud_score_threshold": 0.6})
    result = summarize(info(stats={"ndvi_mean": 0.5, "ndvi_max": 0.6, "ndvi_count": 420}), older)
    assert "low_coverage" in result.flags


def test_no_scenes_is_a_terminal_error():
    with pytest.raises(JobError) as err:
        summarize(info(scenes=[]), JOB)
    assert err.value.code == "NO_IMAGERY"
    assert "60%" in err.value.message
    assert not err.value.retryable


@pytest.mark.parametrize(
    "stats",
    [
        {"ndvi_mean": None, "ndvi_max": None, "ndvi_count": 0},
        {},
    ],
)
def test_no_clear_pixels_is_an_error_not_zero(stats):
    with pytest.raises(JobError) as err:
        summarize(info(stats=stats), JOB)
    assert err.value.code == "NO_CLEAR_PIXELS"
    assert not err.value.retryable


def test_valid_pixel_pct_is_capped_at_100():
    # The composite and the parcel mask can disagree by a pixel or two on the edge.
    result = summarize(info(stats={"ndvi_mean": 0.5, "ndvi_max": 0.6, "ndvi_count": 1003}), JOB)
    assert result.measurements["valid_pixel_pct"].value == 100.0


def test_end_date_is_inclusive():
    assert end_exclusive(date(2025, 4, 30)) == "2025-05-01"
    assert end_exclusive(date(2024, 12, 31)) == "2025-01-01"


def test_supported_methodology_passes():
    check_supported(METHODOLOGY)
    check_supported(replace(METHODOLOGY, bands=["B8", "B4"]))


@pytest.mark.parametrize(
    "change",
    [
        {"index_type": "evi"},
        {"satellite": "landsat-9"},
        {"cloud_mask": "qa60"},
        {"composite_method": "mean"},
        {"bands": ["B4", "B8", "B2"]},
    ],
)
def test_unsupported_methodology_is_terminal(change):
    with pytest.raises(JobError) as err:
        check_supported(replace(METHODOLOGY, **change))
    assert err.value.code == "UNSUPPORTED_METHODOLOGY"
    assert not err.value.retryable


def test_map_image_is_added_as_an_artifact(monkeypatch):
    image = Artifact(key="ndvi_map", media_type="image/png", content=b"png", metadata={"width": 1})
    monkeypatch.setattr(compute.render, "map_image", lambda geometry, geojson, composite: image)
    result = compute.with_map_image(summarize(info(), JOB), None, "{}", None)
    assert result.artifacts == [image]
    assert "map_image_failed" not in result.flags


def test_a_failed_map_image_is_flagged_and_the_snapshot_still_completes(monkeypatch):
    def fail(geometry, geojson, composite):
        raise RuntimeError("computePixels: too many pixels")

    monkeypatch.setattr(compute.render, "map_image", fail)
    measured = summarize(info(), JOB)
    result = compute.with_map_image(measured, None, "{}", None)
    assert result.artifacts == []
    assert result.measurements == measured.measurements
    assert result.flags["map_image_failed"] == "computePixels: too many pixels"


def test_float_versions_are_sealed_as_integers():
    scene = [*SCENE[:5], float(SCENE[5])]
    (s,) = summarize(info(scenes=[scene]), JOB).scenes
    assert s.source_version == str(SCENE[5])
