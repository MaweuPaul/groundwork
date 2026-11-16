import { Link2, ShieldCheck } from 'lucide-react';

import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import LocalTime from '@/components/shared/local-time';
import type { Snapshot } from '@/lib/types';

// What each record format seals, matching snapshot_record_text() in the database.
function coveredBy(format: number) {
  if (format >= 3) {
    return 'the inputs and every parameter, the frozen boundary, every measurement, every source with the version it was read at, any quality flags';
  }
  if (format === 2) return 'the inputs and cloud settings, the frozen boundary, every measurement and scene';
  return 'the inputs, the frozen boundary, every measurement and scene';
}

function HashRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 rounded-sm bg-surface-low p-3 sm:grid-cols-[150px_minmax(0,1fr)_auto] sm:items-center sm:gap-3">
      <span className="font-mono text-micro font-semibold tracking-widest text-muted-foreground uppercase">{label}</span>
      <span className="font-mono text-caption break-all">{value}</span>
      <CopyButton value={value} label={`Copy ${label.toLowerCase()}`} />
    </div>
  );
}

const IntegrityPanel = ({ snapshot }: { snapshot: Snapshot }) => {
  const sealed = snapshot.status === 'completed' && snapshot.record_hash && snapshot.prev_record_hash;

  return (
    <Border className="flex flex-col gap-4 p-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Link2 className="size-5" />
          Cryptographic integrity chain
        </h2>
        {sealed && (
          <span className="flex items-center gap-1.5 rounded-sm bg-primary-fixed px-2 py-1 font-mono text-micro font-semibold tracking-[0.08em] text-primary-container uppercase">
            <ShieldCheck className="size-3.5" />
            Sealed · record #{snapshot.chain_seq}
          </span>
        )}
      </div>

      {sealed ? (
        <>
          <div className="flex flex-col gap-2">
            <HashRow label="Record hash (SHA-256)" value={snapshot.record_hash!} />
            <HashRow label="Previous record hash" value={snapshot.prev_record_hash!} />
            <HashRow label="Boundary hash" value={snapshot.geometry_hash} />
          </div>
          <p className="text-caption text-muted-foreground">
            Sealed {snapshot.completed_at && <LocalTime iso={snapshot.completed_at} />} as record #{snapshot.chain_seq} (format
            v{snapshot.record_format}). The record hash covers {coveredBy(snapshot.record_format)}, and the previous
            record&apos;s hash, so changing anything in this record or any earlier one breaks the chain. The
            database&apos;s <code className="font-mono">verify_snapshot_chain()</code> recomputes every hash to check
            it.
          </p>
        </>
      ) : (
        <>
          <HashRow label="Boundary hash" value={snapshot.geometry_hash} />
          <p className="text-caption text-muted-foreground">
            {snapshot.status === 'failed'
              ? 'This snapshot failed, so it was never sealed into the chain. Failed attempts leave no record hash.'
              : 'The record hash is computed and chained when the snapshot completes.'}
          </p>
        </>
      )}
    </Border>
  );
};

export default IntegrityPanel;
