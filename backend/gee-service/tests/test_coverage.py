from datetime import date

from app import coverage
from app.contract import Manifest, Plugin


class FakeConn:
    def __init__(self):
        self.calls = []

    def execute(self, sql, params):
        self.calls.append(params)


def plugin(slug, latest_data=None):
    return Plugin(manifest=Manifest(slug, "1.0.0", [], []), compute=lambda job, m: None, latest_data=latest_data)


def test_records_the_latest_day_of_lagging_sources_only():
    conn = FakeConn()
    coverage.refresh(conn, {"rain": plugin("rain", lambda: date(2026, 9, 30)), "ndvi": plugin("ndvi")})
    assert conn.calls == [(date(2026, 9, 30), "rain")]


def test_a_failed_check_is_skipped_and_the_rest_still_run():
    def unreachable():
        raise RuntimeError("Earth Engine is down")

    conn = FakeConn()
    coverage.refresh(conn, {"a": plugin("a", unreachable), "b": plugin("b", lambda: date(2026, 10, 31))})
    assert conn.calls == [(date(2026, 10, 31), "b")]
