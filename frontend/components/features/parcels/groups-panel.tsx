import { Combine, ShieldCheck } from 'lucide-react';
import Link from 'next/link';

import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import LocalTime from '@/components/shared/local-time';
import StatusBadge from '@/components/shared/status-badge';
import { methodologyView, resultSummary } from '@/lib/methodologies';
import { snapshotPath } from '@/lib/paths';
import type { GroupSummary } from '@/lib/types';

// Several methodologies run for the same window, shown side by side with the group's seal.
const GroupsPanel = ({ groups }: { groups: GroupSummary[] }) => {
  if (groups.length === 0) return null;

  return (
    <Border className="flex flex-col gap-4 p-5">
      <h2 className="flex items-center gap-2 text-h2 font-semibold">
        <Combine className="size-5" />
        Combined runs
      </h2>
      <ul className="flex flex-col gap-3">
        {groups.map((g) => (
          <li key={g.id} className="flex flex-col gap-3 rounded-md bg-surface-low p-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-mono text-label font-semibold">
                {g.date_start} → {g.date_end}
              </span>
              <span className="rounded-sm bg-surface-container px-2 py-0.5 font-mono text-micro uppercase">
                Group {g.short_id}
              </span>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              {g.members.map((m) => {
                const view = methodologyView(m.methodology);
                const Icon = view?.icon;
                return (
                  <Link
                    key={m.snapshot_id}
                    href={snapshotPath({ short_id: m.short_id })}
                    className="flex flex-col gap-1.5 rounded-sm border border-border bg-surface-lowest p-3 transition-colors hover:border-outline"
                  >
                    <span className="flex items-center justify-between gap-2">
                      <span className="flex items-center gap-1.5 text-label font-semibold">
                        {Icon && <Icon className="size-4" />}
                        {view?.title.replace(' measurements', '') ?? m.methodology}
                      </span>
                      <StatusBadge status={m.status} />
                    </span>
                    <span className="font-mono text-label">{resultSummary(m.methodology, m.measurements) ?? '—'}</span>
                    <span className="font-mono text-micro text-muted-foreground">
                      {m.methodology} v{m.methodology_version}
                    </span>
                  </Link>
                );
              })}
            </div>

            {g.record_hash && g.sealed_at ? (
              <div className="flex items-center gap-2 font-mono text-micro">
                <ShieldCheck className="size-3.5 shrink-0 text-olive" />
                <span className="text-muted-foreground">
                  Sealed <LocalTime iso={g.sealed_at} />
                </span>
                <span className="truncate">{g.record_hash}</span>
                <CopyButton value={g.record_hash} label="Copy group hash" />
              </div>
            ) : (
              <span className="font-mono text-micro text-muted-foreground">
                Sealed with one hash over every member once all have finished.
              </span>
            )}
          </li>
        ))}
      </ul>
    </Border>
  );
};

export default GroupsPanel;
