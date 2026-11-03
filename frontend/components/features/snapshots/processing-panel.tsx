import { Activity, RotateCcw } from 'lucide-react';

import { retryPlan } from '@/components/features/parcels/retry-plan';
import RunSnapshotDialog from '@/components/features/parcels/run-snapshot-dialog';
import Border from '@/components/shared/border';
import StatusBadge from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import { formatUtc } from '@/lib/format';
import type { Methodology, Snapshot } from '@/lib/types';

const jobLabels: Record<string, string> = {
  queued: 'Queued',
  running: 'Running',
  succeeded: 'Succeeded',
  failed: 'Failed',
};

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-3 border-b border-border py-2 last:border-0">
      <span className="text-label text-muted-foreground">{label}</span>
      <span className="text-right font-mono text-caption">{children}</span>
    </div>
  );
}

const ProcessingPanel = ({
  snapshot,
  parcelName,
  methodologies,
}: {
  snapshot: Snapshot;
  parcelName: string;
  methodologies: Methodology[];
}) => {
  const job = snapshot.job;
  const reason = snapshot.failure_message?.split(': ').slice(1).join(': ') || snapshot.failure_message;

  return (
    <Border className="flex flex-col gap-4 p-5">
      <h2 className="flex items-center gap-2 text-h2 font-semibold">
        <Activity className="size-5" />
        Processing
      </h2>

      <div className="flex flex-col">
        <Row label="Status">
          <StatusBadge status={snapshot.status} />
        </Row>
        {job && (
          <>
            <Row label="Job">{jobLabels[job.status] ?? job.status}</Row>
            <Row label="Attempts">
              {job.attempts} of {job.max_attempts}
            </Row>
            {job.next_attempt_at && <Row label="Next attempt">{formatUtc(job.next_attempt_at)}</Row>}
            {job.last_error_code && <Row label="Last error">{job.last_error_code}</Row>}
          </>
        )}
        <Row label="Requested">{formatUtc(snapshot.created_at)}</Row>
        {snapshot.started_at && <Row label="Started">{formatUtc(snapshot.started_at)}</Row>}
        {snapshot.completed_at && <Row label="Completed">{formatUtc(snapshot.completed_at)}</Row>}
      </div>

      {snapshot.status === 'failed' && reason && (
        <p className="rounded-sm bg-danger-container px-3 py-2 text-caption text-on-danger-container">{reason}</p>
      )}

      {snapshot.status === 'failed' && (
        <RunSnapshotDialog
          parcelId={snapshot.parcel_id}
          parcelName={parcelName}
          methodologies={methodologies}
          {...retryPlan(snapshot)}
          trigger={
            <Button size="lg" variant="outline">
              <RotateCcw />
              Retry with options
            </Button>
          }
        />
      )}
    </Border>
  );
};

export default ProcessingPanel;
