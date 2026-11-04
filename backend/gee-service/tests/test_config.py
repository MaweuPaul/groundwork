import pytest

from app.config import SERVICE_DIR, ConfigError, load_settings


def env(tmp_path, **overrides):
    key = tmp_path / "key.json"
    key.write_text("{}")
    return {
        "DATABASE_URL": "postgresql://gee_worker:x@localhost/groundwork",
        "GEE_PROJECT": "my-project",
        "GOOGLE_APPLICATION_CREDENTIALS": str(key),
        "WORKER_ID": "w1",
        **overrides,
    }


def test_loads_settings(tmp_path):
    settings = load_settings(env(tmp_path))
    assert settings.gee_project == "my-project"
    assert settings.worker_id == "w1"
    assert settings.lock_minutes * 60 > settings.ee_deadline_seconds


def test_missing_settings_are_listed(tmp_path):
    with pytest.raises(ConfigError, match="DATABASE_URL, GEE_PROJECT"):
        load_settings(env(tmp_path, DATABASE_URL="", GEE_PROJECT=" "))


def test_missing_key_file_is_reported(tmp_path):
    with pytest.raises(ConfigError, match="not found"):
        load_settings(env(tmp_path, GOOGLE_APPLICATION_CREDENTIALS=str(tmp_path / "nope.json")))


def test_relative_key_path_is_resolved_from_the_service_dir(tmp_path):
    with pytest.raises(ConfigError, match=str(SERVICE_DIR / "credentials" / "k.json").replace("\\", "\\\\")):
        load_settings(env(tmp_path, GOOGLE_APPLICATION_CREDENTIALS="credentials/k.json"))


def test_worker_id_defaults_to_host_and_pid(tmp_path):
    settings = load_settings(env(tmp_path, WORKER_ID=""))
    assert "-" in settings.worker_id
