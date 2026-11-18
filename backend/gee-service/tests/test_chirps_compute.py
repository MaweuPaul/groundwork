from datetime import date
from uuid import uuid4

import pytest

from app.errors import JobError
from app.models import Job, Methodology
from app.plugins.chirps_rain_season.compute import (
    baseline_windows,
    check_supported,
    longest_run_below,
    summarize,
)

METHODOLOGY = Methodology(
    slug="chirps-rain-season",
    version="1.0.0",
    index_type="rainfall",
    satellite="chirps",
    bands=["precipitation"],
    cloud_mask=None,
    max_cloud_pct=20.0,
    composite_method="sum",
)

JOB = Job(
    job_id=uuid4(),
    snapshot_id=uuid4(),
    attempt=1,
    methodology_id=uuid4(),
    methodology_version="1.0.0",
    date_start=date(2025, 3, 1),
    date_end=date(2025, 3, 3),
    parameters={"rainy_day_mm": 1.0},
    geometry_geojson="{}",
)

MARCH_1 = 1740787200000  # 2025-03-01T00:00:00Z
DAY_MS = 86_400_000


def day(n: int, mm: float | None) -> list:
    return [f"2025030{n}", MARCH_1 + (n - 1) * DAY_MS, 1700000000000000, mm]


def info(days=None, baseline=None):
    return {
        "days": days if days is not None else [day(1, 0.0), day(2, 5.0), day(3, 1.0)],
        "baseline_totals": baseline if baseline is not None else [12.0] * 30,
    }


def test_summarize_builds_measurements_and_sources():
    result = summarize(info(), JOB)

    m = {slug: x.value for slug, x in result.measurements.items()}
    assert m["rain_total_mm"] == 6.0
    assert m["rain_normal_mm"] == 12.0
    assert m["rain_pct_of_normal"] == pytest.approx(50.0)
    assert m["rainy_days"] == 2  # 1.0 mm equals the threshold, so it counts as rainy
    assert m["longest_dry_spell_days"] == 1
    assert result.measurements["rain_total_mm"].metadata["native_scale_m"] == 5566
    assert result.flags == {}

    assert [s.scene_id for s in result.scenes] == ["20250301", "20250302", "20250303"]
    first = result.scenes[0]
    assert first.asset_uri == "UCSB-CHG/CHIRPS/DAILY/20250301"
    assert first.source_version == "1700000000000000"
    assert first.cloud_cover_pct is None
    assert first.acquisition_time.date() == date(2025, 3, 1)


def test_days_out_of_order_are_sorted_before_counting_dry_spells():
    days = [day(3, 0.0), day(1, 0.0), day(2, 5.0)]
    assert summarize(info(days=days), JOB).measurements["longest_dry_spell_days"].value == 1


def test_zero_normal_is_flagged_without_a_percentage():
    result = summarize(info(baseline=[0.0] * 30), JOB)
    assert "rain_pct_of_normal" not in result.measurements
    assert "zero_normal" in result.flags


@pytest.mark.parametrize(
    "bad_info, text",
    [
        (info(days=[]), "No CHIRPS data in this window"),
        (info(days=[day(1, 0.0), day(2, 5.0)]), "2 of the 3 days"),
        (info(days=[day(1, 0.0), day(2, None), day(3, 1.0)]), "over this parcel"),
        (info(baseline=[12.0] * 29 + [None]), "over this parcel"),
        (info(baseline=[12.0] * 29), "29 of the 30 baseline years"),
    ],
)
def test_missing_data_fails_instead_of_reporting_a_value(bad_info, text):
    with pytest.raises(JobError) as err:
        summarize(bad_info, JOB)
    assert err.value.code == "NO_IMAGERY"
    assert not err.value.retryable
    assert text in err.value.message


def test_default_threshold_is_used_when_the_parameter_is_missing():
    job = Job(**{**JOB.__dict__, "parameters": {}})
    assert summarize(info(), job).measurements["rainy_days"].metadata["rainy_day_mm"] == 1


def test_baseline_windows_cover_every_baseline_year():
    windows = baseline_windows(date(2025, 3, 1), date(2025, 5, 31))
    assert len(windows) == 30
    assert windows[0] == (date(1991, 3, 1), date(1991, 5, 31))
    assert windows[-1] == (date(2020, 3, 1), date(2020, 5, 31))


def test_baseline_window_crossing_a_year_end_keeps_its_shape():
    windows = baseline_windows(date(2024, 11, 1), date(2025, 2, 28))
    assert windows[0] == (date(1991, 11, 1), date(1992, 2, 28))


def test_baseline_window_on_29_february_falls_back_to_the_28th():
    windows = baseline_windows(date(2024, 2, 29), date(2024, 3, 31))
    assert windows[0] == (date(1991, 2, 28), date(1991, 3, 31))
    assert windows[1] == (date(1992, 2, 29), date(1992, 3, 31))


@pytest.mark.parametrize(
    "values, expected",
    [
        ([], 0),
        ([0.0, 0.0, 0.0], 3),
        ([2.0, 3.0], 0),
        ([0.0, 2.0, 0.0, 0.0], 2),
        ([2.0, 0.0, 0.5, 0.9], 3),
    ],
)
def test_longest_run_below(values, expected):
    assert longest_run_below(values, 1.0) == expected


def test_check_supported():
    check_supported(METHODOLOGY)
    with pytest.raises(JobError) as err:
        check_supported(Methodology(**{**METHODOLOGY.__dict__, "composite_method": "median"}))
    assert err.value.code == "UNSUPPORTED_METHODOLOGY"
