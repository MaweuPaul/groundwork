'use client';

import { CircleCheck, CircleX, ShieldCheck, X } from 'lucide-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useRef, useState } from 'react';

import { flagLabel } from '@/components/shared/flag-badge';
import { formatNdvi } from '@/lib/format';
import { parcelPath, snapshotPath } from '@/lib/paths';
import type { ActivityEvent } from '@/lib/types';

// Fired by anything that queues work, so polling speeds up straight away.
export const ACTIVITY_CHANGED = 'groundwork:activity-changed';

const BUSY_MS = 10_000;
const IDLE_MS = 60_000;
const TOAST_MS = 8_000;
const MAX_TOASTS = 3;

const FINISHED: ActivityEvent['kind'][] = ['snapshot_completed', 'snapshot_failed', 'series_sealed'];

type Toast = { key: string; event: ActivityEvent };

const keyOf = (e: ActivityEvent) => `${e.kind}:${e.snapshot_short_id ?? e.series_short_id ?? e.parcel_id}:${e.at}`;

// A series is still running while its started event has no sealed event in the feed.
function busy(events: ActivityEvent[]) {
  const sealed = new Set(events.filter((e) => e.kind === 'series_sealed').map((e) => e.series_short_id));
  return events.some(
    (e) => e.kind === 'snapshot_queued' || (e.kind === 'series_started' && !sealed.has(e.series_short_id)),
  );
}

const reason = (message?: string) => message?.split(': ').slice(1).join(': ') || message;

function ToastBody({ e }: { e: ActivityEvent }) {
  const parcel = parcelPath({ name: e.parcel_name, short_id: e.parcel_short_id });
  if (e.kind === 'snapshot_failed') {
    return (
      <>
        <p className="font-semibold">Snapshot failed</p>
        <p className="text-caption opacity-90">
          {e.parcel_name}: {reason(e.detail) ?? 'no reason given'}
        </p>
        <Link
          href={e.snapshot_short_id ? snapshotPath({ short_id: e.snapshot_short_id }) : parcel}
          className="w-fit font-mono text-micro underline"
        >
          Open and retry
        </Link>
      </>
    );
  }
  if (e.kind === 'series_sealed') {
    return (
      <>
        <p className="font-semibold">Series sealed</p>
        <p className="text-caption opacity-90">
          {e.windows} windows for {e.parcel_name} are finished and sealed with one hash.
        </p>
        <Link href={parcel} className="w-fit font-mono text-micro underline">
          View series
        </Link>
      </>
    );
  }
  return (
    <>
      <p className="font-semibold">Snapshot sealed</p>
      <p className="text-caption opacity-90">
        {e.parcel_name}
        {e.ndvi_mean !== undefined && (
          <>
            {' '}
            · mean NDVI <span className="font-mono">{formatNdvi(e.ndvi_mean)}</span>
          </>
        )}
        {e.flags?.length ? ` · ${e.flags.map(flagLabel).join(', ')}` : ''}
      </p>
      {e.snapshot_short_id && (
        <Link href={snapshotPath({ short_id: e.snapshot_short_id })} className="w-fit font-mono text-micro underline">
          View snapshot
        </Link>
      )}
    </>
  );
}

const icons = { snapshot_completed: CircleCheck, snapshot_failed: CircleX, series_sealed: ShieldCheck } as const;

const SnapshotNotifier = () => {
  const router = useRouter();
  const [toasts, setToasts] = useState<Toast[]>([]);
  const seen = useRef<Set<string> | null>(null);

  const dismiss = useCallback((key: string) => setToasts((all) => all.filter((t) => t.key !== key)), []);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;

    async function poll() {
      clearTimeout(timer);
      let events: ActivityEvent[] = [];
      try {
        const res = await fetch('/api/activity', { cache: 'no-store' });
        if (res.ok) events = await res.json();
      } catch {
        // Offline or API down: try again on the idle schedule.
      }
      if (stopped) return;

      // The first poll only learns what already happened; it never toasts history.
      if (seen.current === null) {
        seen.current = new Set(events.map(keyOf));
      } else {
        const known = seen.current;
        const fresh = events.filter((e) => !known.has(keyOf(e)));
        fresh.forEach((e) => known.add(keyOf(e)));
        const finished = fresh.filter((e) => FINISHED.includes(e.kind));
        if (finished.length) {
          setToasts((all) => [...finished.map((event) => ({ key: keyOf(event), event })), ...all].slice(0, MAX_TOASTS));
        }
        if (fresh.length) router.refresh();
      }

      clearTimeout(timer);
      if (document.visibilityState === 'visible') {
        timer = setTimeout(poll, busy(events) ? BUSY_MS : IDLE_MS);
      }
    }

    const onVisible = () => document.visibilityState === 'visible' && poll();
    timer = setTimeout(poll, 0);
    document.addEventListener('visibilitychange', onVisible);
    window.addEventListener(ACTIVITY_CHANGED, poll);
    return () => {
      stopped = true;
      clearTimeout(timer);
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener(ACTIVITY_CHANGED, poll);
    };
  }, [router]);

  useEffect(() => {
    if (!toasts.length) return;
    const oldest = toasts.at(-1)!;
    const t = setTimeout(() => dismiss(oldest.key), TOAST_MS);
    return () => clearTimeout(t);
  }, [toasts, dismiss]);

  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed right-4 bottom-4 z-50 flex w-[min(380px,calc(100vw-2rem))] flex-col gap-2"
    >
      {toasts.map(({ key, event }) => {
        const Icon = icons[event.kind as keyof typeof icons];
        const failed = event.kind === 'snapshot_failed';
        return (
          <div
            key={key}
            role="status"
            className={`pointer-events-auto flex gap-3 rounded-md p-4 shadow-lg ${
              failed ? 'bg-danger-container text-on-danger-container' : 'bg-primary-container text-primary-foreground'
            }`}
          >
            <Icon className="mt-0.5 size-5 shrink-0" />
            <div className="flex min-w-0 flex-1 flex-col gap-1 text-label">
              <ToastBody e={event} />
            </div>
            <div className="flex flex-col items-end gap-2">
              <button
                type="button"
                onClick={() => dismiss(key)}
                aria-label="Dismiss"
                className="opacity-80 hover:opacity-100"
              >
                <X className="size-4" />
              </button>
              <span className="font-mono text-micro font-semibold tracking-[0.12em] opacity-80">LIVE</span>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default SnapshotNotifier;
