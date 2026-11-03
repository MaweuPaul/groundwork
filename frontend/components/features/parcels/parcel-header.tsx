import { MapPin } from 'lucide-react';

import RunSnapshotDialog from '@/components/features/parcels/run-snapshot-dialog';
import Border from '@/components/shared/border';
import StatusBadge from '@/components/shared/status-badge';
import { formatArea, formatCoordinates } from '@/lib/format';
import type { Methodology, Parcel } from '@/lib/types';

const chip = 'rounded-sm bg-surface-container px-2 py-1 font-mono text-caption';

const ParcelHeader = ({ parcel, methodologies }: { parcel: Parcel; methodologies: Methodology[] }) => {
  const region = typeof parcel.metadata.region === 'string' ? parcel.metadata.region : null;

  return (
    <Border className="flex flex-wrap items-start justify-between gap-4 p-6">
      <div className="flex min-w-0 flex-col gap-3">
        <h1 className="flex flex-wrap items-center gap-3 text-display font-semibold tracking-tight">
          {parcel.name}
          <StatusBadge status={parcel.latest_snapshot?.status ?? null} />
        </h1>
        <div className="flex flex-wrap items-center gap-2">
          <span className={chip}>ID: {parcel.id.slice(0, 8).toUpperCase()}</span>
          <span className={chip}>{formatArea(parcel.area_ha)} ha</span>
          <span className={`flex items-center gap-1.5 ${chip}`}>
            <MapPin className="size-3.5" />
            {region ? `${region} ` : ''}({formatCoordinates(parcel.centroid_lat, parcel.centroid_lon)})
          </span>
        </div>
        {parcel.description && (
          <p className="max-w-2xl text-body text-muted-foreground">{parcel.description}</p>
        )}
      </div>
      <RunSnapshotDialog parcelId={parcel.id} parcelName={parcel.name} methodologies={methodologies} />
    </Border>
  );
};

export default ParcelHeader;
