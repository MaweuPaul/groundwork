import { BadgeCheck, MapPinned, Plus, Satellite, ShieldCheck } from 'lucide-react';
import Link from 'next/link';

import Border from '@/components/shared/border';
import { Button } from '@/components/ui/button';

const steps = [
  {
    title: 'Define boundary',
    body: 'Draw or upload the parcel polygon. Boundaries are validated before they are stored.',
    detail: 'GeoJSON / MultiPolygon',
    icon: MapPinned,
  },
  {
    title: 'Request a snapshot',
    body: 'Sentinel-2 imagery for the date range is composited with cloud masking and NDVI is computed.',
    detail: '10 m spatial resolution',
    icon: Satellite,
  },
  {
    title: 'Cryptographic proof',
    body: 'Each completed snapshot is sealed into a SHA-256 hash chain that anyone can re-verify.',
    detail: 'SHA-256 record chain',
    icon: ShieldCheck,
  },
];

const GetStarted = () => {
  return (
    <Border className="flex flex-col items-center gap-8 px-6 py-12 text-center">
      <div className="flex flex-col items-center gap-4">
        <span className="inline-flex items-center gap-1.5 rounded-sm bg-secondary-container px-2 py-1 font-mono text-micro font-semibold tracking-[0.12em] text-on-secondary-container uppercase">
          <BadgeCheck className="size-3.5" />
          Initial asset onboarding
        </span>
        <h2 className="max-w-2xl text-h1 font-semibold tracking-tight">
          Register your first parcel to start building a verifiable record
        </h2>
        <p className="max-w-xl text-body text-muted-foreground">
          Groundwork measures vegetation on your land from Sentinel-2 surface reflectance and locks
          every result into a tamper-evident ledger.
        </p>
        <Button asChild size="lg" className="mt-2">
          <Link href="/parcels/new">
            <Plus />
            Register parcel
          </Link>
        </Button>
      </div>

      <div className="grid w-full max-w-4xl gap-4 text-left md:grid-cols-3">
        {steps.map((step, i) => (
          <div key={step.title} className="flex flex-col gap-2 rounded-md bg-surface-container p-5">
            <div className="flex items-center gap-2">
              <span className="flex size-6 items-center justify-center rounded-sm bg-primary font-mono text-caption font-semibold text-primary-foreground">
                {i + 1}
              </span>
              <h3 className="text-h3 font-semibold">{step.title}</h3>
            </div>
            <p className="text-label text-muted-foreground">{step.body}</p>
            <span className="mt-auto flex items-center gap-1.5 pt-2 font-mono text-micro text-muted-foreground">
              <step.icon className="size-3.5" />
              {step.detail}
            </span>
          </div>
        ))}
      </div>
    </Border>
  );
};

export default GetStarted;
