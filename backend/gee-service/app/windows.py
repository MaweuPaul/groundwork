from datetime import date, timedelta


def end_exclusive(day: date) -> str:
    """Snapshot windows include their end day; Earth Engine's filterDate excludes it."""
    return (day + timedelta(days=1)).isoformat()
