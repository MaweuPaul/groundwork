import { ShieldCheck } from 'lucide-react';

import Border from '@/components/shared/border';

const AuditorEmpty = () => {
  return (
    <Border className="flex flex-col items-center gap-3 px-6 py-16 text-center">
      <span className="flex size-12 items-center justify-center rounded-full bg-surface-container text-olive">
        <ShieldCheck className="size-6" />
      </span>
      <h2 className="text-h2 font-semibold">No parcels shared with you yet</h2>
      <p className="max-w-md text-body text-muted-foreground">
        When an asset holder grants you access to a parcel, it will appear here with its snapshots
        and hash-chain records for review.
      </p>
    </Border>
  );
};

export default AuditorEmpty;
