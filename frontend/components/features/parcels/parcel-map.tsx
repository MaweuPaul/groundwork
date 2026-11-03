'use client';

import 'maplibre-gl/dist/maplibre-gl.css';

import { ChevronDown } from 'lucide-react';
import type { Map as MapLibreMap } from 'maplibre-gl';
import { useEffect, useRef, useState } from 'react';

import { bounds } from '@/lib/geo';
import { basemapStyle, loadMapLibre, setBasemap as applyBasemap, type Basemap } from '@/lib/map';
import type { MultiPolygon, NdviImage } from '@/lib/types';

const LIME = '#d9eaa3';

export type NdviLayer = {
  shortId: string;
  label: string;
  image: NdviImage;
};

const ParcelMap = ({ geometry, ndviLayers }: { geometry: MultiPolygon; ndviLayers: NdviLayer[] }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const [ready, setReady] = useState(false);
  const [basemap, setBasemap] = useState<Basemap>('satellite');
  const [showNdvi, setShowNdvi] = useState(true);
  const [selectedId, setSelectedId] = useState<string>();

  // Newest first; falls back to the newest when the chosen one is gone or none was chosen.
  const selected = ndviLayers.find((l) => l.shortId === selectedId) ?? ndviLayers[0];
  const active = showNdvi ? selected : undefined;

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

  // The NDVI image sits under the boundary outline. The boundary's tint is turned off while
  // it shows, so the colours aren't shifted.
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !ready) return;

    if (map.getLayer('ndvi')) map.removeLayer('ndvi');
    if (map.getSource('ndvi')) map.removeSource('ndvi');
    map.setPaintProperty('parcel-fill', 'fill-opacity', active ? 0 : 0.2);
    if (!active) return;

    const [west, south, east, north] = active.image.bounds;
    map.addSource('ndvi', {
      type: 'image',
      url: `/api/snapshots/${active.shortId}/ndvi`,
      coordinates: [
        [west, north],
        [east, north],
        [east, south],
        [west, south],
      ],
    });
    map.addLayer(
      { id: 'ndvi', type: 'raster', source: 'ndvi', paint: { 'raster-opacity': 0.92, 'raster-fade-duration': 0 } },
      'parcel-fill',
    );
  }, [ready, active]);

  const chooseBasemap = (next: Basemap) => {
    if (!mapRef.current) return;
    applyBasemap(mapRef.current, next);
    setBasemap(next);
  };

  const toggle = (on: boolean) =>
    `rounded-sm px-2.5 py-1.5 font-mono text-micro font-semibold tracking-[0.08em] uppercase transition-colors disabled:opacity-40 ${
      on ? 'bg-primary text-primary-foreground' : 'text-foreground hover:bg-surface-container'
    }`;

  const vis = active?.image.visualization;

  return (
    <div className="relative h-90 overflow-hidden rounded-md border border-border bg-surface-container">
      {/* maplibre-gl.css forces position: relative on this element, so it's sized with h-full, not inset-0. */}
      <div ref={containerRef} className="h-full w-full" />

      <div className="absolute top-3 left-3 z-10 flex flex-wrap items-center gap-2">
        <div className="flex gap-1 rounded-md bg-surface-lowest p-1 shadow-sm">
          <button type="button" disabled={!ready} onClick={() => chooseBasemap('satellite')} className={toggle(basemap === 'satellite')}>
            Satellite
          </button>
          <button type="button" disabled={!ready} onClick={() => chooseBasemap('streets')} className={toggle(basemap === 'streets')}>
            Streets
          </button>
        </div>
        {ndviLayers.length > 0 && (
          <div className="flex items-center gap-1 rounded-md bg-surface-lowest p-1 shadow-sm">
            <button
              type="button"
              disabled={!ready}
              aria-pressed={showNdvi}
              onClick={() => setShowNdvi((on) => !on)}
              className={toggle(showNdvi)}
            >
              NDVI
            </button>
            {ndviLayers.length > 1 && (
              <div className="relative">
                <select
                  aria-label="Snapshot shown"
                  value={selected?.shortId}
                  onChange={(e) => {
                    setSelectedId(e.target.value);
                    setShowNdvi(true);
                  }}
                  className="appearance-none rounded-sm bg-transparent py-1.5 pr-6 pl-2 font-mono text-micro outline-none hover:bg-surface-container"
                >
                  {ndviLayers.map((l) => (
                    <option key={l.shortId} value={l.shortId}>
                      {l.label}
                    </option>
                  ))}
                </select>
                <ChevronDown className="pointer-events-none absolute top-1/2 right-1.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
              </div>
            )}
          </div>
        )}
      </div>

      {active && vis && (
        <div className="absolute bottom-3 left-3 z-10 w-56 rounded-md bg-surface-lowest/95 px-3 py-2 shadow-sm backdrop-blur">
          <div className="flex items-baseline justify-between font-mono text-micro">
            <span className="font-semibold tracking-[0.12em] uppercase">NDVI</span>
            <span className="text-muted-foreground">{active.label}</span>
          </div>
          <div
            className="mt-1.5 h-2.5 rounded-sm"
            style={{ background: `linear-gradient(to right, ${vis.palette.join(', ')})` }}
            aria-hidden
          />
          <div className="mt-1 flex justify-between font-mono text-micro text-muted-foreground">
            <span>{vis.min.toFixed(1)} bare</span>
            <span>{vis.max.toFixed(1)} dense</span>
          </div>
        </div>
      )}

      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center font-mono text-caption text-muted-foreground">
          Loading map…
        </div>
      )}
    </div>
  );
};

export default ParcelMap;
