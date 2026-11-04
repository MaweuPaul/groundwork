import { ArrowLeft, Map as MapIcon } from 'lucide-react';
import Link from 'next/link';
import { notFound, redirect } from 'next/navigation';

import ParcelMap from '@/components/features/parcels/parcel-map';
import ImageryTable from '@/components/features/snapshots/imagery-table';
import IntegrityPanel from '@/components/features/snapshots/integrity-panel';
import MeasurementsPanel from '@/components/features/snapshots/measurements-panel';
import MethodologyPanel from '@/components/features/snapshots/methodology-panel';
import ProcessingPanel from '@/components/features/snapshots/processing-panel';
import Border from '@/components/shared/border';
import LiveRefresh from '@/components/shared/live-refresh';
import StatusBadge from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import { getMethodologies, getParcel, getSnapshot } from '@/lib/api';
import { formatUtc } from '@/lib/format';
import { parcelPath, snapshotPath } from '@/lib/paths';
import type { Snapshot } from '@/lib/types';

// Keep refreshing while the worker is still busy with this snapshot: computing it, or
// rendering its map image in the minutes after it completes.
function stillWorking(s: Snapshot) {
  if (s.status === 'pending' || s.status === 'processing') return true;
  return (
    s.status === 'completed' &&
    !s.ndvi_image &&
    !!s.completed_at &&
    Date.now() - Date.parse(s.completed_at) < 10 * 60_000
  );
}

export default async function SnapshotPage({ params }: { params: Promise<{ ref: string }> }) {
  const { ref } = await params;

  // The API answers 404 for both a missing snapshot and someone else's.
  const snapshot = await getSnapshot(decodeURIComponent(ref));
  if (!snapshot.ok) {
    if (snapshot.status === 404) notFound();
    throw new Error(snapshot.error);
  }
  const s = snapshot.data;

  // An old UUID link lands on the short URL.
  if (decodeURIComponent(ref) !== s.short_id) redirect(snapshotPath(s));

  const [parcel, methodologies] = await Promise.all([getParcel(s.parcel_short_id), getMethodologies()]);
  if (!parcel.ok) throw new Error(parcel.error);
  if (!methodologies.ok) throw new Error(methodologies.error);

  const methodology = methodologies.data.find(
    (m) => m.slug === s.methodology && m.version === s.methodology_version,
  );
  const parcelHref = parcelPath({ name: s.parcel_name, short_id: s.parcel_short_id });

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <LiveRefresh active={stillWorking(s)} />

      <Border className="flex flex-wrap items-start justify-between gap-4 p-6">
        <div className="flex min-w-0 flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            {s.chain_seq && (
              <span className="rounded-sm bg-primary px-2 py-0.5 font-mono text-micro font-semibold tracking-[0.08em] text-primary-foreground uppercase">
                Record #{s.chain_seq}
              </span>
            )}
            <StatusBadge status={s.status} />
            {s.completed_at && (
              <span className="font-mono text-micro text-muted-foreground">Sealed {formatUtc(s.completed_at)}</span>
            )}
          </div>
          <h1 className="text-display font-semibold tracking-tight">
            Snapshot {s.date_start} → {s.date_end}
          </h1>
          <div className="flex flex-wrap items-center gap-2 font-mono text-caption">
            <span className="rounded-sm bg-surface-container px-2 py-1">ID: {s.short_id.toUpperCase()}</span>
            <Link href={parcelHref} className="flex items-center gap-1.5 rounded-sm bg-surface-container px-2 py-1 hover:underline">
              <MapIcon className="size-3.5" />
              Parcel: {s.parcel_name}
            </Link>
          </div>
        </div>
        <Button asChild variant="outline" size="lg">
          <Link href={parcelHref}>
            <ArrowLeft />
            Back to parcel
          </Link>
        </Button>
      </Border>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="flex min-w-0 flex-col gap-6">
          <IntegrityPanel snapshot={s} />

          <div className="grid gap-6 lg:grid-cols-2">
            <Border className="flex flex-col gap-4 p-5">
              <h2 className="flex items-center gap-2 text-h2 font-semibold">
                <MapIcon className="size-5" />
                Boundary &amp; NDVI
              </h2>
              <ParcelMap
                geometry={parcel.data.geometry}
                ndviLayers={
                  s.ndvi_image
                    ? [{ shortId: s.short_id, label: `${s.date_start} → ${s.date_end}`, image: s.ndvi_image }]
                    : []
                }
              />
              {s.status === 'completed' && !s.ndvi_image && (
                <p className="font-mono text-micro text-muted-foreground">
                  The NDVI image is rendered shortly after completion and appears here automatically.
                </p>
              )}
            </Border>
            <MethodologyPanel snapshot={s} methodology={methodology} />
          </div>

          <MeasurementsPanel snapshot={s} />
          <ImageryTable sources={s.imagery_sources} status={s.status} />
        </div>

        <div className="xl:sticky xl:top-20 xl:self-start">
          <ProcessingPanel snapshot={s} parcelName={s.parcel_name} methodologies={methodologies.data} />
        </div>
      </div>
    </main>
  );
}
