from datetime import date
from uuid import uuid4

import pytest

from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.ndvi_anomaly_s2.compute import baseline_windows, check_supported, summarize

METHODOLOGY = Methodology(
    slug="ndvi-anomaly-s2",
    version="1.0.0",
    index_type="ndvi_anomaly",
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
    date_start=date(2026, 3, 1),
    date_end=date(2026, 5, 31),
    parameters={"max_scene_cloud_pct": 20, "cloud_score_threshold": 0.6, "min_clear_pct": 50},
    geometry_geojson="{}",
)


def scene(scene_id):
    return [scene_id, 1772323200000, 5.0, "Sentinel-2A", f"P_{scene_id}", 1]


def window(mean, count, scene_id):
    return {"stats": {"ndvi_mean": mean, "ndvi_count": count}, "scenes": [scene(scene_id)]}


def info(current, baseline, parcel_pixels=100):
    return {"parcel_pixels": parcel_pixels, "window": current, "baseline": baseline}


def test_baseline_windows_go_back_five_years_but_not_before_2019():
    assert baseline_windows(date(2026, 3, 1), date(2026, 5, 31))[0] == (date(2025, 3, 1), date(2025, 5, 31))
    assert [s.year for s, _ in baseline_windows(date(2026, 3, 1), date(2026, 5, 31))] == [2025, 2024, 2023, 2022, 2021]
    assert [s.year for s, _ in baseline_windows(date(2022, 3, 1), date(2022, 5, 31))] == [2021, 2020, 2019]


def test_a_window_across_new_year_keeps_its_shape():
    assert baseline_windows(date(2025, 11, 1), date(2026, 2, 28))[0] == (date(2024, 11, 1), date(2025, 2, 28))


def test_anomaly_is_the_window_minus_the_mean_of_the_counted_years():
    result = summarize(
        info(
            window(0.70, 90, "now"),
            [window(0.60, 80, "y1"), window(0.50, 90, "y2"), window(0.40, 20, "y3"), window(0.55, 70, "y4")],
        ),
        JOB,
    )
    values = {slug: m.value for slug, m in result.measurements.items()}
    assert values["ndvi_mean"] == pytest.approx(0.70)
    assert values["ndvi_baseline_mean"] == pytest.approx(0.55)  # 2023 was 20% clear, left out
    assert values["ndvi_anomaly"] == pytest.approx(0.15)
    assert values["ndvi_baseline_years"] == 3
    assert values["valid_pixel_pct"] == pytest.approx(90)
    meta = result.measurements["ndvi_baseline_mean"].metadata
    assert meta["years"] == {"2025": 0.60, "2024": 0.50, "2022": 0.55}
    assert meta["skipped_clear_pct"] == {"2023": 20.0}
    assert [s.scene_id for s in result.scenes] == ["now", "y1", "y2", "y3", "y4"]
    assert result.flags == {}


def test_too_few_clear_years_fails_permanently():
    with pytest.raises(JobError) as err:
        summarize(info(window(0.7, 90, "now"), [window(0.6, 80, "a"), window(0.5, 10, "b"), window(None, 0, "c")]), JOB)
    assert err.value.code == "INSUFFICIENT_BASELINE"
    assert not err.value.retryable


def test_a_cloudy_window_still_completes_flagged():
    years = [window(0.6, 80, f"y{i}") for i in range(3)]
    result = summarize(info(window(0.7, 30, "now"), years), JOB)
    assert "low_coverage" in result.flags


def test_no_scenes_and_no_clear_pixels_fail_like_ndvi():
    with pytest.raises(JobError) as err:
        summarize(info({"stats": {}, "scenes": []}, []), JOB)
    assert err.value.code == "NO_IMAGERY"
    with pytest.raises(JobError) as err:
        summarize(info(window(None, 0, "now"), []), JOB)
    assert err.value.code == "NO_CLEAR_PIXELS"


def test_a_scene_shared_by_two_windows_is_listed_once():
    years = [window(0.6, 80, "same"), window(0.6, 80, "b"), window(0.6, 80, "c")]
    result = summarize(info(window(0.7, 90, "same"), years), JOB)
    assert [s.scene_id for s in result.scenes] == ["same", "b", "c"]


def test_only_this_methodology_is_supported():
    check_supported(METHODOLOGY)
    with pytest.raises(JobError):
        check_supported(Methodology(**{**METHODOLOGY.__dict__, "index_type": "ndvi"}))
