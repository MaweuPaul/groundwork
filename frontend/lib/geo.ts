import type { MultiPolygon, Polygon } from '@/lib/types';

export type Ring = [number, number][];
export type PolygonCoords = Ring[];

const EARTH_RADIUS_M = 6_378_137;
const rad = (deg: number) => (deg * Math.PI) / 180;

// Spherical excess formula (same approach as turf/area); rings are closed, so the last point is skipped.
function ringArea(ring: Ring) {
  const n = ring.length - 1;
  if (n < 3) return 0;
  let total = 0;
  for (let i = 0; i < n; i++) {
    const prev = ring[(i - 1 + n) % n];
    const next = ring[(i + 1) % n];
    total += (rad(next[0]) - rad(prev[0])) * Math.sin(rad(ring[i][1]));
  }
  return Math.abs((total * EARTH_RADIUS_M * EARTH_RADIUS_M) / 2);
}

function distance([lon1, lat1]: [number, number], [lon2, lat2]: [number, number]) {
  const a =
    Math.sin(rad(lat2 - lat1) / 2) ** 2 +
    Math.cos(rad(lat1)) * Math.cos(rad(lat2)) * Math.sin(rad(lon2 - lon1) / 2) ** 2;
  return 2 * EARTH_RADIUS_M * Math.asin(Math.sqrt(a));
}

export function measure(polygons: PolygonCoords[]) {
  let areaM2 = 0;
  let perimeterM = 0;
  let vertices = 0;
  for (const [outer, ...holes] of polygons) {
    areaM2 += ringArea(outer) - holes.reduce((sum, hole) => sum + ringArea(hole), 0);
    for (let i = 1; i < outer.length; i++) perimeterM += distance(outer[i - 1], outer[i]);
    vertices += outer.length - 1;
  }
  return { areaHa: areaM2 / 10_000, perimeterM, vertices, pieces: polygons.length };
}

// Average of the outer-ring vertices: good enough to show where the parcel is, not for analysis.
export function centroid(polygons: PolygonCoords[]): [number, number] | null {
  const points = polygons.flatMap((polygon) => polygon[0].slice(0, -1));
  if (points.length === 0) return null;
  const [x, y] = points.reduce(([sx, sy], [px, py]) => [sx + px, sy + py], [0, 0]);
  return [x / points.length, y / points.length];
}

export function toGeometry(polygons: PolygonCoords[]): Polygon | MultiPolygon | null {
  if (polygons.length === 0) return null;
  if (polygons.length === 1) return { type: 'Polygon', coordinates: polygons[0] };
  return { type: 'MultiPolygon', coordinates: polygons };
}

export function bounds(polygons: PolygonCoords[]): [[number, number], [number, number]] {
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  for (const polygon of polygons) {
    for (const [x, y] of polygon[0]) {
      minX = Math.min(minX, x);
      minY = Math.min(minY, y);
      maxX = Math.max(maxX, x);
      maxY = Math.max(maxY, y);
    }
  }
  return [[minX, minY], [maxX, maxY]];
}

type GeoJSONInput = { type?: string; geometry?: GeoJSONInput; features?: GeoJSONInput[]; coordinates?: unknown };

function geometriesOf(input: GeoJSONInput): GeoJSONInput[] {
  if (input.type === 'FeatureCollection') return (input.features ?? []).flatMap(geometriesOf);
  if (input.type === 'Feature') return input.geometry ? geometriesOf(input.geometry) : [];
  return [input];
}

function isRing(value: unknown): value is Ring {
  return (
    Array.isArray(value) &&
    value.length >= 4 &&
    value.every(
      (p) =>
        Array.isArray(p) &&
        typeof p[0] === 'number' &&
        typeof p[1] === 'number' &&
        Math.abs(p[0]) <= 180 &&
        Math.abs(p[1]) <= 90,
    )
  );
}

function toPolygon(value: unknown): PolygonCoords {
  if (!Array.isArray(value) || value.length === 0 || !value.every(isRing)) {
    throw new Error(
      'Coordinates must be WGS84 longitude/latitude (EPSG:4326) and every ring needs at least 4 points.',
    );
  }
  // Two-dimensional, and 9 decimals (~0.1 mm) so the drawing tool accepts it.
  return value.map((ring) =>
    ring.map(([x, y]) => [Number(x.toFixed(9)), Number(y.toFixed(9))] as [number, number]),
  );
}

// Accepts a Polygon or MultiPolygon, a Feature, or a FeatureCollection of them.
export function parseBoundary(text: string): PolygonCoords[] {
  let input: GeoJSONInput;
  try {
    input = JSON.parse(text);
  } catch {
    throw new Error('The file is not valid JSON.');
  }

  const polygons: PolygonCoords[] = [];
  for (const geometry of geometriesOf(input)) {
    if (geometry.type === 'Polygon') polygons.push(toPolygon(geometry.coordinates));
    else if (geometry.type === 'MultiPolygon' && Array.isArray(geometry.coordinates))
      polygons.push(...geometry.coordinates.map(toPolygon));
    else throw new Error(`Only Polygon and MultiPolygon boundaries are supported, found ${geometry.type ?? 'unknown'}.`);
  }
  if (polygons.length === 0) throw new Error('The file has no polygons in it.');
  return polygons;
}
