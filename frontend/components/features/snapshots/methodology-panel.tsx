import { FlaskConical } from 'lucide-react';

import Border from '@/components/shared/border';
import type { Methodology, Snapshot } from '@/lib/types';

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[140px_minmax(0,1fr)] gap-3 border-b border-border py-2 last:border-0">
      <span className="text-label text-muted-foreground">{label}</span>
      <span className="font-mono text-caption wrap-break-word">{children}</span>
    </div>
  );
}

const MethodologyPanel = ({ snapshot, methodology }: { snapshot: Snapshot; methodology?: Methodology }) => {
  return (
    <Border className="flex flex-col gap-3 p-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <FlaskConical className="size-5" />
          Methodology &amp; inputs
        </h2>
        <span className="rounded-sm bg-secondary-container px-2 py-0.5 font-mono text-micro font-semibold text-on-secondary-container">
          v{snapshot.methodology_version}
        </span>
      </div>

      <div className="flex flex-col">
        <Row label="Pipeline">
          {snapshot.methodology} v{snapshot.methodology_version}
        </Row>
        {methodology && (
          <>
            <Row label="Sensor">{methodology.satellite}, Level-2A surface reflectance</Row>
            <Row label="Bands">B4 (red), B8 (near-infrared)</Row>
            <Row label="Composite">{methodology.composite_method}, NDVI = (B8 − B4) / (B8 + B4)</Row>
          </>
        )}
        <Row label="Window">
          {snapshot.date_start} → {snapshot.date_end} (UTC, inclusive)
        </Row>
        <Row label="Scene cloud limit">≤ {snapshot.max_scene_cloud_pct}% per scene</Row>
        <Row label="Clear-pixel mask">Cloud Score+ cs_cdf ≥ {snapshot.cloud_score_threshold.toFixed(2)}</Row>
        <Row label="Resolution">10 m</Row>
        <Row label="Coordinates">WGS84 (EPSG:4326)</Row>
      </div>

      <p className="text-caption text-muted-foreground">
        These inputs are frozen with the snapshot. Anyone with Earth Engine access can rerun the same methodology on
        the same boundary and window to reproduce the measurements.
      </p>
    </Border>
  );
};

export default MethodologyPanel;
