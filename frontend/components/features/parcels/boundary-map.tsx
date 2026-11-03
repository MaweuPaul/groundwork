'use client';

import 'maplibre-gl/dist/maplibre-gl.css';

import { FileUp, MousePointerClick, PenLine, Spline, Trash2, X } from 'lucide-react';
import type { Map as MapLibreMap } from 'maplibre-gl';
import { useEffect, useRef, useState } from 'react';
import type { GeoJSONStoreFeatures, TerraDraw } from 'terra-draw';

import { bounds, centroid, parseBoundary, type PolygonCoords } from '@/lib/geo';
import { basemapStyle, loadMapLibre, setBasemap as applyBasemap, type Basemap } from '@/lib/map';

type Tool = 'polygon' | 'select';

const LIME = '#d9eaa3';
const WHITE = '#ffffff';
const FOREST = '#012d1d';

const hints: Record<Tool, string> = {
  polygon: 'Click the map to place boundary points. Click the first point again to close the shape.',
  select: 'Click a shape to select it. Drag the white handles to move points, or a green midpoint to add one.',
};

const drawnFeatures = (features: GeoJSONStoreFeatures[]) =>
  features.filter((f) => f.geometry.type === 'Polygon' && !f.properties.currentlyDrawing);

const BoundaryMap = ({
  polygons,
  onChange,
}: {
  polygons: PolygonCoords[];
  onChange: (polygons: PolygonCoords[]) => void;
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const drawRef = useRef<TerraDraw | null>(null);
  const onChangeRef = useRef(onChange);
  const lastEmitted = useRef('[]');

  const [ready, setReady] = useState(false);
  const [tool, setTool] = useState<Tool>('polygon');
  const [basemap, setBasemap] = useState<Basemap>('satellite');
  const [selectedId, setSelectedId] = useState<string | number | null>(null);
  const [zoom, setZoom] = useState(6);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    let cancelled = false;
    let emitTimer: ReturnType<typeof setTimeout> | undefined;

    (async () => {
      const [maplibregl, td, { TerraDrawMapLibreGLAdapter }] = await Promise.all([
        loadMapLibre(),
        import('terra-draw'),
        import('terra-draw-maplibre-gl-adapter'),
      ]);
      if (cancelled || !containerRef.current) return;

      const map = new maplibregl.Map({
        container: containerRef.current,
        center: [37.9, 0.2],
        zoom: 6,
        attributionControl: { compact: true },
        style: basemapStyle,
      });
      mapRef.current = map;
      map.addControl(new maplibregl.NavigationControl({ visualizePitch: false }), 'bottom-right');
      map.addControl(
        new maplibregl.GeolocateControl({ fitBoundsOptions: { maxZoom: 15 } }),
        'bottom-right',
      );
      map.on('zoom', () => setZoom(map.getZoom()));

      map.on('load', () => {
        if (cancelled) return;

        const draw = new td.TerraDraw({
          adapter: new TerraDrawMapLibreGLAdapter({ map }),
          modes: [
            new td.TerraDrawPolygonMode({
              validation: (feature) => td.ValidateNotSelfIntersecting(feature),
              styles: {
                fillColor: LIME,
                fillOpacity: 0.25,
                outlineColor: LIME,
                outlineWidth: 2,
                closingPointColor: FOREST,
                closingPointOutlineColor: LIME,
                closingPointWidth: 6,
                closingPointOutlineWidth: 2,
              },
            }),
            new td.TerraDrawSelectMode({
              flags: {
                polygon: {
                  feature: {
                    // Off so dragging inside a selected shape pans the map instead of moving the parcel.
                    draggable: false,
                    coordinates: { midpoints: true, draggable: true, deletable: true },
                    validation: (feature) => td.ValidateNotSelfIntersecting(feature),
                  },
                },
              },
              styles: {
                selectedPolygonColor: LIME,
                selectedPolygonFillOpacity: 0.35,
                selectedPolygonOutlineColor: WHITE,
                selectedPolygonOutlineWidth: 2,
                selectionPointColor: WHITE,
                selectionPointOutlineColor: FOREST,
                selectionPointWidth: 6,
                selectionPointOutlineWidth: 2,
                midPointColor: LIME,
                midPointOutlineColor: FOREST,
                midPointWidth: 4,
              },
            }),
          ],
        });

        const emit = () => {
          const polygons = drawnFeatures(draw.getSnapshot()).map(
            (f) => f.geometry.coordinates as PolygonCoords,
          );
          const serialised = JSON.stringify(polygons);
          if (serialised === lastEmitted.current) return;
          lastEmitted.current = serialised;
          onChangeRef.current(polygons);
        };

        // Drags fire 'update' on every pointer move. The form hears about them on 'finish', or
        // 150 ms after movement stops (a drag released outside the map never finishes),
        // so it re-renders once per edit instead of once per frame.
        draw.on('change', (_ids, type) => {
          clearTimeout(emitTimer);
          if (type === 'update') emitTimer = setTimeout(emit, 150);
          else emit();
        });
        draw.on('select', (id) => setSelectedId(id));
        draw.on('deselect', () => setSelectedId(null));
        draw.on('finish', (id, context) => {
          clearTimeout(emitTimer);
          emit();
          // A finished shape goes straight into edit mode so its handles and Delete are available.
          if (context.mode !== 'polygon' || context.action !== 'draw') return;
          draw.setMode('select');
          draw.selectFeature(id);
          setTool('select');
        });

        draw.start();
        draw.setMode('polygon');
        drawRef.current = draw;
        setReady(true);
      });
    })();

    return () => {
      cancelled = true;
      clearTimeout(emitTimer);
      drawRef.current?.stop();
      drawRef.current = null;
      mapRef.current?.remove();
      mapRef.current = null;
    };
  }, []);

  const chooseTool = (next: Tool) => {
    drawRef.current?.setMode(next);
    setTool(next);
    setSelectedId(null);
  };

  const chooseBasemap = (next: Basemap) => {
    if (!mapRef.current) return;
    applyBasemap(mapRef.current, next);
    setBasemap(next);
  };

  // Deletes the selected shape, or the most recent one when nothing is selected.
  const deleteShape = () => {
    const draw = drawRef.current;
    if (!draw) return;
    const id = selectedId ?? drawnFeatures(draw.getSnapshot()).at(-1)?.id;
    if (id === undefined || id === null) return;
    if (selectedId !== null) draw.deselectFeature(selectedId);
    draw.removeFeatures([id]);
    setSelectedId(null);
  };

  const clearAll = () => {
    drawRef.current?.clear();
    setSelectedId(null);
    setError(null);
    chooseTool('polygon');
  };

  const upload = async (file: File) => {
    const draw = drawRef.current;
    if (!draw) return;
    setError(null);

    let shapes: PolygonCoords[];
    try {
      shapes = parseBoundary(await file.text());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not read the file.');
      return;
    }

    draw.clear();
    setSelectedId(null);
    const results = draw.addFeatures(
      shapes.map((coordinates) => ({
        type: 'Feature',
        geometry: { type: 'Polygon', coordinates },
        properties: { mode: 'polygon' },
      })),
    );
    const rejected = results.filter((r) => !r.valid);
    if (rejected.length) {
      draw.clear();
      setError(`The boundary was rejected: ${rejected[0].reason ?? 'invalid polygon'}.`);
      return;
    }

    mapRef.current?.fitBounds(bounds(shapes), { padding: 60, maxZoom: 17, duration: 800 });
    chooseTool('select');
  };

  const center = centroid(polygons);
  const canDelete = ready && polygons.length > 0;

  const toolButton = (active: boolean) =>
    `flex items-center gap-1.5 rounded-sm px-2.5 py-1.5 font-mono text-micro font-semibold tracking-[0.08em] whitespace-nowrap uppercase transition-colors disabled:pointer-events-none disabled:opacity-40 ${
      active ? 'bg-primary text-primary-foreground' : 'text-foreground hover:bg-surface-container'
    }`;
  const chip =
    'rounded-sm bg-surface-lowest/95 px-2.5 py-1.5 font-mono text-micro shadow-sm backdrop-blur';

  return (
    <div className="flex flex-col gap-2">
      <div className="relative h-[calc(100svh-13rem)] min-h-130 overflow-hidden rounded-md border border-border bg-surface-container">
        {/* maplibre-gl.css forces position: relative on this element, so it's sized with h-full, not inset-0. */}
        <div ref={containerRef} className="h-full w-full" />

        <div className="pointer-events-none absolute inset-x-3 top-3 z-10 flex flex-wrap items-start justify-between gap-2 *:pointer-events-auto">
          <div className="flex flex-wrap items-center gap-1 rounded-md bg-surface-lowest p-1 shadow-sm">
            <button type="button" disabled={!ready} onClick={() => chooseTool('polygon')} className={toolButton(tool === 'polygon')}>
              <PenLine className="size-3.5" />
              Polygon draw
            </button>
            <button type="button" disabled={!ready} onClick={() => chooseTool('select')} className={toolButton(tool === 'select')}>
              <Spline className="size-3.5" />
              Edit vertices
            </button>
            <button type="button" disabled={!canDelete} onClick={deleteShape} className={toolButton(false)}>
              <Trash2 className="size-3.5" /> Delete
            </button>
            <button type="button" disabled={!ready} onClick={() => fileRef.current?.click()} className={toolButton(false)}>
              <FileUp className="size-3.5" /> Upload GeoJSON
            </button>
            <button type="button" disabled={!canDelete} onClick={clearAll} className={toolButton(false)}>
              <X className="size-3.5" /> Clear
            </button>
            <input
              ref={fileRef}
              type="file"
              accept=".geojson,.json,application/geo+json,application/json"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) upload(file);
                e.target.value = '';
              }}
            />
          </div>
          <div className="flex items-center gap-2">
            <div className={`flex items-center gap-2 font-semibold ${chip}`}>
              <span className={`size-1.5 rounded-full ${ready ? 'bg-olive' : 'bg-outline-variant'}`} />
              ZOOM {zoom.toFixed(1)}
            </div>
            <div className="flex gap-1 rounded-md bg-surface-lowest p-1 shadow-sm">
              <button type="button" onClick={() => chooseBasemap('satellite')} className={toolButton(basemap === 'satellite')}>
                Satellite
              </button>
              <button type="button" onClick={() => chooseBasemap('streets')} className={toolButton(basemap === 'streets')}>
                Streets
              </button>
            </div>
          </div>
        </div>

        {error && (
          <div
            role="alert"
            className="absolute inset-x-3 top-28 z-10 flex items-start justify-between gap-3 rounded-md border border-danger/20 bg-danger-container px-3 py-2 text-label text-on-danger-container"
          >
            {error}
            <button type="button" onClick={() => setError(null)} aria-label="Dismiss">
              <X className="size-4" />
            </button>
          </div>
        )}

        <div className={`pointer-events-none absolute bottom-3 left-3 z-10 ${chip}`}>
          <span className="block tracking-[0.12em] text-muted-foreground uppercase">
            Centroid coordinates
          </span>
          <span className="font-semibold">
            {center ? `LAT ${center[1].toFixed(4)}°, LON ${center[0].toFixed(4)}°` : '—'}
          </span>
        </div>

        {!ready && (
          <div className="absolute inset-0 flex items-center justify-center font-mono text-caption text-muted-foreground">
            Loading map…
          </div>
        )}
      </div>

      <p className="flex items-center gap-2 font-mono text-caption text-muted-foreground">
        <MousePointerClick className="size-3.5 shrink-0" />
        {hints[tool]}
      </p>
    </div>
  );
};

export default BoundaryMap;
