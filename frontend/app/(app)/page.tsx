import AuditorEmpty from '@/components/features/overview/auditor-empty';
import GetStarted from '@/components/features/overview/get-started';
import OverviewStats from '@/components/features/overview/overview-stats';
import RecentActivity from '@/components/features/overview/recent-activity';
import RecentParcels from '@/components/features/overview/recent-parcels';
import PageHeader from '@/components/shared/page-header';
import { getActivity, getMe, getParcels } from '@/lib/api';

const RECENT_LIMIT = 5;
const ACTIVITY_LIMIT = 8;

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

  const [parcels, activity] = await Promise.all([getParcels(), getActivity(ACTIVITY_LIMIT)]);
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
        <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
          <RecentParcels parcels={parcels.data.slice(0, RECENT_LIMIT)} total={parcels.data.length} />
          <RecentActivity
            events={activity.ok ? activity.data : []}
            error={activity.ok ? undefined : activity.error}
          />
        </div>
      )}
    </main>
  );
}
