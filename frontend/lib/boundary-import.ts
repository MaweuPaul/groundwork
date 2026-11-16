import { resolveCrs } from '@/lib/crs';
import { parseBoundary, polygonsOf, type GeoJSONInput, type PolygonCoords } from '@/lib/geo';

export type ImportedBoundary = { polygons: PolygonCoords[]; source: string; crs: string };

type Entry = { name: string; bytes: Uint8Array };

const extension = (name: string) => name.slice(name.lastIndexOf('.') + 1).toLowerCase();
const stem = (name: string) => name.slice(name.lastIndexOf('/') + 1, name.lastIndexOf('.')).toLowerCase();

async function entriesOf(files: File[]): Promise<Entry[]> {
  const entries: Entry[] = [];
  for (const file of files) {
    const bytes = new Uint8Array(await file.arrayBuffer());
    if (extension(file.name) !== 'zip') {
      entries.push({ name: file.name, bytes });
      continue;
    }
    const { unzipSync } = await import('fflate');
    let unzipped: Record<string, Uint8Array>;
    try {
      unzipped = unzipSync(bytes);
    } catch {
      throw new Error(`${file.name} isn't a readable zip file.`);
    }
    for (const [name, data] of Object.entries(unzipped)) {
      if (!name.endsWith('/') && !name.startsWith('__MACOSX/')) entries.push({ name, bytes: data });
    }
  }
  return entries;
}

type Coords = number[] | Coords[];

function reproject(coords: Coords, forward: (p: [number, number]) => [number, number]): Coords {
  if (typeof coords[0] === 'number') return forward([coords[0], coords[1] as number]);
  return (coords as Coords[]).map((c) => reproject(c, forward));
}

async function fromShapefile(entries: Entry[]): Promise<ImportedBoundary> {
  const shps = entries.filter((e) => extension(e.name) === 'shp');
  if (shps.length === 0) {
    throw new Error('No .shp file found. Upload the .shp together with its .prj (or the whole set zipped).');
  }
  if (shps.length > 1) {
    throw new Error(
      `Found ${shps.length} layers (${shps.map((s) => s.name).join(', ')}). Upload one boundary layer at a time.`,
    );
  }
  const shp = shps[0];
  const prjs = entries.filter((e) => extension(e.name) === 'prj');
  const prj = prjs.find((p) => stem(p.name) === stem(shp.name)) ?? (prjs.length === 1 ? prjs[0] : undefined);
  if (!prj) {
    throw new Error(
      `${shp.name} has no .prj file, so its coordinate system is unknown. Add the .prj from the same export and upload again.`,
    );
  }

  const crs = resolveCrs(new TextDecoder().decode(prj.bytes));

  const { parseShp } = await import('shpjs');
  let geometries: GeoJSONInput[];
  try {
    geometries = parseShp(shp.bytes) as GeoJSONInput[];
  } catch {
    throw new Error(`${shp.name} couldn't be read as a shapefile.`);
  }
  geometries = geometries.filter(Boolean);

  if (crs.proj4) {
    const { default: proj4 } = await import('proj4');
    const converter = proj4(crs.proj4, 'WGS84');
    const forward = (p: [number, number]) => converter.forward(p) as [number, number];
    geometries = geometries.map((g) => ({ ...g, coordinates: reproject(g.coordinates as Coords, forward) }));
  }

  let polygons: PolygonCoords[];
  try {
    polygons = polygonsOf({
      type: 'FeatureCollection',
      features: geometries.map((geometry) => ({ type: 'Feature', geometry })),
    });
  } catch (e) {
    if (e instanceof Error && e.message.startsWith('Coordinates must be')) {
      throw new Error(`The coordinates don't fit ${crs.label}. Check that ${prj.name} belongs to this layer.`);
    }
    throw e;
  }

  return {
    polygons,
    source: shp.name.slice(shp.name.lastIndexOf('/') + 1),
    crs: crs.label,
  };
}

const GEOJSON = ['geojson', 'json'];
const SHAPEFILE_PARTS = ['zip', 'shp', 'shx', 'dbf', 'prj', 'cpg', 'sbn', 'sbx', 'qmd', 'xml'];

// A GeoJSON file in WGS84, or a shapefile: the .shp with its .prj (other parts are ignored), or all of it zipped.
export async function importBoundary(files: File[]): Promise<ImportedBoundary> {
  if (files.length === 1 && GEOJSON.includes(extension(files[0].name))) {
    return { polygons: parseBoundary(await files[0].text()), source: files[0].name, crs: 'WGS 84 (EPSG:4326)' };
  }
  const unsupported = files.find((f) => !SHAPEFILE_PARTS.includes(extension(f.name)));
  if (unsupported) {
    throw new Error(
      `${unsupported.name} isn't a supported file. Upload GeoJSON, or a shapefile (.shp with its .prj, or zipped).`,
    );
  }
  return fromShapefile(await entriesOf(files));
}
