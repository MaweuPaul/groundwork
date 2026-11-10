"""Checks each plugin's manifest against what the migrations published, at worker startup."""

import psycopg

from app.contract import Manifest, Parameter, Plugin


def published_parameters(conn: psycopg.Connection, slug: str, version: str) -> list[Parameter] | None:
    """The parameters stored for this methodology, or None if the methodology isn't there."""
    found = conn.execute(
        "SELECT 1 FROM methodologies WHERE slug = %s AND version = %s", (slug, version)
    ).fetchone()
    if found is None:
        return None
    rows = conn.execute(
        """
        SELECT p.name, p.label, p.type,
               p.min_value::float8, p.max_value::float8, p.default_value::float8
        FROM methodology_parameters p
        JOIN methodologies m ON m.id = p.methodology_id
        WHERE m.slug = %s AND m.version = %s
        """,
        (slug, version),
    ).fetchall()
    return [Parameter(*row) for row in rows]


def differences(manifest: Manifest, published: list[Parameter] | None) -> list[str]:
    label = f"{manifest.slug}@{manifest.version}"
    if published is None:
        return [f"{label} is not in methodologies; add a migration for it."]
    ours = {p.name: p for p in manifest.parameters}
    theirs = {p.name: p for p in published}
    return [
        f"{label}: parameter {name} is {ours.get(name)} in the manifest but {theirs.get(name)} in the database."
        for name in sorted(ours.keys() | theirs.keys())
        if ours.get(name) != theirs.get(name)
    ]


def check(conn: psycopg.Connection, plugins: dict[str, Plugin]) -> list[str]:
    problems = []
    for plugin in plugins.values():
        m = plugin.manifest
        problems += differences(m, published_parameters(conn, m.slug, m.version))
    return problems
