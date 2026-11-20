import { FlaskConical, Lock } from 'lucide-react';
import Link from 'next/link';

import { methodologyView } from '@/lib/methodologies';
import { methodologyKey, methodologyPath } from '@/lib/paths';
import type { Methodology } from '@/lib/types';

const MethodologyList = ({ methodologies, selected }: { methodologies: Methodology[]; selected: string }) => (
  <nav aria-label="Methodologies" className="flex flex-col gap-3">
    {methodologies.map((m) => {
      const view = methodologyView(m.slug);
      const Icon = view?.icon ?? FlaskConical;
      const current = methodologyKey(m) === selected;
      return (
        <Link
          key={m.id}
          href={`${methodologyPath(m)}#spec`}
          aria-current={current ? 'page' : undefined}
          className={`flex flex-col gap-2 rounded-md border p-4 transition-colors ${
            current
              ? 'border-primary bg-surface-lowest shadow-sm'
              : 'border-border bg-surface-low hover:border-outline hover:bg-surface-lowest'
          }`}
        >
          <span className="flex items-start justify-between gap-3">
            <span className="flex items-center gap-2 text-label font-semibold">
              <Icon className="size-4 shrink-0" />
              {m.name}
            </span>
            <span className="flex shrink-0 items-center gap-1 rounded-sm bg-secondary-container px-1.5 py-0.5 font-mono text-micro font-semibold text-on-secondary-container uppercase">
              <Lock className="size-3" />
              Published
            </span>
          </span>
          <span className="font-mono text-micro text-muted-foreground">
            {m.slug} v{m.version}
          </span>
          <span className="text-caption text-muted-foreground">
            {view?.scale ? `${view.scale} · ` : ''}data from{' '}
            <span className="whitespace-nowrap">{m.data_start.slice(0, 10)}</span>
          </span>
        </Link>
      );
    })}
  </nav>
);

export default MethodologyList;
