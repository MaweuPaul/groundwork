// Reads a shapefile's .prj (WKT) and decides how to turn its coordinates into WGS84
// longitude/latitude. Only systems that convert reliably are accepted: any WGS84-based
// one, and Arc 1960 (Kenya's survey datum). Arc 1960 sits 200-300 m from WGS84, and many
// .prj files omit the shift, so its parameters come from EPSG, not from the file.

export type Crs = {
  label: string;
  // A proj4 definition, or null when the coordinates are already WGS84 longitude/latitude.
  proj4: string | null;
};

// EPSG:1122, Arc 1960 to WGS 84 (Kenya and Tanzania), accurate to about 20 m.
const ARC_1960_TO_WGS84 = '+ellps=clrk80 +towgs84=-160,-6,-302,0,0,0,0';

const norm = (s: string) => s.toLowerCase().replace(/[^a-z0-9]/g, '');

const quoted = (wkt: string, keyword: string) =>
  new RegExp(`${keyword}\\s*\\[\\s*"([^"]+)"`, 'i').exec(wkt)?.[1] ?? null;

function parameter(wkt: string, name: string): number | null {
  const match = new RegExp(`PARAMETER\\s*\\[\\s*"${name}"\\s*,\\s*(-?[\\d.eE+-]+)`, 'i').exec(wkt);
  return match ? Number(match[1]) : null;
}

const pretty = (name: string) => name.replace(/_/g, ' ');

function datumKind(wkt: string): 'wgs84' | 'arc1960' | null {
  const datum = quoted(wkt, 'DATUM');
  if (!datum) return null;
  const n = norm(datum).replace(/^d(?=wgs|arc)/, '');
  if (['wgs1984', 'wgs84', 'worldgeodeticsystem1984'].includes(n)) return 'wgs84';
  if (n === 'arc1960') return 'arc1960';
  return null;
}

// Arc 1960 is used with UTM zones 36 and 37, north and south (EPSG:21036/21037/21096/21097).
function arc1960Utm(wkt: string): Crs | null {
  const projection = norm(quoted(wkt, 'PROJECTION') ?? '');
  const unit = norm(quoted(wkt.slice(wkt.lastIndexOf('UNIT')), 'UNIT') ?? '');
  const cm = parameter(wkt, 'central_meridian');
  const k = parameter(wkt, 'scale_factor');
  const fe = parameter(wkt, 'false_easting');
  const fn = parameter(wkt, 'false_northing');
  const lat0 = parameter(wkt, 'latitude_of_origin') ?? 0;

  if (projection !== 'transversemercator' || !['meter', 'metre'].includes(unit)) return null;
  if (k !== 0.9996 || fe !== 500000 || lat0 !== 0 || (fn !== 0 && fn !== 10000000)) return null;
  const zone = cm === 33 ? 36 : cm === 39 ? 37 : null;
  if (!zone) return null;

  const south = fn === 10000000;
  const epsg = south ? `210${zone}` : `2109${zone - 30}`;
  return {
    label: `Arc 1960 / UTM zone ${zone}${south ? 'S' : 'N'} (EPSG:${epsg})`,
    proj4: `+proj=utm +zone=${zone}${south ? ' +south' : ''} ${ARC_1960_TO_WGS84} +units=m +no_defs`,
  };
}

export function resolveCrs(prj: string): Crs {
  const wkt = prj.trim();
  const projected = /^(PROJCS|PROJCRS|PROJECTEDCRS)\b/i.test(wkt);
  const name = pretty(
    quoted(wkt, projected ? '(?:PROJCS|PROJCRS|PROJECTEDCRS)' : '(?:GEOGCS|GEOGCRS|GEODCRS)') ?? 'unnamed system',
  );
  const kind = datumKind(wkt);

  if (kind === 'wgs84') {
    if (!projected) return { label: 'WGS 84 (EPSG:4326)', proj4: null };
    return { label: name, proj4: wkt };
  }

  if (kind === 'arc1960') {
    if (!projected) return { label: 'Arc 1960 (EPSG:4210)', proj4: `+proj=longlat ${ARC_1960_TO_WGS84} +no_defs` };
    const utm = arc1960Utm(wkt);
    if (utm) return utm;
  }

  throw new Error(
    `The layer is in ${name}, which Groundwork can't convert reliably yet. Export it as WGS 84 (EPSG:4326) or Arc 1960 / UTM zone 37S (EPSG:21037) and upload again.`,
  );
}
