import json

import ee

from app.config import Settings


def initialize(settings: Settings) -> None:
    """Signs in to Earth Engine as the service account. Raises if the key or project is wrong."""
    key = json.loads(settings.credentials_file.read_text())
    credentials = ee.ServiceAccountCredentials(key["client_email"], key_file=str(settings.credentials_file))
    ee.Initialize(credentials, project=settings.gee_project)
    ee.data.setDeadline(settings.ee_deadline_seconds * 1000)
