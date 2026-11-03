import { Satellite } from 'lucide-react';

import Border from '@/components/shared/border';
import { formatUtc } from '@/lib/format';
import type { ImagerySource, SnapshotStatus } from '@/lib/types';

const columns = ['Scene identifier', 'Satellite', 'Acquired', 'Scene cloud'];

const ImageryTable = ({ sources, status }: { sources: ImagerySource[]; status: SnapshotStatus }) => {
  return (
    <Border className="overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-2 p-5">
        <h2 className="flex items-center gap-2 text-h2 font-semibold">
          <Satellite className="size-5" />
          Imagery sources
        </h2>
        <span className="rounded-sm bg-surface-container px-2 py-1 font-mono text-micro tracking-[0.08em] uppercase">
          {sources.length} {sources.length === 1 ? 'scene' : 'scenes'} in the composite
        </span>
      </div>

      {sources.length === 0 ? (
        <p className="border-t border-border px-5 py-8 text-center text-label text-muted-foreground">
          {status === 'failed'
            ? 'No scenes were recorded: the snapshot failed.'
            : 'Scenes are listed once the snapshot completes.'}
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
                  <td className="px-5 py-3 font-mono text-caption break-all">{s.scene_id}</td>
                  <td className="px-5 py-3 font-mono text-caption whitespace-nowrap capitalize">{s.satellite}</td>
                  <td className="px-5 py-3 font-mono text-caption whitespace-nowrap text-muted-foreground">
                    {formatUtc(s.acquisition_time)}
                  </td>
                  <td className="px-5 py-3 font-mono text-caption">
                    {s.cloud_cover_pct === undefined || s.cloud_cover_pct === null ? '—' : `${s.cloud_cover_pct.toFixed(1)}%`}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <p className="border-t border-border px-5 py-3 font-mono text-micro text-muted-foreground">
        Copernicus Sentinel-2 Level-2A surface reflectance via Google Earth Engine. Scene cloud is the whole tile&apos;s
        estimate; clouds over the parcel were masked per pixel with Cloud Score+.
      </p>
    </Border>
  );
};

export default ImageryTable;
