import re
from dataclasses import dataclass

import ee


@dataclass
class JobError(Exception):
    """A failure recorded on the job. code is stored and shown to clients; message stays internal."""

    code: str
    message: str
    retryable: bool

    def __str__(self) -> str:
        return f"{self.code}: {self.message}"


# Matched against Earth Engine's error text, which has no stable error codes.
# Interactive requests are capped (about 5 minutes and a memory limit). Hitting the cap means
# this input is too big, and every retry would hit it again, so these fail at once.
_TOO_LARGE_EE = ("computation timed out", "memory limit exceeded", "too many pixels")
TOO_LARGE_MESSAGE = (
    "Earth Engine couldn't compute this in one request: the parcel and window are too large "
    "together. Use a shorter window or a smaller parcel."
)

# HTTP status codes are matched as whole numbers, so a project ID like "groundwork-503310"
# doesn't count as a 503.
_RETRYABLE_EE = [
    ("GEE_QUOTA", re.compile(r"quota|too many concurrent|rate limit|429")),
    ("GEE_TIMEOUT", re.compile(r"deadline|timed out|timeout")),
    ("UPSTREAM_ERROR", re.compile(r"internal error|service unavailable|backend error|50[03]")),
]


def classify(exc: BaseException) -> JobError:
    """Turns any exception from computing a job into a JobError for fail_snapshot_job."""
    if isinstance(exc, JobError):
        return exc

    message = str(exc)[:1000] or type(exc).__name__

    if isinstance(exc, ee.EEException):
        lowered = message.lower()
        if any(n in lowered for n in _TOO_LARGE_EE):
            return JobError("COMPUTATION_TOO_LARGE", f"{TOO_LARGE_MESSAGE} ({message})", retryable=False)
        for code, pattern in _RETRYABLE_EE:
            if pattern.search(lowered):
                return JobError(code, message, retryable=True)
        # e.g. an invalid geometry or a missing permission: retrying the same request won't help.
        return JobError("GEE_ERROR", message, retryable=False)

    if isinstance(exc, (ConnectionError, TimeoutError, OSError)):
        return JobError("UPSTREAM_ERROR", message, retryable=True)

    return JobError("WORKER_ERROR", f"{type(exc).__name__}: {message}", retryable=True)
