'use client';

import { Satellite, TrendingUp } from 'lucide-react';
import { useState } from 'react';

import ParcelMap, { type NdviLayer } from '@/components/features/parcels/parcel-map';
import ParcelTrend, { type NdviPoint, type RainPoint } from '@/components/features/parcels/parcel-trend';
import Border from '@/components/shared/border';
import type { MultiPolygon } from '@/lib/types';

type Props = {
  geometry: MultiPolygon;
  ndviLayers: NdviLayer[];
  ndvi: NdviPoint[];
  rain: RainPoint[];
};

// The map and the trend side by side; the chart marks the window whose NDVI map is on show,
// so playing the maps moves the marker along the chart.
const ParcelImagery = ({ geometry, ndviLayers, ndvi, rain }: Props) => {
  const [shown, setShown] = useState<string>();

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <Border className="flex flex-col gap-4 p-5">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Satellite className="size-5" />
          Boundary &amp; satellite imagery
        </h2>
        <ParcelMap geometry={geometry} ndviLayers={ndviLayers} onFrameChange={setShown} />
      </Border>

      <Border className="flex flex-col gap-4 p-5">
        <div>
          <h2 className="flex items-center gap-2 text-h2 font-semibold">
            <TrendingUp className="size-5" />
            Vegetation &amp; rainfall
          </h2>
          <p className="text-label text-muted-foreground">
            Mean NDVI and rainfall against the 1991–2020 normal, each plotted at the end of its window on one time axis,
            so a change in vegetation can be read against the rain.
          </p>
        </div>
        <ParcelTrend ndvi={ndvi} rain={rain} marked={shown} />
      </Border>
    </div>
  );
};

export default ParcelImagery;
