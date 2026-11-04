"""Methodology plugins: the one place the worker learns what it can compute.

To add a methodology: create a folder in app/plugins exporting PLUGIN (see app/contract.py),
register it here under its slug, and seed the methodology and its measurement types in a
migration. Nothing else in the worker changes. See docs/architecture.md.
"""

from app.contract import Plugin
from app.errors import JobError
from app.plugins import ndvi_s2_median

PLUGINS: dict[str, Plugin] = {
    "ndvi-s2-median": ndvi_s2_median.PLUGIN,
}


def renderable_slugs() -> list[str]:
    return [slug for slug, plugin in PLUGINS.items() if plugin.render is not None]


def plugin_for(slug: str) -> Plugin:
    plugin = PLUGINS.get(slug)
    if plugin is None:
        raise JobError("UNSUPPORTED_METHODOLOGY", f"This worker has no plugin for {slug}.", retryable=False)
    return plugin
