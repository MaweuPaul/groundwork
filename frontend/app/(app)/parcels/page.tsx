import { Layers, Plus, Search } from 'lucide-react';
import Link from 'next/link';
import { redirect } from 'next/navigation';

import ParcelsTable from '@/components/features/parcels/parcels-table';
import Border from '@/components/shared/border';
import PageHeader from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { getMe, getParcels } from '@/lib/api';
import type { Parcel } from '@/lib/types';

const filters = {
  all: { label: 'All', match: () => true },
  completed: { label: 'Completed', match: (p: Parcel) => p.latest_snapshot?.status === 'completed' },
  active: {
    label: 'In progress',
    match: (p: Parcel) => p.latest_snapshot?.status === 'pending' || p.latest_snapshot?.status === 'processing',
  },
  failed: { label: 'Failed', match: (p: Parcel) => p.latest_snapshot?.status === 'failed' },
  none: { label: 'No snapshot', match: (p: Parcel) => !p.latest_snapshot },
} as const;

type FilterKey = keyof typeof filters;

export default async function ParcelsPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string; status?: string }>;
}) {
  const me = await getMe();
  if (!me.ok) throw new Error(me.error);
  if (me.data.role !== 'asset_holder') redirect('/');

  const parcels = await getParcels();
  if (!parcels.ok) throw new Error(parcels.error);

  const { q = '', status = 'all' } = await searchParams;
  const query = q.trim().toLowerCase();
  const active: FilterKey = status in filters ? (status as FilterKey) : 'all';

  const matchesQuery = (p: Parcel) =>
    !query ||
    p.name.toLowerCase().includes(query) ||
    p.short_id.includes(query) ||
    (typeof p.metadata.region === 'string' && p.metadata.region.toLowerCase().includes(query));

  const searched = parcels.data.filter(matchesQuery);
  const shown = searched.filter(filters[active].match);

  const href = (key: FilterKey) => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (key !== 'all') params.set('status', key);
    const qs = params.toString();
    return qs ? `/parcels?${qs}` : '/parcels';
  };

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <PageHeader
        icon={Layers}
        title="Parcels"
        description="Every boundary you've registered, with its latest satellite result."
        actions={
          <Button asChild size="lg">
            <Link href="/parcels/new">
              <Plus />
              Register parcel
            </Link>
          </Button>
        }
      />

      <Border className="overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-3 p-4">
          <nav className="flex flex-wrap gap-1" aria-label="Filter by status">
            {(Object.keys(filters) as FilterKey[]).map((key) => (
              <Link
                key={key}
                href={href(key)}
                aria-current={key === active ? 'page' : undefined}
                className={`rounded-sm px-2.5 py-1.5 font-mono text-micro font-semibold tracking-[0.08em] uppercase transition-colors ${
                  key === active
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-surface-container hover:text-foreground'
                }`}
              >
                {filters[key].label}{' '}
                <span className={key === active ? 'opacity-70' : 'opacity-60'}>
                  {searched.filter(filters[key].match).length}
                </span>
              </Link>
            ))}
          </nav>

          <form action="/parcels" className="flex w-full items-center gap-2 sm:w-72">
            {active !== 'all' && <input type="hidden" name="status" value={active} />}
            <label className="flex h-9 flex-1 items-center gap-2 rounded-md border border-border bg-surface-lowest px-3 focus-within:border-ring">
              <Search className="size-4 shrink-0 text-muted-foreground" />
              <input
                type="search"
                name="q"
                defaultValue={q}
                placeholder="Name, region or ID"
                aria-label="Search parcels"
                className="min-w-0 flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted-foreground"
              />
            </label>
          </form>
        </div>

        {shown.length > 0 ? (
          <ParcelsTable parcels={shown} />
        ) : (
          <div className="flex flex-col items-center gap-3 border-t border-border px-6 py-14 text-center">
            {parcels.data.length === 0 ? (
              <>
                <p className="text-h3 font-semibold">No parcels yet</p>
                <p className="max-w-sm text-label text-muted-foreground">
                  Register a boundary to start building its satellite-verified record.
                </p>
                <Button asChild>
                  <Link href="/parcels/new">
                    <Plus />
                    Register parcel
                  </Link>
                </Button>
              </>
            ) : (
              <>
                <p className="text-h3 font-semibold">No parcels match</p>
                <Link href="/parcels" className="text-label text-primary underline underline-offset-2">
                  Clear search and filters
                </Link>
              </>
            )}
          </div>
        )}

        <div className="border-t border-border px-5 py-3 text-label text-muted-foreground">
          Showing <strong className="text-foreground">{shown.length}</strong> of{' '}
          <strong className="text-foreground">{parcels.data.length}</strong> parcels
        </div>
      </Border>
    </main>
  );
}
