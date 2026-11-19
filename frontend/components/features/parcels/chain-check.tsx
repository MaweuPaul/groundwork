'use client';

import { RefreshCw, ShieldAlert, ShieldCheck } from 'lucide-react';
import Link from 'next/link';
import { useState, useTransition } from 'react';

import { verifyChainAction } from '@/app/(app)/parcels/[slug]/actions';
import Border from '@/components/shared/border';
import CopyButton from '@/components/shared/copy-button';
import LocalTime from '@/components/shared/local-time';
import { Button } from '@/components/ui/button';
import { snapshotPath } from '@/lib/paths';
import type { ChainCheck as Check, ChainProblem } from '@/lib/types';

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`;

const ProblemSubject = ({ problem }: { problem: ChainProblem }) => {
  if (problem.short_id) {
    return (
      <Link href={snapshotPath({ short_id: problem.short_id })} className="font-semibold underline-offset-2 hover:underline">
        Record #{problem.seq} ({problem.short_id})
      </Link>
    );
  }
  if (problem.group_short_id) return <span className="font-semibold">Combined run {problem.group_short_id}</span>;
  return <span className="font-semibold">Chain head</span>;
};

const Result = ({ check }: { check: Check }) => {
  if (!check.intact) {
    return (
      <div className="flex flex-col gap-2">
        <p className="flex items-center gap-2 text-label font-semibold text-danger">
          <ShieldAlert className="size-4 shrink-0" />
          {plural(check.problems.length, 'problem')} found
        </p>
        <ul className="flex flex-col gap-1.5">
          {check.problems.map((p, i) => (
            <li key={i} className="rounded-sm bg-danger-container px-3 py-2 text-label">
              <ProblemSubject problem={p} />: {p.problem}
            </li>
          ))}
        </ul>
      </div>
    );
  }

  if (check.records === 0) {
    return <p className="text-label text-muted-foreground">No sealed records yet. Nothing to check.</p>;
  }

  return (
    <div className="flex flex-col gap-1.5">
      <p className="flex items-center gap-2 text-label font-semibold">
        <ShieldCheck className="size-4 shrink-0 text-olive" />
        All {plural(check.records, 'record')} verify
        {check.groups > 0 && `, and ${plural(check.groups, 'combined run')}`}
      </p>
      {check.head_hash && (
        <div className="flex items-center gap-2 font-mono text-micro">
          <span className="shrink-0 text-muted-foreground">Head #{check.head_seq}</span>
          <span className="truncate">{check.head_hash}</span>
          <CopyButton value={check.head_hash} label="Copy chain head hash" />
        </div>
      )}
    </div>
  );
};

// Asks the API to recompute every sealed record and group of the parcel from what is stored now.
const ChainCheck = ({ parcelId }: { parcelId: string }) => {
  const [check, setCheck] = useState<Check | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, startTransition] = useTransition();

  const run = () =>
    startTransition(async () => {
      const result = await verifyChainAction(parcelId);
      setCheck(result.ok ? result.data : null);
      setError(result.ok ? null : result.error);
    });

  return (
    <Border className="flex flex-col gap-4 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h2 className="flex items-center gap-2 text-h2 font-semibold">
            <ShieldCheck className="size-5" />
            Chain check
          </h2>
          <p className="max-w-prose text-label text-muted-foreground">
            Recomputes every sealed record and combined run of this parcel from what is stored now. A record changed
            or removed after it was sealed shows up here.
          </p>
        </div>
        <Button variant="outline" size="lg" onClick={run} disabled={pending}>
          <RefreshCw className={pending ? 'animate-spin' : undefined} />
          {pending ? 'Checking…' : 'Recheck chain'}
        </Button>
      </div>

      <div aria-live="polite" className="flex flex-col gap-2">
        {error && <p className="text-label text-danger">{error}</p>}
        {check && (
          <>
            <Result check={check} />
            <span className="font-mono text-micro text-muted-foreground">
              Checked <LocalTime iso={check.checked_at} />
            </span>
          </>
        )}
      </div>
    </Border>
  );
};

export default ChainCheck;
