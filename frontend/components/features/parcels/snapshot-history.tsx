import { ArrowRight, Fingerprint, LayoutList, RotateCcw } from 'lucide-react';
import Link from 'next/link';

import RunSnapshotDialog, { type SnapshotOptions } from '@/components/features/parcels/run-snapshot-dialog';
import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import StatusBadge from '@/components/shared/status-badge';
import { formatDate, formatNdvi } from '@/lib/format';
import { snapshotPath } from '@/lib/paths';
import type { Methodology, SnapshotSummary } from '@/lib/types';

const columns = ['Date range', 'Methodology', 'NDVI mean', 'Valid pixels', 'Status', 'Record hash', ''];

const shortHash = (hash: string) => `${hash.slice(0, 6)}…${hash.slice(-4)}`;

// failure_message is "CODE: reason".
const failureCode = (message?: string) => message?.split(': ')[0];

function FailureReason({ message }: { message: string }) {
  const [, ...rest] = message.split(': ');
  return <p className="mt-1.5 max-w-64 text-caption text-on-danger-container">{rest.join(': ') || message}</p>;
}

const MIN_WINDOW_DAYS = 30;
const EARLIEST = '2015-06-23';

const utc = (day: string) => new Date(`${day}T00:00:00Z`);

function windowDays(dateStart: string, dateEnd: string) {
  return (utc(dateEnd).getTime() - utc(dateStart).getTime()) / 86_400_000 + 1;
}

function daysBefore(day: string, days: number) {
  const d = utc(day);
  d.setUTCDate(d.getUTCDate() - days);
  const result = d.toISOString().slice(0, 10);
  return result < EARLIEST ? EARLIEST : result;
}

// Starts a short window 30 days before its end, so more satellite passes are included.
function widened(dateStart: string, dateEnd: string) {
  if (windowDays(dateStart, dateEnd) >= MIN_WINDOW_DAYS) return dateStart;
  return daysBefore(dateEnd, MIN_WINDOW_DAYS - 1);
}

// What to change for the retry, based on why the snapshot failed. Data failures get looser
// settings; anything else retries with the same ones.
function retryPlan(s: SnapshotSummary): { initial: SnapshotOptions; note: string } {
  const same: SnapshotOptions = {
    dateStart: s.date_start,
    dateEnd: s.date_end,
    maxSceneCloudPct: s.max_scene_cloud_pct,
    cloudScoreThreshold: s.cloud_score_threshold,
  };
  const code = failureCode(s.failure_message);
  if (code === 'COMPUTATION_TOO_LARGE') {
    const days = windowDays(s.date_start, s.date_end);
    const dateStart = daysBefore(s.date_end, MIN_WINDOW_DAYS - 1);
    return {
      initial: { ...same, dateStart: dateStart > s.date_start ? dateStart : s.date_start },
      note: `The ${days}-day window was too much imagery for one Earth Engine request. Suggested: the last ${MIN_WINDOW_DAYS} days of it. Run more short snapshots to cover the rest of the period.`,
    };
  }
  if (code !== 'NO_IMAGERY' && code !== 'NO_CLEAR_PIXELS') {
    return { initial: same, note: 'Prefilled with the failed attempt’s settings. Adjust any of them before starting.' };
  }

  const cloudPct = Math.max(60, Math.min(100, s.max_scene_cloud_pct * 2));
  const dateStart = widened(s.date_start, s.date_end);
  const changes = [`scenes up to ${cloudPct}% cloud (was ${s.max_scene_cloud_pct}%)`];
  if (dateStart !== s.date_start) changes.push(`a ${MIN_WINDOW_DAYS}-day window`);

  if (code === 'NO_CLEAR_PIXELS') {
    changes.push('the lenient clear-pixel mask');
    return {
      initial: { ...same, dateStart, maxSceneCloudPct: cloudPct, cloudScoreThreshold: 0.5 },
      note: `Every scene was cloudy over the parcel. Suggested: ${changes.join(', ')}.`,
    };
  }
  return {
    initial: { ...same, dateStart, maxSceneCloudPct: cloudPct },
    note: `No scene passed the ${s.max_scene_cloud_pct}% cloud limit. Suggested: ${changes.join(', ')}. Clouds over the parcel are still masked pixel by pixel.`,
  };
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
