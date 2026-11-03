'use client';

import { ChevronDown, CirclePlay, CloudSun, Lightbulb, Lock, PlusCircle, Satellite } from 'lucide-react';
import { useActionState, useState } from 'react';

import { runSnapshotAction, type SnapshotFormState } from '@/app/(app)/parcels/[slug]/actions';
import DatePicker from '@/components/shared/date-picker';
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

const CLOUD_PRESETS = [20, 40, 60, 80];

// Cloud Score+ cs_cdf a pixel needs to count as clear. The API accepts 0.40 to 0.80.
const MASKS = [
  { value: 0.65, label: 'Strict', help: 'Fewer pixels, least haze' },
  { value: 0.6, label: 'Standard', help: 'Google’s recommended balance' },
  { value: 0.5, label: 'Lenient', help: 'More pixels in cloudy seasons' },
];

export type SnapshotOptions = {
  dateStart: string;
  dateEnd: string;
  maxSceneCloudPct: number;
  cloudScoreThreshold: number;
};

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
const choiceClass = (active: boolean) =>
  `rounded-sm border px-3 py-2 text-left transition-colors ${
    active ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-surface-lowest hover:border-outline'
  }`;

const RunSnapshotDialog = ({
  parcelId,
  parcelName,
  methodologies,
  initial,
  note,
  trigger,
}: {
  parcelId: string;
  parcelName: string;
  methodologies: Methodology[];
  // Prefills the form, e.g. when retrying a failed snapshot.
  initial?: SnapshotOptions;
  note?: string;
  trigger?: React.ReactNode;
}) => {
  const first = methodologies[0];
  const [open, setOpen] = useState(false);
  const [range] = useState(defaultRange);
  const [dateStart, setDateStart] = useState(initial?.dateStart ?? range.start);
  const [dateEnd, setDateEnd] = useState(initial?.dateEnd ?? range.end);
  const [methodologyKey, setMethodologyKey] = useState(first ? `${first.slug}@${first.version}` : '');
  const [cloudPct, setCloudPct] = useState(String(initial?.maxSceneCloudPct ?? first?.max_cloud_pct ?? 20));
  const [threshold, setThreshold] = useState(initial?.cloudScoreThreshold ?? 0.6);

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
        {trigger ?? (
          <Button size="lg" className="h-10">
            <PlusCircle />
            Run snapshot
          </Button>
        )}
      </DialogTrigger>

      <DialogContent className="flex max-h-[92svh] flex-col gap-0 overflow-hidden p-0 sm:max-w-xl">
        <DialogHeader className="flex-row items-center gap-3 bg-surface-container px-6 py-5 text-left">
          <Satellite className="size-6 shrink-0 text-primary" />
          <div>
            <DialogTitle className="text-h2 font-semibold">
              {initial ? 'Retry satellite snapshot' : 'Run satellite snapshot'}
            </DialogTitle>
            <DialogDescription className="font-mono text-micro tracking-[0.08em] uppercase">
              Target: {parcelName}
            </DialogDescription>
          </div>
        </DialogHeader>

        <form action={formAction} className="flex min-h-0 flex-col">
          <input type="hidden" name="parcel_id" value={parcelId} />
          <input type="hidden" name="cloud_score_threshold" value={threshold} />

          <div className="flex flex-col gap-5 overflow-y-auto px-6 py-5">
            {state.error && (
              <p
                role="alert"
                className="rounded-sm border border-danger/20 bg-danger-container px-3 py-2 text-label text-on-danger-container"
              >
                {state.error}
              </p>
            )}

            {note && (
              <p className="flex gap-2 rounded-sm bg-secondary-container px-3 py-2 text-caption text-on-secondary-container">
                <Lightbulb className="size-4 shrink-0" />
                {note}
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
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="date_start" className={labelClass}>
                    Acquisition start
                  </label>
                  <DatePicker
                    id="date_start"
                    name="date_start"
                    value={dateStart}
                    onChange={setDateStart}
                    min={EARLIEST}
                    max={dateEnd}
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <label htmlFor="date_end" className={labelClass}>
                    Acquisition end
                  </label>
                  <DatePicker
                    id="date_end"
                    name="date_end"
                    value={dateEnd}
                    onChange={setDateEnd}
                    min={dateStart}
                    max={today}
                  />
                </div>
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
                  {selected.satellite}, bands {selected.bands.join('/')}, {selected.composite_method} composite,
                  clouds masked per pixel with {selected.cloud_mask.replaceAll('_', ' ')}.
                </p>
              )}
            </div>

            <fieldset className="flex flex-col gap-4 rounded-md border border-border p-4">
              <legend className="flex items-center gap-1.5 px-1 text-label font-semibold">
                <CloudSun className="size-4" />
                Cloud settings
              </legend>

              <div className="flex flex-col gap-1.5">
                <label htmlFor="max_scene_cloud_pct" className={labelClass}>
                  Max scene cloud cover
                </label>
                <div className="flex flex-wrap items-center gap-2">
                  {CLOUD_PRESETS.map((pct) => (
                    <button
                      key={pct}
                      type="button"
                      onClick={() => setCloudPct(String(pct))}
                      className={`${choiceClass(Number(cloudPct) === pct)} px-3 py-1.5 font-mono text-label`}
                    >
                      {pct}%
                    </button>
                  ))}
                  <div className="flex items-center gap-1">
                    <input
                      id="max_scene_cloud_pct"
                      name="max_scene_cloud_pct"
                      type="number"
                      min={1}
                      max={100}
                      step={1}
                      required
                      value={cloudPct}
                      onChange={(e) => setCloudPct(e.target.value)}
                      className={`${inputClass} w-20 py-1.5`}
                    />
                    <span className="font-mono text-label text-muted-foreground">%</span>
                  </div>
                </div>
                <p className="text-caption text-muted-foreground">
                  Skips whole scenes cloudier than this across their ~100 km tile. Clouds over the parcel
                  are masked pixel by pixel anyway, so a higher limit often finds clear views in rainy
                  seasons.
                </p>
              </div>

              <div className="flex flex-col gap-1.5">
                <span className={labelClass}>Clear-pixel mask</span>
                <div className="grid grid-cols-3 gap-2" role="radiogroup" aria-label="Clear-pixel mask">
                  {MASKS.map((mask) => (
                    <button
                      key={mask.value}
                      type="button"
                      role="radio"
                      aria-checked={threshold === mask.value}
                      onClick={() => setThreshold(mask.value)}
                      className={choiceClass(threshold === mask.value)}
                    >
                      <span className="block text-label font-semibold">{mask.label}</span>
                      <span className="block font-mono text-micro opacity-80">≥ {mask.value.toFixed(2)}</span>
                      <span className="mt-0.5 block text-micro opacity-80">{mask.help}</span>
                    </button>
                  ))}
                </div>
              </div>

              <p className="font-mono text-micro text-muted-foreground">
                Both settings are saved with the snapshot and sealed into its record hash.
              </p>
            </fieldset>

            <div className="flex gap-3 rounded-md bg-surface-container p-4">
              <Lock className="size-4 shrink-0 text-olive" />
              <p className="text-caption text-muted-foreground">
                <span className="block font-mono font-semibold text-foreground">Immutability notice</span>
                Completed snapshots are sealed into the hash chain. They can&apos;t be edited,
                overwritten, or deleted, including by you.
              </p>
            </div>
          </div>

          <DialogFooter className="mx-0 mb-0 shrink-0 rounded-b-none border-t border-border bg-surface-low px-6 py-4">
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
