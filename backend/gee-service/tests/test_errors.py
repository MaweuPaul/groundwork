import ee
import pytest

from app.errors import JobError, classify


@pytest.mark.parametrize(
    ("message", "code", "retryable"),
    [
        ("Quota exceeded for quota metric 'Requests'", "GEE_QUOTA", True),
        ("Too many concurrent aggregations.", "GEE_QUOTA", True),
        ("Deadline exceeded", "GEE_TIMEOUT", True),
        ("Request timed out", "GEE_TIMEOUT", True),
        ("An internal error has occurred", "UPSTREAM_ERROR", True),
        ("Geometry.polygon: Invalid geometry.", "GEE_ERROR", False),
        ("HTTP 503 Service Unavailable", "UPSTREAM_ERROR", True),
        # Digits inside an ID are not a status code.
        ("Permission 'earthengine.x' denied on resource 'projects/groundwork-503310'", "GEE_ERROR", False),
    ],
)
def test_earth_engine_errors(message, code, retryable):
    err = classify(ee.EEException(message))
    assert (err.code, err.retryable) == (code, retryable)
    assert err.message == message


@pytest.mark.parametrize(
    "message",
    ["Computation timed out.", "User memory limit exceeded.", "Too many pixels in the region."],
)
def test_requests_over_earth_engine_limits_fail_at_once(message):
    err = classify(ee.EEException(message))
    assert err.code == "COMPUTATION_TOO_LARGE"
    assert not err.retryable
    assert "shorter window" in err.message
    assert message in err.message


def test_network_errors_are_retried():
    err = classify(ConnectionResetError("connection reset by peer"))
    assert (err.code, err.retryable) == ("UPSTREAM_ERROR", True)


def test_unexpected_errors_are_retried_with_their_type():
    err = classify(KeyError("ndvi_mean"))
    assert err.code == "WORKER_ERROR"
    assert err.retryable
    assert err.message.startswith("KeyError")


def test_job_errors_pass_through():
    original = JobError("NO_IMAGERY", "nothing", retryable=False)
    assert classify(original) is original


def test_long_messages_are_truncated():
    err = classify(ee.EEException("x" * 5000))
    assert len(err.message) == 1000
