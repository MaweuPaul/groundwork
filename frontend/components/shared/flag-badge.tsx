import { TriangleAlert } from 'lucide-react';

import type { SnapshotFlag } from '@/lib/types';

const labels: Record<string, string> = {
  low_coverage: 'Low coverage',
  map_image_failed: 'No map image',
};

export const flagLabel = (flag: string) => labels[flag] ?? flag.replaceAll('_', ' ');

const FlagBadge = ({ flag }: { flag: SnapshotFlag }) => (
  <span
    title={flag.detail}
    className="inline-flex items-center gap-1 rounded-sm bg-warning-container px-2 py-0.5 font-mono text-micro font-semibold tracking-[0.08em] whitespace-nowrap text-on-warning-container uppercase"
  >
    <TriangleAlert className="size-3" />
    {flagLabel(flag.flag)}
  </span>
);

export default FlagBadge;
