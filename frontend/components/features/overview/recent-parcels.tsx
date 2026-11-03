import { ArrowRight, Grid2x2, Plus } from 'lucide-react';
import Link from 'next/link';

import Border from '@/components/shared/border';
import StatusBadge from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import { formatArea, formatCoordinates, formatDate, formatNdvi } from '@/lib/format';
import type { Parcel } from '@/lib/types';

const columns = ['Parcel', 'Centroid', 'Area (ha)', 'Latest NDVI', 'Last snapshot', 'Status'];

const RecentParcels = ({ parcels, total }: { parcels: Parcel[]; total: number }) => {
  return (
    <Border className="overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-4 p-5">
        <div>
          <h2 className="text-h2 font-semibold">My parcels</h2>
          <p className="text-label text-muted-foreground">
            Registered boundaries with their latest Sentinel-2 NDVI result.
          </p>
        </div>
        <Button asChild size="lg">
          <Link href="/parcels/new">
            <Plus />
            Register parcel
          </Link>
        </Button>
      </div>

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
            {parcels.map((parcel) => (
              <tr key={parcel.id} className="transition-colors hover:bg-surface-low">
                <td className="px-5 py-4">
                  <Link href={`/parcels/${parcel.id}`} className="flex items-center gap-3">
                    <span className="flex size-8 shrink-0 items-center justify-center rounded-sm bg-surface-container text-olive">
                      <Grid2x2 className="size-4" />
                    </span>
                    <span>
                      <span className="block text-body font-semibold hover:underline">
                        {parcel.name}
                      </span>
                      <span className="block font-mono text-micro text-muted-foreground uppercase">
                        {parcel.id.slice(0, 8)}
                      </span>
                    </span>
                  </Link>
                </td>
                <td className="px-5 py-4 font-mono text-caption whitespace-nowrap text-muted-foreground">
                  {formatCoordinates(parcel.centroid_lat, parcel.centroid_lon)}
                </td>
                <td className="px-5 py-4 font-mono text-label">{formatArea(parcel.area_ha)}</td>
                <td className="px-5 py-4 font-mono text-label font-semibold text-olive">
                  {formatNdvi(parcel.latest_result?.measurements.ndvi_mean)}
                </td>
                <td className="px-5 py-4 font-mono text-caption whitespace-nowrap text-muted-foreground">
                  {parcel.latest_result ? formatDate(parcel.latest_result.completed_at) : '—'}
                </td>
                <td className="px-5 py-4">
                  <StatusBadge status={parcel.latest_snapshot?.status ?? null} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between border-t border-border px-5 py-3 text-label text-muted-foreground">
        <span>
          Showing <strong className="text-foreground">{parcels.length}</strong> of{' '}
          <strong className="text-foreground">{total}</strong> parcels
        </span>
        {total > parcels.length && (
          <Link
            href="/parcels"
            className="flex items-center gap-1 font-mono text-caption font-semibold text-primary hover:underline"
          >
            View all parcels <ArrowRight className="size-3.5" />
          </Link>
        )}
      </div>
    </Border>
  );
};

export default RecentParcels;
