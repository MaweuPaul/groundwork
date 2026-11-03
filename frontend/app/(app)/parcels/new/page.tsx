import { MapPinPlus, ShieldCheck } from 'lucide-react';
import { redirect } from 'next/navigation';

import RegisterParcelForm from '@/components/features/parcels/register-parcel-form';
import PageHeader from '@/components/shared/page-header';
import { getMe } from '@/lib/api';

export default async function NewParcelPage() {
  const me = await getMe();
  if (!me.ok) throw new Error(me.error);
  if (me.data.role !== 'asset_holder') redirect('/');

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <PageHeader
        icon={MapPinPlus}
        badge="Cadastre draft"
        title="Register new parcel"
        description="Draw the boundary on the map or upload a GeoJSON file to add it to the verification ledger."
        actions={
          <span className="flex items-center gap-1.5 rounded-sm bg-surface-container px-3 py-1.5 font-mono text-micro text-muted-foreground">
            <ShieldCheck className="size-3.5" />
            Boundaries are validated by PostGIS on save
          </span>
        }
      />
      <RegisterParcelForm />
    </main>
  );
}
