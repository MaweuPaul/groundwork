import type { StyleSpecification } from 'maplibre-gl';

const ESRI = 'https://server.arcgisonline.com/ArcGIS/rest/services';

export type Basemap = 'satellite' | 'streets';

export const basemapStyle: StyleSpecification = {
  version: 8,
  sources: {
    satellite: {
      type: 'raster',
      tiles: [`${ESRI}/World_Imagery/MapServer/tile/{z}/{y}/{x}`],
      tileSize: 256,
      maxzoom: 19,
      attribution: 'Imagery © Esri, Maxar, Earthstar Geographics',
    },
    labels: {
      type: 'raster',
      tiles: [`${ESRI}/Reference/World_Boundaries_and_Places/MapServer/tile/{z}/{y}/{x}`],
      tileSize: 256,
      maxzoom: 19,
    },
    streets: {
      type: 'raster',
      tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
      tileSize: 256,
      maxzoom: 19,
      attribution: '© OpenStreetMap contributors',
    },
  },
  layers: [
    { id: 'satellite', type: 'raster', source: 'satellite' },
    { id: 'labels', type: 'raster', source: 'labels' },
    { id: 'streets', type: 'raster', source: 'streets', layout: { visibility: 'none' } },
  ],
};

// MapLibre touches window on import, so it loads only in the browser. It also looks for its
// worker next to its own file, which the bundler doesn't copy, so the URL is set explicitly.
export async function loadMapLibre() {
  const maplibregl = await import('maplibre-gl');
  maplibregl.setWorkerUrl(new URL('maplibre-gl/dist/maplibre-gl-worker.mjs', import.meta.url).href);
  return maplibregl;
}

export function setBasemap(map: import('maplibre-gl').Map, basemap: Basemap) {
  const satellite = basemap === 'satellite' ? 'visible' : 'none';
  map.setLayoutProperty('satellite', 'visibility', satellite);
  map.setLayoutProperty('labels', 'visibility', satellite);
  map.setLayoutProperty('streets', 'visibility', basemap === 'streets' ? 'visible' : 'none');
}
