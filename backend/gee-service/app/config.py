import os
import socket
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv

SERVICE_DIR = Path(__file__).resolve().parent.parent


class ConfigError(Exception):
    pass


@dataclass(frozen=True)
class Settings:
    database_url: str
    gee_project: str
    credentials_file: Path
    worker_id: str
    # The job lock must outlast the Earth Engine deadline, or another worker
    # could take the job over while this one is still computing it.
    lock_minutes: int = 15
    ee_deadline_seconds: int = 600
    # Fallback poll; NOTIFY normally wakes the worker as soon as a job is queued.
    poll_seconds: int = 30


def load_settings(env: dict[str, str] | None = None) -> Settings:
    if env is None:
        load_dotenv(SERVICE_DIR / ".env")
        env = dict(os.environ)

    required = ["DATABASE_URL", "GEE_PROJECT", "GOOGLE_APPLICATION_CREDENTIALS"]
    missing = [name for name in required if not env.get(name, "").strip()]
    if missing:
        raise ConfigError(f"Missing settings: {', '.join(missing)}. See .env.example.")

    credentials_file = Path(env["GOOGLE_APPLICATION_CREDENTIALS"])
    if not credentials_file.is_absolute():
        credentials_file = SERVICE_DIR / credentials_file
    if not credentials_file.is_file():
        raise ConfigError(f"Service account key not found: {credentials_file}")

    return Settings(
        database_url=env["DATABASE_URL"],
        gee_project=env["GEE_PROJECT"],
        credentials_file=credentials_file,
        worker_id=env.get("WORKER_ID") or f"{socket.gethostname()}-{os.getpid()}",
    )
