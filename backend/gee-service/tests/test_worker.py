from datetime import date
from types import SimpleNamespace
from uuid import uuid4

from app import images, queue, registry, worker
from app.contract import Manifest, Measurement, Plugin, Result
from app.models import Job

SETTINGS = SimpleNamespace(worker_id="test")


def job() -> Job:
    return Job(
        job_id=uuid4(),
        snapshot_id=uuid4(),
        attempt=1,
        methodology_id=uuid4(),
        methodology_version="1.0.0",
        date_start=date(2026, 1, 1),
        date_end=date(2026, 3, 31),
        max_scene_cloud_pct=60,
        cloud_score_threshold=0.6,
        geometry_geojson="{}",
    )


def test_process_handles_a_plugin_with_its_own_measurements(monkeypatch):
    result = Result(measurements={"rain_total_mm": Measurement(412.5)}, scenes=[])
    saved = []
    manifest = Manifest(slug="rain", version="1.0.0", parameters=[], measurements=["rain_total_mm"])
    monkeypatch.setattr(queue, "load_methodology", lambda conn, id: SimpleNamespace(slug="rain"))
    monkeypatch.setattr(registry, "plugin_for", lambda slug: Plugin(manifest=manifest, compute=lambda j, m: result))
    monkeypatch.setattr(queue, "save_and_complete", lambda conn, j, w, r: saved.append(r) or "hash")

    worker.process(None, SETTINGS, job())

    assert saved == [result]



def test_render_only_asks_for_plugins_with_a_map_image(monkeypatch):
    asked = []
    monkeypatch.setattr(registry, "renderable_slugs", lambda: ["with-map"])
    monkeypatch.setattr(images, "next_missing", lambda conn, slugs: asked.append(slugs))
    assert worker.render_next_image(None) is False
    assert asked == [["with-map"]]
