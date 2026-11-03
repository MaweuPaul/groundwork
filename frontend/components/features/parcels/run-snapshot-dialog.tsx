'use client';

import { ChevronDown, CirclePlay, Lock, PlusCircle, Satellite } from 'lucide-react';
import { useActionState, useState } from 'react';

import { runSnapshotAction, type SnapshotFormState } from '@/app/(app)/parcels/[id]/actions';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import type { Methodology } from '@/lib/types';

// Sentinel-2B's first imagery; the API rejects earlier start dates.
const EARLIEST = '2015-06-23';

const isoDay = (d: Date) => d.toISOString().slice(0, 10);

function defaultRange() {
  const end = new Date();
  end.setUTCDate(end.getUTCDate() - 1);
  const start = new Date(end);
  start.setUTCDate(start.getUTCDate() - 29);
  return { start: isoDay(start), end: isoDay(end) };
}

const labelClass = 'font-mono text-micro font-semibold tracking-widest text-muted-foreground uppercase';
const inputClass =
  'w-full rounded-sm border border-transparent bg-surface-container px-3 py-2.5 font-mono text-label outline-none transition-colors focus:border-primary';

const RunSnapshotDialog = ({
  parcelId,
  parcelName,
  methodologies,
}: {
  parcelId: string;
  parcelName: string;
  methodologies: Methodology[];
}) => {
  const [open, setOpen] = useState(false);
  const [range] = useState(defaultRange);
  const [methodologyKey, setMethodologyKey] = useState(
    methodologies[0] ? `${methodologies[0].slug}@${methodologies[0].version}` : '',
  );

  const [state, formAction, pending] = useActionState(
    async (prev: SnapshotFormState, formData: FormData) => {
      const result = await runSnapshotAction(prev, formData);
      if (result.ok) setOpen(false);
      return result;
    },
    {},
  );

  const selected = methodologies.find((m) => `${m.slug}@${m.version}` === methodologyKey);
  const today = isoDay(new Date());

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="lg" className="h-10">
          <PlusCircle />
          Run snapshot
        </Button>
      </DialogTrigger>

      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-xl">
        <DialogHeader className="flex-row items-center gap-3 bg-surface-container px-6 py-5 text-left">
          <Satellite className="size-6 shrink-0 text-primary" />
          <div>
            <DialogTitle className="text-h2 font-semibold">Run satellite snapshot</DialogTitle>
            <DialogDescription className="font-mono text-micro tracking-[0.08em] uppercase">
              Target: {parcelName}
            </DialogDescription>
          </div>
        </DialogHeader>

        <form action={formAction} className="flex flex-col">
          <input type="hidden" name="parcel_id" value={parcelId} />

          <div className="flex flex-col gap-5 px-6 py-5">
            {state.error && (
              <p
                role="alert"
                className="rounded-sm border border-danger/20 bg-danger-container px-3 py-2 text-label text-on-danger-container"
              >
                {state.error}
              </p>
            )}

            <fieldset className="flex flex-col gap-2">
              <legend className="mb-2 flex w-full justify-between text-label font-semibold">
                Observation window
                <span className="font-mono text-micro font-normal text-muted-foreground">
                  UTC · max one year
                </span>
              </legend>
              <div className="grid grid-cols-2 gap-3">
                <label className="flex flex-col gap-1.5">
                  <span className={labelClass}>Acquisition start</span>
                  <input
                    type="date"
                    name="date_start"
                    required
                    min={EARLIEST}
                    max={today}
                    defaultValue={state.values?.date_start ?? range.start}
                    className={inputClass}
                  />
                </label>
                <label className="flex flex-col gap-1.5">
                  <span className={labelClass}>Acquisition end</span>
                  <input
                    type="date"
                    name="date_end"
                    required
                    min={EARLIEST}
                    max={today}
                    defaultValue={state.values?.date_end ?? range.end}
                    className={inputClass}
                  />
                </label>
              </div>
            </fieldset>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="methodology" className="text-label font-semibold">
                Processing methodology
              </label>
              <div className="relative">
                <select
                  id="methodology"
                  name="methodology"
                  value={methodologyKey}
                  onChange={(e) => setMethodologyKey(e.target.value)}
                  className={`${inputClass} appearance-none pr-9`}
                >
                  {methodologies.map((m) => (
                    <option key={m.id} value={`${m.slug}@${m.version}`}>
                      {m.name} (v{m.version})
                    </option>
                  ))}
                </select>
                <ChevronDown className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground" />
              </div>
              {selected && (
                <p className="text-caption text-muted-foreground">
                  {selected.satellite}, bands {selected.bands.join('/')}, {selected.composite_method}{' '}
                  composite, cloud mask {selected.cloud_mask.replaceAll('_', ' ')} (max{' '}
                  {selected.max_cloud_pct}% cloud).
                </p>
              )}
            </div>

            <div className="flex gap-3 rounded-md bg-surface-container p-4">
              <Lock className="size-4 shrink-0 text-olive" />
              <p className="text-caption text-muted-foreground">
                <span className="block font-mono font-semibold text-foreground">Immutability notice</span>
                Completed snapshots are sealed into the hash chain. They can&apos;t be edited,
                overwritten, or deleted, including by you.
              </p>
            </div>
          </div>

          <DialogFooter className="mx-0 mb-0 rounded-b-none border-t border-border bg-surface-low px-6 py-4">
            <DialogClose asChild>
              <Button type="button" variant="outline" size="lg">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" size="lg" disabled={pending || !selected}>
              <CirclePlay />
              {pending ? 'Queuing…' : 'Start snapshot'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export default RunSnapshotDialog;
