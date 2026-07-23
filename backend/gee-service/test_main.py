import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch, MagicMock
from main import app
from models import SnapshotRequest, MethodologyConfig

client = TestClient(app)

@pytest.fixture(autouse=True)
def mock_ee():
    """Mock the entire Earth Engine library so we don't need real auth or network calls during tests."""
    with patch("main.ee") as mock_ee_main, patch("services.ee") as mock_ee_services:
        # Pretend Earth Engine is always initialized successfully
        mock_ee_main.data.is_initialized.return_value = True
        mock_ee_services.data.is_initialized.return_value = True
        yield mock_ee_services

def test_health_check():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json()["status"] == "healthy"

@patch("main.process_snapshot")
def test_calculate_snapshot_success(mock_process_snapshot):
    # Mock the return value of process_snapshot
    from models import SnapshotResponse, ImagerySource
    mock_process_snapshot.return_value = SnapshotResponse(
        measurements={"ndvi_mean": 0.5},
        imagery_sources=[
            ImagerySource(
                satellite="sentinel-2",
                scene_id="test_scene",
                acquisition_time="2023-01-01T00:00:00Z"
            )
        ]
    )

    request_payload = {
        "geometry_geojson": {
            "type": "Polygon",
            "coordinates": [[[0, 0], [1, 0], [1, 1], [0, 1], [0, 0]]]
        },
        "date_start": "2023-01-01",
        "date_end": "2023-01-31",
        "methodology": {
            "satellite": "sentinel-2",
            "index_type": "ndvi"
        }
    }

    response = client.post("/calculate-snapshot", json=request_payload)
    assert response.status_code == 200
    data = response.json()
    assert "measurements" in data
    assert data["measurements"]["ndvi_mean"] == 0.5
    assert len(data["imagery_sources"]) == 1
    assert data["imagery_sources"][0]["scene_id"] == "test_scene"

def test_calculate_snapshot_invalid_payload():
    # Missing geometry
    request_payload = {
        "date_start": "2023-01-01",
        "date_end": "2023-01-31",
        "methodology": {
            "satellite": "sentinel-2",
            "index_type": "ndvi"
        }
    }
    response = client.post("/calculate-snapshot", json=request_payload)
    # FastAPI should automatically reject this with a 422 Unprocessable Entity
    assert response.status_code == 422
