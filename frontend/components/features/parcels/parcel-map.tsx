'use client';

import 'maplibre-gl/dist/maplibre-gl.css';

import type { Map as MapLibreMap } from 'maplibre-gl';
import { useEffect, useRef, useState } from 'react';

import { bounds } from '@/lib/geo';
import { basemapStyle, loadMapLibre, setBasemap as applyBasemap, type Basemap } from '@/lib/map';
import type { MultiPolygon } from '@/lib/types';

const LIME = '#d9eaa3';

const ParcelMap = ({ geometry }: { geometry: MultiPolygon }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const [ready, setReady] = useState(false);
  const [basemap, setBasemap] = useState<Basemap>('satellite');

  useEffect(() => {
    let cancelled = false;

    (async () => {
      const maplibregl = await loadMapLibre();
      if (cancelled || !containerRef.current) return;

      const map = new maplibregl.Map({
        container: containerRef.current,
        style: basemapStyle,
        bounds: bounds(geometry.coordinates),
        fitBoundsOptions: { padding: 48, maxZoom: 17 },
        attributionControl: { compact: true },
      });
      mapRef.current = map;
      map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right');
      map.addControl(new maplibregl.FullscreenControl(), 'bottom-right');

      map.on('load', () => {
        if (cancelled) return;
        map.addSource('parcel', {
          type: 'geojson',
          data: { type: 'Feature', geometry, properties: {} },
        });
        map.addLayer({
          id: 'parcel-fill',
          type: 'fill',
          source: 'parcel',
          paint: { 'fill-color': LIME, 'fill-opacity': 0.2 },
        });
        map.addLayer({
          id: 'parcel-outline',
          type: 'line',
          source: 'parcel',
          paint: { 'line-color': LIME, 'line-width': 2 },
        });
        setReady(true);
      });
    })();

    return () => {
      cancelled = true;
      mapRef.current?.remove();
      mapRef.current = null;
    };
  }, [geometry]);

  const chooseBasemap = (next: Basemap) => {
    if (!mapRef.current) return;
    applyBasemap(mapRef.current, next);
    setBasemap(next);
  };

  const toggle = (active: boolean) =>
    `rounded-sm px-2.5 py-1.5 font-mono text-micro font-semibold tracking-[0.08em] uppercase transition-colors ${
      active ? 'bg-primary text-primary-foreground' : 'text-foreground hover:bg-surface-container'
    }`;

  return (
    <div className="relative h-90 overflow-hidden rounded-md border border-border bg-surface-container">
      {/* maplibre-gl.css forces position: relative on this element, so it's sized with h-full, not inset-0. */}
      <div ref={containerRef} className="h-full w-full" />

      <div className="absolute top-3 left-3 z-10 flex gap-1 rounded-md bg-surface-lowest p-1 shadow-sm">
        <button type="button" disabled={!ready} onClick={() => chooseBasemap('satellite')} className={toggle(basemap === 'satellite')}>
          Satellite
        </button>
        <button type="button" disabled={!ready} onClick={() => chooseBasemap('streets')} className={toggle(basemap === 'streets')}>
          Streets
        </button>
      </div>

      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center font-mono text-caption text-muted-foreground">
          Loading map…
        </div>
      )}
    </div>
  );
};

export default ParcelMap;
