"""Methodology plugins: the one place the worker learns what it can compute.

To add a methodology: write a module with a compute(job, methodology) -> Result function
(and optionally render(req) -> Rendered for a map image), register it here under its
slug, and seed the methodology and its measurement types in a migration. Nothing else in
the worker changes. See docs/architecture.md.
"""

from collections.abc import Callable
from dataclasses import dataclass

from app import compute, render
from app.compute import Result
from app.errors import JobError
from app.models import Job, Methodology
from app.render import ImageRequest, Rendered


@dataclass(frozen=True)
class Plugin:
    compute: Callable[[Job, Methodology], Result]
    # None when the methodology has no map image.
    render: Callable[[ImageRequest], Rendered] | None = None


PLUGINS: dict[str, Plugin] = {
    "ndvi-s2-median": Plugin(compute=compute.run, render=render.render),
}


def plugin_for(slug: str) -> Plugin:
    plugin = PLUGINS.get(slug)
    if plugin is None:
        raise JobError("UNSUPPORTED_METHODOLOGY", f"This worker has no plugin for {slug}.", retryable=False)
    return plugin
