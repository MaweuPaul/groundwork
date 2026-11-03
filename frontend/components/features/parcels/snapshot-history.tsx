import { ArrowRight, Fingerprint, LayoutList, RotateCcw } from 'lucide-react';
import Link from 'next/link';

import { retryPlan } from '@/components/features/parcels/retry-plan';
import RunSnapshotDialog from '@/components/features/parcels/run-snapshot-dialog';
import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import StatusBadge from '@/components/shared/status-badge';
import { formatDate, formatNdvi } from '@/lib/format';
import { snapshotPath } from '@/lib/paths';
import type { Methodology, SnapshotSummary } from '@/lib/types';

const columns = ['Date range', 'Methodology', 'NDVI mean', 'Valid pixels', 'Status', 'Record hash', ''];

const shortHash = (hash: string) => `${hash.slice(0, 6)}…${hash.slice(-4)}`;

// failure_message is "CODE: reason"; the code is dropped for display.
function FailureReason({ message }: { message: string }) {
  const [, ...rest] = message.split(': ');
  return <p className="mt-1.5 max-w-64 text-caption text-on-danger-container">{rest.join(': ') || message}</p>;
}

const SnapshotHistory = ({
  snapshots,
  parcelId,
  parcelName,
  methodologies,
}: {
  snapshots: SnapshotSummary[];
  parcelId: string;
  parcelName: string;
  methodologies: Methodology[];
}) => {
  const sealed = snapshots.filter((s) => s.record_hash);
  const latestSealed = sealed.find((s) => s.completed_at);

  return (
    <Border className="overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-3 p-5">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <LayoutList className="size-5" />
          Snapshots &amp; verification records
        </h2>
        <span className="font-mono text-micro tracking-[0.1em] text-muted-foreground uppercase">
          {sealed.length} sealed
          {latestSealed?.completed_at && ` · latest ${formatDate(latestSealed.completed_at)} UTC`}
        </span>
      </div>

      {snapshots.length === 0 ? (
        <p className="border-t border-border px-5 py-10 text-center text-body text-muted-foreground">
          No snapshots yet. Run the first one to start this parcel&apos;s verifiable record.
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="border-y border-border bg-surface-low">
              <tr>
                {columns.map((column) => (
                  <th
                    key={column}
                    className="px-5 py-3 font-mono text-micro font-semibold tracking-[0.12em] whitespace-nowrap text-muted-foreground uppercase"
                  >
                    {column}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {snapshots.map((s) => (
                <tr key={s.id} className="transition-colors hover:bg-surface-low">
                  <td className="px-5 py-4 font-mono text-caption whitespace-nowrap">
                    {s.date_start} → {s.date_end}
                  </td>
                  <td className="px-5 py-4">
                    <span className="rounded-sm bg-surface-container px-2 py-1 font-mono text-micro whitespace-nowrap">
                      {s.methodology} v{s.methodology_version}
                    </span>
                    <span className="mt-1.5 block font-mono text-micro whitespace-nowrap text-muted-foreground">
                      scenes ≤{s.max_scene_cloud_pct}% · clear ≥{s.cloud_score_threshold.toFixed(2)}
                    </span>
                  </td>
                  <td className="px-5 py-4 font-mono text-label font-semibold">
                    {formatNdvi(s.measurements.ndvi_mean)}
                  </td>
                  <td className="px-5 py-4 font-mono text-caption text-muted-foreground">
                    {s.measurements.valid_pixel_pct === undefined
                      ? '—'
                      : `${s.measurements.valid_pixel_pct.toFixed(0)}%`}
                  </td>
                  <td className="px-5 py-4">
                    <StatusBadge status={s.status} />
                    {s.status === 'failed' && s.failure_message && <FailureReason message={s.failure_message} />}
                    {s.status === 'failed' && (
                      <RunSnapshotDialog
                        parcelId={parcelId}
                        parcelName={parcelName}
                        methodologies={methodologies}
                        {...retryPlan(s)}
                        trigger={
                          <button
                            type="button"
                            className="mt-2 inline-flex items-center gap-1 font-mono text-micro font-semibold tracking-[0.06em] whitespace-nowrap text-primary uppercase hover:underline"
                          >
                            <RotateCcw className="size-3.5" />
                            Retry with options
                          </button>
                        }
                      />
                    )}
                  </td>
                  <td className="px-5 py-4 font-mono text-caption whitespace-nowrap text-muted-foreground">
                    {s.record_hash ? (
                      <span className="flex items-center gap-2">
                        {shortHash(s.record_hash)}
                        <CopyButton value={s.record_hash} label="Copy record hash" />
                      </span>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-5 py-4 text-right">
                    <Link
                      href={snapshotPath(s)}
                      className="inline-flex items-center gap-1 font-mono text-caption font-semibold whitespace-nowrap text-primary hover:underline"
                    >
                      Details <ArrowRight className="size-3.5" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="flex items-center gap-2 border-t border-border px-5 py-3 font-mono text-micro text-muted-foreground">
        <Fingerprint className="size-3.5" />
        Completed snapshots are sealed into a SHA-256 hash chain and can&apos;t be edited or deleted.
      </div>
    </Border>
  );
};

export default SnapshotHistory;
