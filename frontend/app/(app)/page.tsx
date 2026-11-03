import AuditorEmpty from '@/components/features/overview/auditor-empty';
import GetStarted from '@/components/features/overview/get-started';
import OverviewStats from '@/components/features/overview/overview-stats';
import RecentParcels from '@/components/features/overview/recent-parcels';
import PageHeader from '@/components/shared/page-header';
import { getMe, getParcels } from '@/lib/api';

const RECENT_LIMIT = 5;

export default async function OverviewPage() {
  const me = await getMe();
  if (!me.ok) throw new Error(me.error);

  if (me.data.role === 'auditor') {
    return (
      <main className="flex w-full flex-col gap-6 p-6">
        <PageHeader
          eyebrow="Audit overview"
          title={`Welcome back, ${me.data.first_name}`}
          description="Review parcels and snapshot records that asset holders have shared with you."
        />
        <AuditorEmpty />
      </main>
    );
  }

  const parcels = await getParcels();
  if (!parcels.ok) throw new Error(parcels.error);

  return (
    <main className="flex w-full flex-col gap-6 p-6">
      <PageHeader
        eyebrow="Portfolio overview"
        title={`Welcome back, ${me.data.first_name}`}
        description="Your registered parcels and their latest satellite-verified vegetation results."
      />
      <OverviewStats parcels={parcels.data} />
      {parcels.data.length === 0 ? (
        <GetStarted />
      ) : (
        <RecentParcels parcels={parcels.data.slice(0, RECENT_LIMIT)} total={parcels.data.length} />
      )}
    </main>
  );
}
