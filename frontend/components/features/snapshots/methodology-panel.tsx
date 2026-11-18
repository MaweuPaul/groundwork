import { FlaskConical } from 'lucide-react';

import Border from '@/components/shared/border';
import { methodologyView } from '@/lib/methodologies';
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
        {methodologyView(snapshot.methodology)?.inputs.map(([label, value]) => (
          <Row key={label} label={label}>
            {value}
          </Row>
        ))}
        <Row label="Window">
          {snapshot.date_start} → {snapshot.date_end} (UTC, inclusive)
        </Row>
        {Object.entries(snapshot.parameters ?? {})
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([name, value]) => (
            <Row key={name} label={methodology?.parameters.find((p) => p.name === name)?.label ?? name}>
              {value}
            </Row>
          ))}
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
