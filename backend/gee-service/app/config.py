import os
import socket
import subprocess
from collections.abc import Callable
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
    # Sealed into every v4 record: which code computed it.
    code_version: str
    # The job lock must outlast the Earth Engine deadline, or another worker
    # could take the job over while this one is still computing it.
    lock_minutes: int = 15
    ee_deadline_seconds: int = 600
    # Fallback poll; NOTIFY normally wakes the worker as soon as a job is queued.
    poll_seconds: int = 30


def git_commit(cwd: Path = SERVICE_DIR) -> str | None:
    """The checked-out commit, with -dirty when the worker's files have uncommitted changes;
    None without git or outside a checkout."""

    def git(*args: str) -> str:
        return subprocess.run(["git", *args], cwd=cwd, capture_output=True, text=True, check=True).stdout.strip()

    try:
        commit = git("rev-parse", "HEAD")
        changed = git("status", "--porcelain", "--", ".")
    except (OSError, subprocess.CalledProcessError):
        return None
    return commit + ("-dirty" if changed else "")


def resolve_code_version(env: dict[str, str], git: Callable[[], str | None] = git_commit) -> str:
    """CODE_VERSION, else the commit Render deployed, else the local checkout. The worker
    won't start without one: a record that can't say what computed it is weaker evidence."""
    for name in ("CODE_VERSION", "RENDER_GIT_COMMIT"):
        if value := env.get(name, "").strip():
            return value
    if version := git():
        return version
    raise ConfigError("Can't tell which code is running. Set CODE_VERSION, or run from a git checkout.")


def load_settings(
    env: dict[str, str] | None = None, git: Callable[[], str | None] = git_commit
) -> Settings:
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
        code_version=resolve_code_version(env, git),
    )
