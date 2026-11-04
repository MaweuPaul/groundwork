import { Clock, Layers, Leaf, Ruler } from 'lucide-react';

import StatCard from '@/components/shared/stat-card';
import { formatArea, formatNdvi, timeAgo } from '@/lib/format';
import type { Parcel } from '@/lib/types';

const OverviewStats = ({ parcels }: { parcels: Parcel[] }) => {
  const totalArea = parcels.reduce((sum, p) => sum + p.area_ha, 0);

  const ndviValues = parcels
    .map((p) => p.latest_result?.measurements.ndvi_mean)
    .filter((v): v is number => v !== undefined);
  const averageNdvi = ndviValues.length
    ? ndviValues.reduce((sum, v) => sum + v, 0) / ndviValues.length
    : undefined;

  const lastCompleted = parcels
    .map((p) => p.latest_result?.completed_at)
    .filter((d): d is string => d !== undefined)
    .sort()
    .at(-1);

  const inProgress = parcels.filter(
    (p) => p.latest_snapshot?.status === 'pending' || p.latest_snapshot?.status === 'processing',
  ).length;

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        label="Registered parcels"
        icon={Layers}
        value={String(parcels.length)}
        unit={parcels.length === 1 ? 'parcel' : 'parcels'}
        caption={`${ndviValues.length} with a completed snapshot`}
      />
      <StatCard
        label="Monitored area"
        icon={Ruler}
        value={formatArea(totalArea)}
        unit="ha"
        caption={`${(totalArea / 100).toFixed(2)} km² · WGS84`}
      />
      <StatCard
        label="Average NDVI"
        icon={Leaf}
        value={formatNdvi(averageNdvi)}
        caption={
          ndviValues.length
            ? `Latest result across ${ndviValues.length} ${ndviValues.length === 1 ? 'parcel' : 'parcels'}`
            : 'No results yet'
        }
      />
      <StatCard
        label="Last snapshot"
        icon={Clock}
        value={lastCompleted ? timeAgo(lastCompleted) : 'None yet'}
        caption={inProgress ? `${inProgress} in progress` : 'Nothing queued'}
      />
    </div>
  );
};

export default OverviewStats;
