import { FlaskConical, Layers, Lock, ShieldCheck } from 'lucide-react';
import Link from 'next/link';

import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import { Button } from '@/components/ui/button';
import { methodologyView } from '@/lib/methodologies';
import type { Methodology } from '@/lib/types';

const Section = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <section className="flex flex-col gap-2 border-t border-border pt-4">
    <h3 className="font-mono text-micro font-semibold tracking-[0.12em] text-muted-foreground uppercase">{title}</h3>
    {children}
  </section>
);

const MethodologySpec = ({ methodology: m, canRun }: { methodology: Methodology; canRun: boolean }) => {
  const view = methodologyView(m.slug);
  const Icon = view?.icon ?? FlaskConical;
  const inputs: [string, string][] = [
    ...(view?.inputs ?? [
      ['Source', m.satellite],
      ['Bands', m.bands.join(', ')],
      ['Composite', m.composite_method],
    ]),
    ['Data from', m.data_start.slice(0, 10)],
  ];

  return (
    <Border id="spec" className="flex scroll-mt-6 flex-col gap-4 p-5 lg:sticky lg:top-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="flex items-center gap-2 text-h2 font-semibold">
            <Icon className="size-5 shrink-0" />
            {m.name}
          </h2>
          <span className="flex flex-wrap items-center gap-2 font-mono text-micro text-muted-foreground">
            {m.slug} v{m.version}
            <span className="flex items-center gap-1 rounded-sm bg-secondary-container px-1.5 py-0.5 font-semibold text-on-secondary-container uppercase">
              <Lock className="size-3" />
              Published
            </span>
          </span>
        </div>
        {canRun && (
          <Button asChild size="lg">
            <Link href="/parcels">
              <Layers />
              Run on a parcel
            </Link>
          </Button>
        )}
      </div>

      {m.description && <p className="max-w-prose text-label text-muted-foreground">{m.description}</p>}

      <Section title="Inputs">
        <dl className="grid gap-x-6 gap-y-2 sm:grid-cols-[max-content_minmax(0,1fr)]">
          {inputs.map(([label, value]) => (
            <div key={label} className="contents">
              <dt className="text-label text-muted-foreground">{label}</dt>
              <dd className="font-mono text-caption wrap-break-word">{value}</dd>
            </div>
          ))}
        </dl>
      </Section>

      {view && (
        <Section title="Formula">
          <div className="flex items-start justify-between gap-3 rounded-sm bg-surface-low px-3 py-2">
            <code className="font-mono text-caption wrap-break-word">{view.formula}</code>
            <CopyButton value={view.formula} label="Copy formula" />
          </div>
        </Section>
      )}

      <Section title="Measurements">
        <ul className="grid gap-2 sm:grid-cols-2">
          {m.measurements.map((mm) => (
            <li key={mm.slug} className="flex flex-col gap-0.5 rounded-sm bg-surface-low px-3 py-2">
              <span className="flex items-baseline justify-between gap-2 text-label font-semibold">
                {mm.name}
                {mm.unit && <span className="font-mono text-micro text-muted-foreground">{mm.unit}</span>}
              </span>
              {mm.description && <span className="text-caption text-muted-foreground">{mm.description}</span>}
            </li>
          ))}
        </ul>
      </Section>

      <Section title="Settings">
        {m.parameters.length === 0 ? (
          <p className="text-label text-muted-foreground">This methodology takes no settings.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[420px] text-left text-label">
              <thead className="font-mono text-micro text-muted-foreground uppercase">
                <tr>
                  <th className="py-1.5 pr-4 font-semibold">Setting</th>
                  <th className="py-1.5 pr-4 font-semibold">Range</th>
                  <th className="py-1.5 font-semibold">Default</th>
                </tr>
              </thead>
              <tbody>
                {m.parameters.map((p) => (
                  <tr key={p.name} className="border-t border-border">
                    <td className="py-2 pr-4">{p.label}</td>
                    <td className="py-2 pr-4 font-mono text-caption">
                      {p.min} – {p.max}
                    </td>
                    <td className="py-2 font-mono text-caption">{p.default}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="text-caption text-muted-foreground">
          Settings can be changed for each snapshot. The values used are sealed into its record.
        </p>
      </Section>

      <Section title="Manifest">
        <p className="text-caption text-muted-foreground">
          The published specification, as text. Every record made with this version seals its SHA-256 hash, so it
          always shows exactly which recipe produced each number.
        </p>
        <div className="flex items-center gap-2 font-mono text-micro">
          <ShieldCheck className="size-3.5 shrink-0 text-olive" />
          <span className="truncate">{m.manifest_sha256}</span>
          <CopyButton value={m.manifest_sha256} label="Copy manifest hash" />
        </div>
        <details className="rounded-sm bg-surface-low px-3 py-2">
          <summary className="cursor-pointer text-label font-semibold">Manifest text</summary>
          <pre className="mt-2 overflow-x-auto font-mono text-micro leading-relaxed">{m.manifest_text}</pre>
        </details>
      </Section>

      {canRun && (
        <p className="border-t border-border pt-4 text-label text-muted-foreground">
          You have <span className="font-semibold text-foreground">{m.records}</span> sealed{' '}
          {m.records === 1 ? 'record' : 'records'} made with this version.
        </p>
      )}
    </Border>
  );
};

export default MethodologySpec;
