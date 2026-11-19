import { CalendarDays, CloudRain, type LucideIcon, Satellite, Sprout } from 'lucide-react';

// How each methodology's results are shown. Names, units and descriptions of measurements come
// from the API; this only adds what the database doesn't say: which result headlines a row,
// and the context lines on the snapshot page.
type MethodologyView = {
  title: string;
  icon: LucideIcon;
  scale: string;
  // Help for the date range in the snapshot dialog.
  windowHelp: string;
  // Measurement order on the snapshot page; ones not listed follow.
  order: string[];
  summary: (m: Record<string, number>) => string | undefined;
  inputs: [label: string, value: string][];
  sources: SourcesView;
};

// How the sources a snapshot was computed from are listed.
export type SourcesView = {
  title: string;
  icon: LucideIcon;
  count: (n: number) => string;
  // One reading per UTC day: list the day, without a time or cloud cover.
  daily: boolean;
  note?: string;
};

const views: Record<string, MethodologyView> = {
  'ndvi-s2-median': {
    title: 'Vegetation measurements',
    icon: Sprout,
    scale: '10 m pixels · median composite',
    windowHelp:
      'The dates of imagery to use. Every clear Sentinel-2 view of the parcel in this range is combined into one median picture, so short gaps in coverage don’t matter.',
    order: ['ndvi_mean', 'ndvi_max', 'valid_pixel_pct'],
    summary: (m) =>
      m.ndvi_mean === undefined
        ? undefined
        : `NDVI ${m.ndvi_mean.toFixed(3)}` +
          (m.valid_pixel_pct === undefined ? '' : ` · ${m.valid_pixel_pct.toFixed(0)}% clear`),
    inputs: [
      ['Sensor', 'Sentinel-2, Level-2A surface reflectance'],
      ['Bands', 'B4 (red), B8 (near-infrared)'],
      ['Composite', 'median, NDVI = (B8 − B4) / (B8 + B4)'],
      ['Resolution', '10 m'],
    ],
    sources: {
      title: 'Imagery sources',
      icon: Satellite,
      count: (n) => `${n} ${n === 1 ? 'scene' : 'scenes'} in the composite`,
      daily: false,
      note: 'Copernicus Sentinel-2 Level-2A surface reflectance via Google Earth Engine. Scene cloud is the whole tile’s estimate; clouds over the parcel were masked per pixel with Cloud Score+.',
    },
  },
  'chirps-rain-season': {
    title: 'Rainfall measurements',
    icon: CloudRain,
    scale: '~5.5 km pixels · 1991–2020 normal',
    windowHelp:
      'The days to total, start and end included. Each day’s rainfall is summed and compared with the same dates in 1991–2020. CHIRPS is published a few weeks after each month ends, so very recent days aren’t available yet.',
    order: ['rain_total_mm', 'rain_normal_mm', 'rain_pct_of_normal', 'rainy_days', 'longest_dry_spell_days'],
    summary: (m) =>
      m.rain_total_mm === undefined
        ? undefined
        : `${Math.round(m.rain_total_mm)} mm` +
          (m.rain_pct_of_normal === undefined ? '' : ` · ${m.rain_pct_of_normal.toFixed(0)}% of normal`),
    inputs: [
      ['Dataset', 'CHIRPS daily rainfall, UCSB Climate Hazards Center'],
      ['Normal', 'Same calendar window, 1991–2020'],
      ['Resolution', '~5.5 km (0.05°); values describe the area around the parcel'],
    ],
    sources: {
      title: 'Rainfall data',
      icon: CalendarDays,
      count: (n) => `${n} daily ${n === 1 ? 'reading' : 'readings'}`,
      daily: true,
      note: 'CHIRPS daily rainfall from the UCSB Climate Hazards Center, via Google Earth Engine. Each day’s grid was averaged over the parcel; the version is the one Earth Engine reported when it was read.',
    },
  },
};

export const methodologyView = (slug: string | undefined) => (slug ? views[slug] : undefined);

const genericSources: SourcesView = {
  title: 'Sources',
  icon: Satellite,
  count: (n) => `${n} ${n === 1 ? 'source' : 'sources'}`,
  daily: false,
};

export const sourcesView = (slug: string | undefined) => methodologyView(slug)?.sources ?? genericSources;

// One-line result for tables and notices, e.g. "NDVI 0.742 · 90% clear" or "575 mm · 155% of normal".
export const resultSummary = (slug: string | undefined, measurements: Record<string, number> | undefined) =>
  measurements ? methodologyView(slug)?.summary(measurements) : undefined;

// Measurement slugs in display order: the methodology's own order first, then any others.
export function orderedSlugs(slug: string, measurements: Record<string, number>) {
  const order = methodologyView(slug)?.order ?? [];
  const known = order.filter((s) => s in measurements);
  return [
    ...known,
    ...Object.keys(measurements)
      .filter((s) => !order.includes(s))
      .sort(),
  ];
}

export function formatMeasurement(value: number, unit: string | null | undefined) {
  switch (unit) {
    case null:
    case undefined:
      return value.toFixed(3);
    case '%':
      return `${value.toFixed(1)}%`;
    case 'mm':
      return `${value.toFixed(1)} mm`;
    case 'days':
      return `${Math.round(value)} ${Math.round(value) === 1 ? 'day' : 'days'}`;
    default:
      return `${value.toLocaleString('en-US', { maximumFractionDigits: 2 })} ${unit}`;
  }
}

// Short text per parameter, by name, so a series can be described without knowing its methodology.
const parameterText: Record<string, (v: number) => string> = {
  max_scene_cloud_pct: (v) => `scenes ≤${v}%`,
  cloud_score_threshold: (v) => `clear ≥${v.toFixed(2)}`,
  min_clear_pct: (v) => `coverage ≥${v}%`,
  rainy_day_mm: (v) => `rainy day ≥${v} mm`,
};

export const settingsSummary = (parameters: Record<string, number> | undefined) =>
  Object.entries(parameters ?? {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, v]) => parameterText[name]?.(v) ?? `${name.replaceAll('_', ' ')} ${v}`)
    .join(' · ');
