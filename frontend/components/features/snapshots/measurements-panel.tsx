import { Gauge, TriangleAlert } from 'lucide-react';

import Border from '@/components/shared/border';
import { formatMeasurement, methodologyView, orderedSlugs } from '@/lib/methodologies';
import type { Snapshot } from '@/lib/types';

const tileLabel = 'font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase';

const MeasurementsPanel = ({ snapshot }: { snapshot: Snapshot }) => {
  const m = snapshot.measurements;
  const view = methodologyView(snapshot.methodology);
  const Icon = view?.icon ?? Gauge;
  const slugs = orderedSlugs(snapshot.methodology, m);
  const valid = m.valid_pixel_pct;
  const vis = snapshot.ndvi_image?.visualization;
  const minClear = snapshot.parameters?.min_clear_pct;
  const lowCoverage = snapshot.flags?.find((f) => f.flag === 'low_coverage');

  return (
    <Border className="flex flex-col gap-4 p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Icon className="size-5" />
          {view?.title ?? 'Measurements'}
        </h2>
        {view && <span className="font-mono text-micro text-muted-foreground">{view.scale}</span>}
      </div>

      {slugs.length === 0 ? (
        <p className="rounded-sm bg-surface-low px-4 py-6 text-center text-label text-muted-foreground">
          {snapshot.status === 'failed' ? 'No measurements: the snapshot failed.' : 'Measurements appear when the snapshot completes.'}
        </p>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            {slugs.map((slug) => {
              const type = snapshot.measurement_types?.[slug];
              return (
                <div key={slug} className="rounded-sm bg-surface-low p-4">
                  <span className={tileLabel}>{type?.name ?? slug.replaceAll('_', ' ')}</span>
                  <span className="mt-1 block font-mono text-[32px] leading-none font-semibold">
                    {formatMeasurement(m[slug], type?.unit)}
                  </span>
                  {type?.description && (
                    <span className="mt-1 block text-caption text-muted-foreground">{type.description}</span>
                  )}
                </div>
              );
            })}
          </div>

          {valid !== undefined && (
            <div className="flex flex-col gap-1.5">
              <div className="flex justify-between font-mono text-micro text-muted-foreground">
                <span>Valid-pixel coverage</span>
                <span>
                  {valid.toFixed(1)}%{minClear !== undefined && ` · minimum ${minClear}%`}
                </span>
              </div>
              <div className="relative h-2 rounded-full bg-surface-high">
                <div
                  className={`h-full rounded-full ${lowCoverage ? 'bg-on-warning-container' : 'bg-primary-container'}`}
                  style={{ width: `${Math.min(100, valid)}%` }}
                />
                {minClear !== undefined && (
                  <div
                    className="absolute -top-1 h-4 w-0.5 bg-foreground"
                    style={{ left: `${Math.min(100, minClear)}%` }}
                    title={`Minimum clear pixels: ${minClear}%`}
                  />
                )}
              </div>
            </div>
          )}

          {lowCoverage && (
            <div className="flex gap-2 rounded-sm bg-warning-container p-3 text-label text-on-warning-container">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <p>
                <span className="font-semibold">Low coverage.</span> {lowCoverage.detail} The values are from the clear
                part of the parcel only, so treat them with care.
              </p>
            </div>
          )}
        </>
      )}

      {vis && (
        <div className="flex flex-col gap-1.5">
          <span className="font-mono text-micro text-muted-foreground">NDVI scale on the map</span>
          <div
            className="h-2.5 rounded-sm"
            style={{ background: `linear-gradient(to right, ${vis.palette.join(', ')})` }}
            aria-hidden
          />
          <div className="flex justify-between font-mono text-micro text-muted-foreground">
            <span>{vis.min.toFixed(1)} bare ground</span>
            <span>{vis.max.toFixed(1)} dense vegetation</span>
          </div>
        </div>
      )}
    </Border>
  );
};

export default MeasurementsPanel;
