from datetime import date
from uuid import uuid4

import pytest

from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.landcover_dynamic_world.compute import check_supported, summarize

METHODOLOGY = Methodology(
    slug="landcover-dynamic-world",
    version="1.0.0",
    index_type="landcover",
    satellite="sentinel-2",
    bands=["label"],
    cloud_mask=None,
    max_cloud_pct=20.0,
    composite_method="mode",
)

JOB = Job(
    job_id=uuid4(),
    snapshot_id=uuid4(),
    attempt=1,
    methodology_id=uuid4(),
    methodology_version="1.0.0",
    date_start=date(2025, 6, 1),
    date_end=date(2025, 8, 31),
    parameters={"min_valid_pct": 50},
    geometry_geojson="{}",
)

SCENES = [
    ["20250603T074621_20250603T075938_T37MBV", 1748937000000, 1749000000000000],
    ["20250618T074619_20250618T075420_T37MBV", 1750233000000, 1750300000000000],
]


def info(histogram, parcel_pixels=1000, scenes=SCENES):
    return {"histogram": histogram, "parcel_pixels": parcel_pixels, "scenes": scenes}


def test_shares_group_the_classes_and_add_up_to_100():
    # crops 600, trees 200, grass 50 + shrub 50, water 30 + flooded vegetation 20, built 30, bare 15 + snow 5
    result = summarize(
        info({"4": 600, "1": 200, "2": 50, "5": 50, "0": 30, "3": 20, "6": 30, "7": 15, "8": 5}), JOB
    )
    values = {slug: m.value for slug, m in result.measurements.items()}
    assert values["crops_pct"] == pytest.approx(60)
    assert values["trees_pct"] == pytest.approx(20)
    assert values["grass_shrub_pct"] == pytest.approx(10)
    assert values["water_pct"] == pytest.approx(5)
    assert values["built_pct"] == pytest.approx(3)
    assert values["bare_pct"] == pytest.approx(2)
    assert sum(v for slug, v in values.items() if slug != "landcover_valid_pct") == pytest.approx(100)
    assert values["landcover_valid_pct"] == pytest.approx(100)
    assert result.flags == {}


def test_shares_are_of_classified_pixels_and_absent_classes_are_zero():
    result = summarize(info({"4": 300, "1": 100}, parcel_pixels=1000), JOB)
    assert result.measurements["crops_pct"].value == pytest.approx(75)
    assert result.measurements["built_pct"].value == 0
    assert result.measurements["landcover_valid_pct"].value == pytest.approx(40)
    assert "low_coverage" in result.flags


def test_scenes_are_listed_with_their_versions():
    result = summarize(info({"4": 10}), JOB)
    assert [s.scene_id for s in result.scenes] == [row[0] for row in SCENES]
    assert result.scenes[0].asset_uri == f"GOOGLE/DYNAMICWORLD/V1/{SCENES[0][0]}"
    assert result.scenes[0].source_version == "1749000000000000"


def test_no_images_fails_permanently():
    with pytest.raises(JobError) as err:
        summarize(info({}, scenes=[]), JOB)
    assert err.value.code == "NO_IMAGERY"
    assert not err.value.retryable


def test_no_classified_pixels_fails_permanently():
    with pytest.raises(JobError) as err:
        summarize(info(None), JOB)
    assert err.value.code == "NO_CLEAR_PIXELS"


def test_an_unknown_class_fails_instead_of_being_dropped():
    with pytest.raises(JobError) as err:
        summarize(info({"4": 10, "9": 1}), JOB)
    assert err.value.code == "UNEXPECTED_DATA"


def test_only_this_methodology_is_supported():
    check_supported(METHODOLOGY)
    with pytest.raises(JobError) as err:
        check_supported(Methodology(**{**METHODOLOGY.__dict__, "composite_method": "median"}))
    assert err.value.code == "UNSUPPORTED_METHODOLOGY"
