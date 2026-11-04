'use client';

import { ChevronDown, ClipboardList, MapPin, ShieldCheck } from 'lucide-react';
import Link from 'next/link';
import { useActionState, useState } from 'react';

import { createParcelAction, type ParcelFormState } from '@/app/(app)/parcels/new/actions';
import BoundaryMap from '@/components/features/parcels/boundary-map';
import Border from '@/components/shared/border';
import { Button } from '@/components/ui/button';
import { measure, toGeometry, type PolygonCoords } from '@/lib/geo';

const initialState: ParcelFormState = {};

const labelClass = 'font-mono text-micro font-semibold tracking-widest text-foreground';
const inputClass =
  'w-full rounded-sm border border-transparent bg-surface-container px-3 py-2.5 text-body outline-none transition-colors placeholder:text-outline focus:border-primary';

function formatPerimeter(m: number) {
  return m >= 1000 ? `${(m / 1000).toFixed(2)} km` : `${m.toFixed(1)} m`;
}

const RegisterParcelForm = () => {
  const [state, formAction, pending] = useActionState(createParcelAction, initialState);
  const [polygons, setPolygons] = useState<PolygonCoords[]>([]);

  const geometry = toGeometry(polygons);
  const stats = measure(polygons);
  const vertices = polygons.flatMap((polygon) => polygon[0].slice(0, -1));
  const geometryError = !geometry ? state.fieldErrors?.geometry : undefined;

  const metrics = [
    { label: 'Surface area', value: `${stats.areaHa.toLocaleString('en-US', { maximumFractionDigits: 2 })} ha`, large: true },
    { label: 'Perimeter', value: formatPerimeter(stats.perimeterM) },
    { label: 'Vertices', value: `${stats.vertices} nodes${stats.pieces > 1 ? ` · ${stats.pieces} pieces` : ''}` },
    { label: 'CRS standard', value: 'WGS84 / EPSG:4326' },
  ];

  return (
    <form action={formAction} noValidate className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
      <BoundaryMap polygons={polygons} onChange={setPolygons} />

      <Border className="flex h-fit flex-col gap-4 p-5">
        <div className="flex items-center justify-between gap-2">
          <h2 className="flex items-center gap-2 text-h2 font-semibold">
            <ClipboardList className="size-5" />
            Cadastral record
          </h2>
          <span className="rounded-sm bg-surface-container px-2 py-0.5 font-mono text-micro text-muted-foreground">
            DRAFT
          </span>
        </div>

        {state.error && (
          <p
            role="alert"
            className="rounded-sm border border-danger/20 bg-danger-container px-3 py-2 text-label text-on-danger-container"
          >
            {state.error}
          </p>
        )}

        <div className="flex flex-col gap-1.5">
          <div className="flex items-center justify-between">
            <label htmlFor="name" className={labelClass}>
              Parcel identifier name
            </label>
            <span className="font-mono text-micro text-olive">REQUIRED</span>
          </div>
          <input
            id="name"
            name="name"
            required
            maxLength={200}
            placeholder="e.g. Nyeri Block A"
            defaultValue={state.values?.name}
            aria-invalid={state.fieldErrors?.name ? true : undefined}
            className={`${inputClass} ${state.fieldErrors?.name ? 'border-danger' : ''}`}
          />
          {state.fieldErrors?.name && <p className="text-caption text-danger">{state.fieldErrors.name}</p>}
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="region" className={labelClass}>
            Region / country jurisdiction
          </label>
          <input
            id="region"
            name="region"
            maxLength={200}
            placeholder="e.g. Nyeri County, Kenya"
            defaultValue={state.values?.region}
            className={inputClass}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="description" className={labelClass}>
            Description &amp; registry notes
          </label>
          <textarea
            id="description"
            name="description"
            rows={3}
            placeholder="Title reference, history, anything an auditor should know."
            defaultValue={state.values?.description}
            className={`${inputClass} resize-none`}
          />
        </div>

        <input type="hidden" name="geometry" value={geometry ? JSON.stringify(geometry) : ''} />

        <div className="flex flex-col gap-3 rounded-md bg-surface-container p-3">
          <div className="flex items-center justify-between px-1">
            <span className="font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase">
              Live computed boundary
            </span>
            <span className={`size-2 rounded-full ${geometry ? 'bg-olive' : 'bg-outline-variant'}`} aria-hidden />
          </div>
          <div className="grid grid-cols-2 gap-2">
            {metrics.map((metric) => (
              <div key={metric.label} className="rounded-sm bg-surface-lowest px-3 py-2.5">
                <span className="block font-mono text-micro tracking-widest text-muted-foreground uppercase">
                  {metric.label}
                </span>
                <span className={`font-mono font-semibold ${metric.large ? 'text-h3' : 'text-caption'}`}>
                  {metric.value}
                </span>
              </div>
            ))}
          </div>

          <details className="group px-1">
            <summary className="flex cursor-pointer list-none items-center justify-between font-mono text-micro tracking-[0.08em] text-muted-foreground uppercase">
              <span className="flex items-center gap-1.5">
                <MapPin className="size-3.5" />
                Vertex coordinates ({vertices.length})
              </span>
              <ChevronDown className="size-3.5 transition-transform group-open:rotate-180" />
            </summary>
            <ol className="mt-2 max-h-40 overflow-y-auto font-mono text-micro text-muted-foreground">
              {vertices.length === 0 && <li>No points yet.</li>}
              {vertices.map(([lon, lat], i) => (
                <li key={i} className="flex justify-between py-0.5">
                  <span>P{i + 1}</span>
                  <span>
                    {lat.toFixed(6)}, {lon.toFixed(6)}
                  </span>
                </li>
              ))}
            </ol>
          </details>

          <p className="px-1 text-caption text-muted-foreground">
            Estimated in the browser. The official area is computed by PostGIS when you save.
          </p>
          {geometryError && <p className="px-1 text-caption text-danger">{geometryError}</p>}
        </div>

        <div className="flex gap-3 rounded-md bg-surface-container p-3 text-caption text-muted-foreground">
          <ShieldCheck className="size-4 shrink-0 text-olive" />
          <p>
            <span className="font-semibold text-foreground">Critical audit notice:</span> boundary edits
            never change past snapshots. Each snapshot locks the exact boundary used at compute time.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <Button type="submit" size="lg" disabled={pending || !geometry} className="h-11 text-body">
            <ShieldCheck />
            {pending ? 'Saving…' : 'Save parcel'}
          </Button>
          <Button asChild variant="secondary" size="lg" className="h-10">
            <Link href="/">Cancel</Link>
          </Button>
        </div>
      </Border>
    </form>
  );
};

export default RegisterParcelForm;
