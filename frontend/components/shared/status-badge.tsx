import type { SnapshotStatus } from '@/lib/types';

const styles: Record<SnapshotStatus | 'none', { label: string; className: string }> = {
  completed: { label: 'Completed', className: 'bg-primary-fixed text-primary-container' },
  processing: { label: 'Processing', className: 'bg-secondary-container text-on-secondary-container' },
  pending: { label: 'Queued', className: 'bg-surface-high text-muted-foreground' },
  failed: { label: 'Failed', className: 'bg-danger-container text-on-danger-container' },
  none: { label: 'No snapshot', className: 'bg-surface-container text-muted-foreground' },
};

const StatusBadge = ({ status }: { status: SnapshotStatus | null }) => {
  const { label, className } = styles[status ?? 'none'];
  return (
    <span
      // Not cn(): it treats the custom text-micro size and the text-* color as a conflict and drops the size.
      className={`inline-flex items-center gap-1.5 rounded-sm px-2 py-0.5 font-mono text-micro font-semibold tracking-[0.08em] whitespace-nowrap uppercase ${className}`}
    >
      <span className="size-1.5 rounded-full bg-current" />
      {label}
    </span>
  );
};

export default StatusBadge;
