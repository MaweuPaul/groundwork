import { Grid2x2 } from 'lucide-react';
import Link from 'next/link';

import StatusBadge from '@/components/shared/status-badge';
import { formatArea, formatCoordinates, formatDate, formatNdvi } from '@/lib/format';
import { parcelPath } from '@/lib/paths';
import type { Parcel } from '@/lib/types';

const columns = ['Parcel', 'Centroid', 'Area (ha)', 'Latest NDVI', 'Last result', 'Status'];

// compact: for a narrower column next to other content; the centroid shows only on 2xl screens.
const ParcelsTable = ({ parcels, compact = false }: { parcels: Parcel[]; compact?: boolean }) => {
  const centroidClass = compact ? 'hidden 2xl:table-cell' : '';
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left">
        <thead className="border-y border-border bg-surface-low">
          <tr>
            {columns.map((column) => (
              <th
                key={column}
                className={`${column === 'Centroid' ? centroidClass : ''} px-5 py-3 font-mono text-micro font-semibold tracking-[0.12em] whitespace-nowrap text-muted-foreground uppercase`}
              >
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {parcels.map((parcel) => (
            <tr key={parcel.id} className="transition-colors hover:bg-surface-low">
              <td className={`px-5 py-4 ${compact ? 'min-w-56' : ''}`}>
                <Link href={parcelPath(parcel)} className="flex items-center gap-3">
                  <span className="flex size-8 shrink-0 items-center justify-center rounded-sm bg-surface-container text-olive">
                    <Grid2x2 className="size-4" />
                  </span>
                  <span>
                    <span className="block text-body font-semibold hover:underline">{parcel.name}</span>
                    <span className="block font-mono text-micro text-muted-foreground uppercase">
                      {parcel.short_id}
                      {typeof parcel.metadata.region === 'string' && ` · ${parcel.metadata.region}`}
                    </span>
                  </span>
                </Link>
              </td>
              <td
                className={`${centroidClass} px-5 py-4 font-mono text-caption whitespace-nowrap text-muted-foreground`}
              >
                {formatCoordinates(parcel.centroid_lat, parcel.centroid_lon)}
              </td>
              <td className="px-5 py-4 font-mono text-label whitespace-nowrap">{formatArea(parcel.area_ha)}</td>
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
  );
};

export default ParcelsTable;
