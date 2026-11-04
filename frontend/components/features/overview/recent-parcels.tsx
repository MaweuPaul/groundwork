import { ArrowRight, Plus } from 'lucide-react';
import Link from 'next/link';

import ParcelsTable from '@/components/features/parcels/parcels-table';
import Border from '@/components/shared/border';
import { Button } from '@/components/ui/button';
import type { Parcel } from '@/lib/types';

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

      <ParcelsTable parcels={parcels} />

      <div className="flex items-center justify-between border-t border-border px-5 py-3 text-label text-muted-foreground">
        <span>
          Showing <strong className="text-foreground">{parcels.length}</strong> of{' '}
          <strong className="text-foreground">{total}</strong> parcels
        </span>
        <Link
          href="/parcels"
          className="flex items-center gap-1 font-mono text-caption font-semibold text-primary hover:underline"
        >
          View all parcels <ArrowRight className="size-3.5" />
        </Link>
      </div>
    </Border>
  );
};

export default RecentParcels;
