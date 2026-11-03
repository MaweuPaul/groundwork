import { ArrowRight, Fingerprint, LayoutList } from 'lucide-react';
import Link from 'next/link';

import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import StatusBadge from '@/components/shared/status-badge';
import { formatDate, formatNdvi } from '@/lib/format';
import type { SnapshotSummary } from '@/lib/types';

const columns = ['Date range', 'Methodology', 'NDVI mean', 'Valid pixels', 'Status', 'Record hash', ''];

const shortHash = (hash: string) => `${hash.slice(0, 6)}…${hash.slice(-4)}`;

const SnapshotHistory = ({ snapshots }: { snapshots: SnapshotSummary[] }) => {
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
                      href={`/snapshots/${s.id}`}
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
