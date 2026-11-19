import { Satellite, TrendingUp } from 'lucide-react';
import { notFound, redirect } from 'next/navigation';

import ChainCheck from '@/components/features/parcels/chain-check';
import ParcelTrend, { type NdviPoint, type RainPoint } from '@/components/features/parcels/parcel-trend';
import GroupsPanel from '@/components/features/parcels/groups-panel';
import ParcelHeader from '@/components/features/parcels/parcel-header';
import ParcelMap, { type NdviLayer } from '@/components/features/parcels/parcel-map';
import SeriesPanel from '@/components/features/parcels/series-panel';
import SnapshotHistory from '@/components/features/parcels/snapshot-history';
import Border from '@/components/shared/border';
import LiveRefresh from '@/components/shared/live-refresh';
import { getMethodologies, getParcel, getParcelGroups, getParcelSeries, getParcelSnapshots } from '@/lib/api';
import { parcelPath, refFromSegment } from '@/lib/paths';

export default async function ParcelPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const ref = refFromSegment(slug);
  if (!ref) notFound();

  const [parcel, snapshots, methodologies, series, groups] = await Promise.all([
    getParcel(ref),
    getParcelSnapshots(ref),
    getMethodologies(),
    getParcelSeries(ref),
    getParcelGroups(ref),
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
  if (!groups.ok) throw new Error(groups.error);

  const inProgress = snapshots.data.some((s) => s.status === 'pending' || s.status === 'processing');

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

  // A window with no normal (0 mm in every baseline year) has no % of normal to plot.
  const rain: RainPoint[] = snapshots.data
    .filter((s) => s.status === 'completed' && s.measurements.rain_pct_of_normal !== undefined)
    .map((s) => ({
      id: s.id,
      dateStart: s.date_start,
      dateEnd: s.date_end,
      pctOfNormal: s.measurements.rain_pct_of_normal,
      totalMm: s.measurements.rain_total_mm,
      normalMm: s.measurements.rain_normal_mm,
    }))
    .sort((a, b) => a.dateEnd.localeCompare(b.dateEnd));

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <LiveRefresh active={inProgress} />
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
              Vegetation &amp; rainfall
            </h2>
            <p className="text-label text-muted-foreground">
              Mean NDVI and rainfall against the 1991–2020 normal, each plotted at the end of its window on one time
              axis, so a change in vegetation can be read against the rain.
            </p>
          </div>
          <ParcelTrend ndvi={trend} rain={rain} />
        </Border>
      </div>

      <GroupsPanel groups={groups.data} />

      <SeriesPanel series={series.data} />

      <SnapshotHistory
        snapshots={snapshots.data}
        parcelId={parcel.data.id}
        parcelName={parcel.data.name}
        methodologies={methodologies.data}
      />

      <ChainCheck parcelId={parcel.data.id} />
    </main>
  );
}
