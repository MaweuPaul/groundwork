'use client';

import 'maplibre-gl/dist/maplibre-gl.css';

import { ChevronDown, Pause, Play } from 'lucide-react';
import type { ImageSource, Map as MapLibreMap } from 'maplibre-gl';
import { useEffect, useMemo, useRef, useState } from 'react';

import { bounds } from '@/lib/geo';
import { basemapStyle, loadMapLibre, setBasemap as applyBasemap, type Basemap } from '@/lib/map';
import type { MultiPolygon, NdviImage } from '@/lib/types';

const LIME = '#d9eaa3';
const FRAME_MS = 1200;

export type NdviLayer = {
  shortId: string;
  label: string;
  dateEnd: string;
  // Share of the parcel that was cloud-free; a mostly blank map is usually cloud.
  clearPct?: number;
  image: NdviImage;
};

const imageUrl = (layer: NdviLayer) => `/api/snapshots/${layer.shortId}/ndvi`;

function preload(url: string) {
  return new Promise<void>((resolve) => {
    const img = new Image();
    img.onload = img.onerror = () => resolve();
    img.src = url;
  });
}

type Props = {
  geometry: MultiPolygon;
  ndviLayers: NdviLayer[];
  // Called with the window end of the image on the map, or undefined when none shows.
  onFrameChange?: (dateEnd: string | undefined) => void;
};

const ParcelMap = ({ geometry, ndviLayers, onFrameChange }: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const [ready, setReady] = useState(false);
  const [basemap, setBasemap] = useState<Basemap>('satellite');
  const [showNdvi, setShowNdvi] = useState(true);
  const [index, setIndex] = useState<number>();
  const [playing, setPlaying] = useState(false);
  const [loadingFrames, setLoadingFrames] = useState(false);

  const frames = useMemo(() => [...ndviLayers].sort((a, b) => a.dateEnd.localeCompare(b.dateEnd)), [ndviLayers]);
  const last = frames.length - 1;
  // The newest until one is chosen; clamped when a refresh brings fewer frames.
  const position = Math.min(index ?? last, last);
  const current = frames[position];
  const active = showNdvi ? current : undefined;

  useEffect(() => onFrameChange?.(active?.dateEnd), [active, onFrameChange]);

  // The last frame is held for one interval before playing stops, like the others.
  useEffect(() => {
    if (!playing) return;
    const timer = setTimeout(() => {
      if (position >= last) setPlaying(false);
      else setIndex(position + 1);
    }, FRAME_MS);
    return () => clearTimeout(timer);
  }, [playing, position, last]);

  // Every frame is loaded before playing starts, so none flashes in half-drawn.
  const play = async () => {
    setIndex(position >= last ? 0 : position);
    setShowNdvi(true);
    setLoadingFrames(true);
    await Promise.all(frames.map((f) => preload(imageUrl(f))));
    setLoadingFrames(false);
    setPlaying(true);
  };

  const choose = (i: number) => {
    setPlaying(false);
    setIndex(i);
    setShowNdvi(true);
  };

  // Keyed on the boundary's shape, not the object: a page refresh passes an equal copy, and
  // rebuilding the map for it would throw away the view.
  const shape = JSON.stringify(geometry);

  useEffect(() => {
    let cancelled = false;
    const geometry = JSON.parse(shape) as MultiPolygon;

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
      // The next map has no layers until it loads.
      setReady(false);
    };
  }, [shape]);

  // The NDVI image sits under the boundary outline. The boundary's tint is turned off while
  // it shows, so the colours aren't shifted. A new frame replaces the image in place, so the
  // map doesn't flash empty between frames.
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !ready) return;

    map.setPaintProperty('parcel-fill', 'fill-opacity', active ? 0 : 0.2);
    const source = map.getSource<ImageSource>('ndvi');
    if (!active) {
      if (map.getLayer('ndvi')) map.removeLayer('ndvi');
      if (source) map.removeSource('ndvi');
      return;
    }

    const [west, south, east, north] = active.image.bounds;
    const image = {
      url: imageUrl(active),
      coordinates: [
        [west, north],
        [east, north],
        [east, south],
        [west, south],
      ] as [[number, number], [number, number], [number, number], [number, number]],
    };
    if (source) {
      source.updateImage(image);
      return;
    }
    map.addSource('ndvi', { type: 'image', ...image });
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
    <div className="flex flex-1 flex-col gap-3">
      <div className="relative min-h-90 flex-1 overflow-hidden rounded-md border border-border bg-surface-container">
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
              {frames.length > 1 && (
                <div className="relative">
                  <select
                    aria-label="Snapshot shown"
                    value={current?.shortId}
                    disabled={!ready}
                    onChange={(e) => choose(frames.findIndex((f) => f.shortId === e.target.value))}
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

      {frames.length > 1 && current && (
        <div className="flex items-center gap-3">
          <button
            type="button"
            disabled={!ready || loadingFrames}
            onClick={playing ? () => setPlaying(false) : play}
            aria-label={playing ? 'Pause' : 'Play the NDVI maps in order'}
            className="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground transition-colors hover:bg-primary-container disabled:opacity-50"
          >
            {playing ? <Pause className="size-4" /> : <Play className="size-4" />}
          </button>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <input
              type="range"
              min={0}
              max={last}
              value={position}
              disabled={!ready}
              onChange={(e) => choose(Number(e.target.value))}
              aria-label="NDVI map shown"
              aria-valuetext={current.label}
              className="w-full accent-primary"
            />
            <div className="flex justify-between gap-2 font-mono text-micro text-muted-foreground">
              <span className="truncate">
                  {loadingFrames ? 'Loading maps…' : current.label}
                  {!loadingFrames && current.clearPct !== undefined && ` · ${current.clearPct.toFixed(0)}% clear`}
                </span>
              <span className="shrink-0">
                {position + 1} / {frames.length}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ParcelMap;
