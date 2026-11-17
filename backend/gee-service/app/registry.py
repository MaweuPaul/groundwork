"""Methodology plugins: the one place the worker learns what it can compute.

To add a methodology: create a folder in app/plugins exporting PLUGIN (see app/contract.py),
and seed the methodology and its measurement types in a migration. It is found automatically,
so nothing else in the worker changes. See docs/architecture.md.
"""

import importlib
import pkgutil

from app import plugins
from app.contract import Plugin
from app.errors import JobError


def discover()-> dict[str,Plugin]:
    found: dict[str,Plugin] = {}
    for module in pkgutil.iter_modules(plugins.__path__):
        plugin =importlib.import_module(f"app.plugins.{module.name}").PLUGIN
        slug = plugin.manifest.slug
        if slug in found:
            raise RuntimeError(f"Two plugins use the slug {slug}.")
        found[slug] = plugin
    return found


PLUGINS = discover()


def plugin_for(slug: str) -> Plugin:
    plugin = PLUGINS.get(slug)
    if plugin is None:
        raise JobError("UNSUPPORTED_METHODOLOGY", f"This worker has no plugin for {slug}.", retryable=False)
    return plugin
