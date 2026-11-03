import { Satellite, TrendingUp } from 'lucide-react';
import { notFound } from 'next/navigation';

import NdviTrend, { type NdviPoint } from '@/components/features/parcels/ndvi-trend';
import ParcelHeader from '@/components/features/parcels/parcel-header';
import ParcelMap from '@/components/features/parcels/parcel-map';
import SnapshotHistory from '@/components/features/parcels/snapshot-history';
import Border from '@/components/shared/border';
import LiveRefresh from '@/components/shared/live-refresh';
import { getMethodologies, getParcel, getParcelSnapshots } from '@/lib/api';

export default async function ParcelPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [parcel, snapshots, methodologies] = await Promise.all([
    getParcel(id),
    getParcelSnapshots(id),
    getMethodologies(),
  ]);

  // The API answers 404 for both a missing parcel and someone else's.
  if (!parcel.ok) {
    if (parcel.status === 404) notFound();
    throw new Error(parcel.error);
  }
  if (!snapshots.ok) throw new Error(snapshots.error);
  if (!methodologies.ok) throw new Error(methodologies.error);

  const inProgress = snapshots.data.some((s) => s.status === 'pending' || s.status === 'processing');

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
    <main className="mx-auto flex w-full max-w-7xl flex-col gap-6 p-6">
      <LiveRefresh active={inProgress} />
      <ParcelHeader parcel={parcel.data} methodologies={methodologies.data} />

      <div className="grid gap-6 lg:grid-cols-2">
        <Border className="flex flex-col gap-4 p-5">
          <h2 className="flex items-center gap-2 text-h2 font-semibold">
            <Satellite className="size-5" />
            Boundary &amp; satellite imagery
          </h2>
          <ParcelMap geometry={parcel.data.geometry} />
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

      <SnapshotHistory snapshots={snapshots.data} />
    </main>
  );
}
