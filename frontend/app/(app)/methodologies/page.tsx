import { FlaskConical, ShieldCheck } from 'lucide-react';
import type { Metadata } from 'next';

import MethodologyList from '@/components/features/methodologies/methodology-list';
import MethodologySpec from '@/components/features/methodologies/methodology-spec';
import PageHeader from '@/components/shared/page-header';
import { getMe, getMethodologies } from '@/lib/api';
import { methodologyKey } from '@/lib/paths';

export const metadata: Metadata = { title: 'Methodologies · Groundwork' };

export default async function MethodologiesPage({ searchParams }: { searchParams: Promise<{ m?: string }> }) {
  const [me, methodologies] = await Promise.all([getMe(), getMethodologies()]);
  if (!me.ok) throw new Error(me.error);
  if (!methodologies.ok) throw new Error(methodologies.error);

  const { m } = await searchParams;
  const selected = methodologies.data.find((x) => methodologyKey(x) === m) ?? methodologies.data[0];

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <PageHeader
        icon={FlaskConical}
        title="Methodologies"
        description="Versioned recipes. A methodology never changes once published; a change is a new version, so old records stay reproducible."
      />

      <p className="flex items-start gap-2 rounded-md bg-surface-low px-4 py-3 text-label">
        <ShieldCheck className="mt-0.5 size-4 shrink-0 text-olive" />
        Every record seals the manifest hash of the methodology that produced it.
      </p>

      {selected ? (
        <div className="grid grid-cols-[minmax(0,1fr)] items-start gap-6 lg:grid-cols-[minmax(0,340px)_minmax(0,1fr)]">
          <MethodologyList methodologies={methodologies.data} selected={methodologyKey(selected)} />
          <MethodologySpec methodology={selected} canRun={me.data.role === 'asset_holder'} />
        </div>
      ) : (
        <p className="text-label text-muted-foreground">No methodologies are published yet.</p>
      )}
    </main>
  );
}
