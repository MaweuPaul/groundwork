import { Layers3, ShieldCheck } from 'lucide-react';

import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import { formatUtc } from '@/lib/format';
import type { SeriesSummary } from '@/lib/types';

const months = (n: number) => `${n} ${n === 1 ? 'month' : 'months'}`;

const SeriesPanel = ({ series }: { series: SeriesSummary[] }) => {
  if (series.length === 0) return null;

  return (
    <Border className="flex flex-col gap-4 p-5">
      <h2 className="flex items-center gap-2 text-h2 font-semibold">
        <Layers3 className="size-5" />
        Time series
      </h2>
      <ul className="flex flex-col gap-3">
        {series.map((s) => {
          const finished = s.completed + s.failed;
          return (
            <li key={s.id} className="flex flex-col gap-2 rounded-md bg-surface-low p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="font-mono text-label font-semibold">
                  {s.date_start} → {s.date_end}
                </span>
                <span className="rounded-sm bg-surface-container px-2 py-0.5 font-mono text-micro uppercase">
                  Series {s.short_id}
                </span>
              </div>
              <span className="font-mono text-micro text-muted-foreground">
                {months(s.window_months)} windows every {months(s.interval_months)} · scenes ≤{s.max_scene_cloud_pct}% ·
                clear ≥{s.cloud_score_threshold.toFixed(2)}
              </span>
              <div className="flex items-center gap-3">
                <div className="h-2 flex-1 overflow-hidden rounded-full bg-surface-high">
                  <div
                    className="h-full rounded-full bg-primary-container"
                    style={{ width: `${(finished / Math.max(1, s.windows)) * 100}%` }}
                  />
                </div>
                <span className="font-mono text-micro whitespace-nowrap text-muted-foreground">
                  {s.completed} done{s.failed > 0 && ` · ${s.failed} failed`} / {s.windows}
                </span>
              </div>
              {s.record_hash && s.sealed_at ? (
                <div className="flex items-center gap-2 font-mono text-micro">
                  <ShieldCheck className="size-3.5 shrink-0 text-olive" />
                  <span className="text-muted-foreground">Sealed {formatUtc(s.sealed_at)}</span>
                  <span className="truncate">{s.record_hash}</span>
                  <CopyButton value={s.record_hash} label="Copy series hash" />
                </div>
              ) : (
                <span className="font-mono text-micro text-muted-foreground">
                  Sealed with one hash over every window once all have finished.
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </Border>
  );
};

export default SeriesPanel;
