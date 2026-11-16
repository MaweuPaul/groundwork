import { Activity, Fingerprint } from 'lucide-react';
import Link from 'next/link';

import Border from '@/components/shared/border';
import { flagLabel } from '@/components/shared/flag-badge';
import { formatNdvi } from '@/lib/format';
import { parcelPath, snapshotPath } from '@/lib/paths';
import type { ActivityEvent } from '@/lib/types';

const shortHash = (hash: string) => `${hash.slice(0, 6)}…${hash.slice(-4)}`;

const dot: Record<ActivityEvent['kind'], string> = {
  parcel_registered: 'bg-outline',
  snapshot_queued: 'bg-olive',
  snapshot_completed: 'bg-primary',
  snapshot_failed: 'bg-danger',
  series_started: 'bg-olive',
  series_sealed: 'bg-primary',
};

// "Today 14:22 UTC", "Yesterday 09:15 UTC", else "2026-11-03 21:20 UTC". Days are UTC days.
function when(iso: string, now = new Date()) {
  const at = new Date(iso);
  const day = (d: Date) => d.toISOString().slice(0, 10);
  const time = at.toISOString().slice(11, 16);
  const yesterday = new Date(now);
  yesterday.setUTCDate(yesterday.getUTCDate() - 1);
  if (day(at) === day(now)) return `Today ${time} UTC`;
  if (day(at) === day(yesterday)) return `Yesterday ${time} UTC`;
  return `${day(at)} ${time} UTC`;
}

// failure_message is "CODE: reason"; only the reason is shown.
const reason = (message?: string) => message?.split(': ').slice(1).join(': ') || message;

function Sentence({ e }: { e: ActivityEvent }) {
  const parcel = (
    <Link
      href={parcelPath({ name: e.parcel_name, short_id: e.parcel_short_id })}
      className="font-semibold hover:underline"
    >
      {e.parcel_name}
    </Link>
  );
  switch (e.kind) {
    case 'parcel_registered':
      return <>Registered {parcel}.</>;
    case 'snapshot_queued':
      return <>Snapshot queued for {parcel}.</>;
    case 'snapshot_completed':
      return (
        <>
          Snapshot completed for {parcel}.
          {e.ndvi_mean !== undefined && (
            <>
              {' '}
              Mean NDVI <span className="font-mono font-semibold">{formatNdvi(e.ndvi_mean)}</span> sealed.
            </>
          )}
        </>
      );
    case 'snapshot_failed':
      return (
        <>
          {e.series_short_id ? 'A series window' : 'Snapshot'} failed for {parcel}.
          {e.detail && <span className="mt-0.5 block text-caption text-on-danger-container">{reason(e.detail)}</span>}
        </>
      );
    case 'series_started':
      return (
        <>
          Series of {e.windows} snapshots started for {parcel}.
        </>
      );
    case 'series_sealed':
      return (
        <>
          Series of {e.windows} windows sealed for {parcel}.
        </>
      );
  }
}

const RecentActivity = ({ events, error }: { events: ActivityEvent[]; error?: string }) => {
  return (
    <Border className="flex flex-col">
      <div className="flex items-center justify-between gap-2 border-b border-border px-5 py-4">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Activity className="size-5" />
          Recent activity
        </h2>
        <span className="font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase">UTC</span>
      </div>

      {error ? (
        <p className="px-5 py-8 text-center text-label text-muted-foreground">Couldn&apos;t load activity.</p>
      ) : events.length === 0 ? (
        <p className="px-5 py-8 text-center text-label text-muted-foreground">Nothing yet.</p>
      ) : (
        <ol className="flex flex-col px-5 py-4">
          {events.map((e, i) => (
            <li
              key={`${e.kind}-${e.snapshot_short_id ?? e.series_short_id ?? e.parcel_id}-${e.at}`}
              className="relative flex gap-3 pb-5 last:pb-0"
            >
              {i < events.length - 1 && (
                <span className="absolute top-3 bottom-0 left-[4.5px] w-px bg-border" aria-hidden />
              )}
              <span className={`relative mt-1.5 size-2.5 shrink-0 rounded-full ${dot[e.kind]}`} aria-hidden />
              <div className="flex min-w-0 flex-col gap-1">
                <span className="font-mono text-micro text-muted-foreground">{when(e.at)}</span>
                <p className="text-label">
                  <Sentence e={e} />
                </p>
                {e.flags?.map((f) => (
                  <span
                    key={f}
                    className="w-fit rounded-sm bg-warning-container px-1.5 py-0.5 font-mono text-micro font-semibold text-on-warning-container uppercase"
                  >
                    {flagLabel(f)}
                  </span>
                ))}
                {(e.record_hash || e.snapshot_short_id) && (
                  <span className="flex items-center gap-1.5 font-mono text-micro text-muted-foreground">
                    {e.record_hash && <Fingerprint className="size-3" />}
                    {e.record_hash && shortHash(e.record_hash)}
                    {e.snapshot_short_id && (
                      <Link
                        href={snapshotPath({ short_id: e.snapshot_short_id })}
                        className="text-primary hover:underline"
                      >
                        {e.record_hash ? '· ' : ''}snapshot {e.snapshot_short_id}
                      </Link>
                    )}
                  </span>
                )}
              </div>
            </li>
          ))}
        </ol>
      )}
    </Border>
  );
};

export default RecentActivity;
