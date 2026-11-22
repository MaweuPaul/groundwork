from datetime import date, timedelta


def end_exclusive(day: date) -> str:
    """Snapshot windows include their end day; Earth Engine's filterDate excludes it."""
    return (day + timedelta(days=1)).isoformat()


def shift_years(day: date, years: int) -> date:
    try:
        return day.replace(year=day.year + years)
    except ValueError:  # 29 February in a year that has none
        return day.replace(year=day.year + years, day=28)
