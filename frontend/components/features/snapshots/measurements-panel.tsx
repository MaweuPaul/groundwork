import { Sprout } from 'lucide-react';

import Border from '@/components/shared/border';
import type { Snapshot } from '@/lib/types';

const tileLabel = 'font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase';

const MeasurementsPanel = ({ snapshot }: { snapshot: Snapshot }) => {
  const m = snapshot.measurements;
  const valid = m.valid_pixel_pct;
  const vis = snapshot.ndvi_image?.visualization;

  return (
    <Border className="flex flex-col gap-4 p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Sprout className="size-5" />
          Vegetation measurements
        </h2>
        <span className="font-mono text-micro text-muted-foreground">10 m pixels · median composite</span>
      </div>

      {m.ndvi_mean === undefined ? (
        <p className="rounded-sm bg-surface-low px-4 py-6 text-center text-label text-muted-foreground">
          {snapshot.status === 'failed' ? 'No measurements: the snapshot failed.' : 'Measurements appear when the snapshot completes.'}
        </p>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-sm bg-surface-low p-4">
              <span className={tileLabel}>Mean NDVI</span>
              <span className="mt-1 block font-mono text-[32px] leading-none font-semibold">{m.ndvi_mean.toFixed(3)}</span>
              <span className="mt-1 block text-caption text-muted-foreground">Average over clear pixels</span>
            </div>
            <div className="rounded-sm bg-surface-low p-4">
              <span className={tileLabel}>Max NDVI</span>
              <span className="mt-1 block font-mono text-[32px] leading-none font-semibold">
                {m.ndvi_max?.toFixed(3) ?? '—'}
              </span>
              <span className="mt-1 block text-caption text-muted-foreground">Densest pixel in the parcel</span>
            </div>
            <div className="rounded-sm bg-surface-low p-4">
              <span className={tileLabel}>Clear pixels</span>
              <span className="mt-1 block font-mono text-[32px] leading-none font-semibold">
                {valid === undefined ? '—' : `${valid.toFixed(1)}%`}
              </span>
              <span className="mt-1 block text-caption text-muted-foreground">Of the parcel, after cloud masking</span>
            </div>
          </div>

          {valid !== undefined && (
            <div className="flex flex-col gap-1.5">
              <div className="flex justify-between font-mono text-micro text-muted-foreground">
                <span>Valid-pixel coverage</span>
                <span>{valid.toFixed(1)}%</span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-surface-high">
                <div className="h-full rounded-full bg-primary-container" style={{ width: `${Math.min(100, valid)}%` }} />
              </div>
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
