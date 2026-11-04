import { Satellite, TrendingUp } from 'lucide-react';
import { notFound, redirect } from 'next/navigation';

import NdviTrend, { type NdviPoint } from '@/components/features/parcels/ndvi-trend';
import ParcelHeader from '@/components/features/parcels/parcel-header';
import ParcelMap, { type NdviLayer } from '@/components/features/parcels/parcel-map';
import SeriesPanel from '@/components/features/parcels/series-panel';
import SnapshotHistory from '@/components/features/parcels/snapshot-history';
import Border from '@/components/shared/border';
import LiveRefresh from '@/components/shared/live-refresh';
import { getMethodologies, getParcel, getParcelSeries, getParcelSnapshots } from '@/lib/api';
import { parcelPath, refFromSegment } from '@/lib/paths';
import type { SnapshotSummary } from '@/lib/types';

// The worker renders the map image shortly after completion; keep refreshing for a few
// minutes so it appears. A render that failed never gets one, hence the time limit.
function awaitingImage(s: SnapshotSummary) {
  return (
    s.status === 'completed' &&
    !s.ndvi_image &&
    !!s.completed_at &&
    Date.now() - Date.parse(s.completed_at) < 10 * 60_000
  );
}

export default async function ParcelPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const ref = refFromSegment(slug);
  if (!ref) notFound();

  const [parcel, snapshots, methodologies, series] = await Promise.all([
    getParcel(ref),
    getParcelSnapshots(ref),
    getMethodologies(),
    getParcelSeries(ref),
  ]);

  // The API answers 404 for both a missing parcel and someone else's.
  if (!parcel.ok) {
    if (parcel.status === 404) notFound();
    throw new Error(parcel.error);
  }

  // A renamed parcel or an old UUID link lands on the current clean URL.
  const canonical = parcelPath(parcel.data);
  if (`/parcels/${decodeURIComponent(slug)}` !== canonical) redirect(canonical);
  if (!snapshots.ok) throw new Error(snapshots.error);
  if (!methodologies.ok) throw new Error(methodologies.error);
  if (!series.ok) throw new Error(series.error);

  const inProgress = snapshots.data.some((s) => s.status === 'pending' || s.status === 'processing');

  const imagePending = snapshots.data.some(awaitingImage);

  // Newest first, as the API returns them.
  const ndviLayers: NdviLayer[] = snapshots.data.flatMap((s) =>
    s.ndvi_image ? [{ shortId: s.short_id, label: `${s.date_start} → ${s.date_end}`, image: s.ndvi_image }] : [],
  );

  const trend: NdviPoint[] = snapshots.data
    .filter((s) => s.status === 'completed' && s.measurements.ndvi_mean !== undefined)
    .map((s) => ({
      id: s.id,
      dateStart: s.date_start,
      dateEnd: s.date_end,
      ndvi: s.measurements.ndvi_mean,
      validPixelPct: s.measurements.valid_pixel_pct,
    }))
    .sort((a, b) => a.dateEnd.localeCompare(b.dateEnd));

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <LiveRefresh active={inProgress || imagePending} />
      <ParcelHeader parcel={parcel.data} methodologies={methodologies.data} />

      <div className="grid gap-6 lg:grid-cols-2">
        <Border className="flex flex-col gap-4 p-5">
          <h2 className="flex items-center gap-2 text-h2 font-semibold">
            <Satellite className="size-5" />
            Boundary &amp; satellite imagery
          </h2>
          <ParcelMap geometry={parcel.data.geometry} ndviLayers={ndviLayers} />
        </Border>

        <Border className="flex flex-col gap-4 p-5">
          <div>
            <h2 className="flex items-center gap-2 text-h2 font-semibold">
              <TrendingUp className="size-5" />
              NDVI time-series trend
            </h2>
            <p className="text-label text-muted-foreground">
              Mean NDVI per completed snapshot, plotted at the end of each observation window.
            </p>
          </div>
          <NdviTrend points={trend} />
        </Border>
      </div>

      <SeriesPanel series={series.data} />

      <SnapshotHistory
        snapshots={snapshots.data}
        parcelId={parcel.data.id}
        parcelName={parcel.data.name}
        methodologies={methodologies.data}
      />
    </main>
  );
}
