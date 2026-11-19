import Border from '@/components/shared/border';
import LocalTime from '@/components/shared/local-time';
import { formatDate } from '@/lib/format';
import { sourcesView } from '@/lib/methodologies';
import type { ImagerySource, SnapshotStatus } from '@/lib/types';

const cell = 'px-5 py-3 font-mono text-caption';

const ImageryTable = ({
  sources,
  status,
  methodology,
}: {
  sources: ImagerySource[];
  status: SnapshotStatus;
  methodology: string;
}) => {
  const view = sourcesView(methodology);
  const Icon = view.icon;
  const columns = view.daily
    ? ['Day (UTC)', 'Version']
    : ['Scene identifier', 'Satellite', 'Acquired', 'Scene cloud', 'Version'];

  return (
    <Border className="overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-2 p-5">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Icon className="size-5" />
          {view.title}
        </h2>
        <span className="rounded-sm bg-surface-container px-2 py-1 font-mono text-micro tracking-[0.08em] uppercase">
          {view.count(sources.length)}
        </span>
      </div>

      {sources.length === 0 ? (
        <p className="border-t border-border px-5 py-8 text-center text-label text-muted-foreground">
          {status === 'failed'
            ? 'No sources were recorded: the snapshot failed.'
            : 'Sources are listed once the snapshot completes.'}
        </p>
      ) : (
        <div className="max-h-112 overflow-auto">
          <table className="w-full text-left">
            <thead className="sticky top-0 border-y border-border bg-surface-low">
              <tr>
                {columns.map((column) => (
                  <th
                    key={column}
                    className="px-5 py-3 font-mono text-micro font-semibold tracking-[0.12em] whitespace-nowrap text-muted-foreground uppercase"
                  >
                    {column}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {sources.map((s) => (
                <tr key={s.scene_id}>
                  {view.daily ? (
                    // A daily reading covers a UTC calendar day, so it is shown as that day, not a local time.
                    <td className={`${cell} whitespace-nowrap`}>{formatDate(s.acquisition_time)}</td>
                  ) : (
                    <>
                      <td className={`${cell} break-all`}>{s.scene_id}</td>
                      <td className={`${cell} whitespace-nowrap capitalize`}>{s.satellite}</td>
                      <td className={`${cell} whitespace-nowrap text-muted-foreground`}>
                        <LocalTime iso={s.acquisition_time} />
                      </td>
                      <td className={cell}>{s.cloud_cover_pct == null ? '—' : `${s.cloud_cover_pct.toFixed(1)}%`}</td>
                    </>
                  )}
                  <td className={`${cell} break-all text-muted-foreground`}>{s.source_version ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {view.note && (
        <p className="border-t border-border px-5 py-3 font-mono text-micro text-muted-foreground">{view.note}</p>
      )}
    </Border>
  );
};

export default ImageryTable;
