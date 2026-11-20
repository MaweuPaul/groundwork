'use client';

import { CirclePlay, Lightbulb, Lock, PlusCircle, Satellite, SlidersHorizontal } from 'lucide-react';
import { useActionState, useState } from 'react';

import { runSnapshotAction, type SnapshotFormState } from '@/app/(app)/parcels/[slug]/actions';
import DatePicker from '@/components/shared/date-picker';
import InfoTip from '@/components/shared/info-tip';
import { ACTIVITY_CHANGED } from '@/components/shared/snapshot-notifier';
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
import { methodologyView } from '@/lib/methodologies';
import { PARAM_PREFIX } from '@/lib/snapshot-form';
import type { Methodology, MethodologyParameter } from '@/lib/types';

// Cloud Score+ cs_cdf a pixel needs to count as clear.
const MASKS = [
  { value: 0.65, label: 'Strict', help: 'Fewer pixels, least haze' },
  { value: 0.6, label: 'Standard', help: 'Google’s recommended balance' },
  { value: 0.5, label: 'Lenient', help: 'More pixels in cloudy seasons' },
];

export type SnapshotOptions = {
  methodology?: string; // "slug@version"; the first methodology when unset
  dateStart: string;
  dateEnd: string;
  parameters?: Record<string, number>;
};

const isoDay = (d: Date) => d.toISOString().slice(0, 10);

const MONTH_CHOICES = [1, 2, 3, 6, 12];

const HELP = {
  window: 'The dates to measure, start and end included. Everything the methodology uses in this range goes into one result.',
  series:
    'The whole period to cover. It is split into windows below; each becomes its own snapshot.',
  windowLength:
    'How much data goes into each point. Longer windows smooth out short gaps but blur changes within them.',
  interval:
    'How far apart the windows start. Equal to the window length gives back-to-back readings with no gaps; larger skips the months in between, e.g. 2 months every 12 compares the same season each year.',
  methodology:
    'The exact, versioned recipe used to compute the result. It is sealed into the record, so anyone can rerun it and check the numbers.',
  cloud:
    'Sentinel-2 rates each whole scene (~100 × 100 km) by cloud cover. Scenes above this limit are skipped. Clouds over your parcel are still removed pixel by pixel, so a higher limit often finds clear views in rainy seasons.',
  mask:
    'How sure Cloud Score+ must be that a pixel is clear before it is used. Strict avoids haze but keeps fewer pixels; lenient keeps more in cloudy seasons.',
  rainyDay:
    'A day with at least this much rain counts as rainy; less counts as dry. It sets the rainy-day count and the longest dry spell. 1 mm is the common standard.',
  minClear:
    'If less of the parcel than this is clear after masking, the snapshot still completes but is flagged low coverage, so readers know its values come from only part of the parcel.',
};

// Friendlier controls for parameters we know. Any other parameter gets a plain number input.
type Extra = {
  label: string;
  unit?: string;
  presets?: number[];
  choices?: typeof MASKS;
  help: string;
  note?: string;
};

const EXTRAS: Record<string, Extra> = {
  max_scene_cloud_pct: {
    label: 'Max scene cloud cover',
    unit: '%',
    presets: [20, 40, 60, 80],
    help: HELP.cloud,
    note: 'Skips whole scenes cloudier than this across their ~100 km tile. Clouds over the parcel are masked pixel by pixel anyway, so a higher limit often finds clear views in rainy seasons.',
  },
  cloud_score_threshold: { label: 'Clear-pixel mask', choices: MASKS, help: HELP.mask },
  min_clear_pct: {
    label: 'Minimum clear pixels',
    unit: '%',
    presets: [25, 50, 75],
    help: HELP.minClear,
    note: 'Below this, the snapshot is still sealed but flagged low coverage. It never fails for it.',
  },
  rainy_day_mm: {
    label: 'Rainy day threshold',
    unit: 'mm',
    presets: [1, 2, 5],
    help: HELP.rainyDay,
  },
};

const EXTRA_ORDER = Object.keys(EXTRAS);

const keyOf = (m: Methodology) => `${m.slug}@${m.version}`;
const rank = (name: string) => (EXTRA_ORDER.includes(name) ? EXTRA_ORDER.indexOf(name) : EXTRA_ORDER.length);

const defaultsFor = (m: Methodology | undefined, given?: Record<string, number>) =>
  Object.fromEntries((m?.parameters ?? []).map((p) => [p.name, String(given?.[p.name] ?? p.default)]));

const MAX_WINDOWS = 24;

// Same rule as the API: a window every `interval` months, `length` months long, the last
// one cut short at the end date.
function seriesWindows(start: string, end: string, length: number, interval: number) {
  const windows: [string, string][] = [];
  for (let i = 0; windows.length <= MAX_WINDOWS; i++) {
    const ws = new Date(`${start}T00:00:00Z`);
    ws.setUTCMonth(ws.getUTCMonth() + i * interval);
    if (isoDay(ws) > end) break;
    const we = new Date(ws);
    we.setUTCMonth(we.getUTCMonth() + length);
    we.setUTCDate(we.getUTCDate() - 1);
    windows.push([isoDay(ws), isoDay(we) > end ? end : isoDay(we)]);
  }
  return windows;
}

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

function ParameterField({
  p,
  name,
  value,
  onChange,
}: {
  p: MethodologyParameter;
  name: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const extra = EXTRAS[p.name];
  const label = extra?.label ?? p.label;
  const inRange = (n: number) => n >= p.min && n <= p.max;
  const help = extra && <InfoTip label={label.toLowerCase()}>{extra.help}</InfoTip>;

  if (extra?.choices) {
    return (
      <div className="flex flex-col gap-1.5">
        <input type="hidden" name={name} value={value} />
        <span className={`${labelClass} flex items-center gap-1.5`}>
          {label}
          {help}
        </span>
        <div className="grid grid-cols-3 gap-2" role="radiogroup" aria-label={label}>
          {extra.choices
            .filter((c) => inRange(c.value))
            .map((c) => (
              <button
                key={c.value}
                type="button"
                role="radio"
                aria-checked={Number(value) === c.value}
                onClick={() => onChange(String(c.value))}
                className={choiceClass(Number(value) === c.value)}
              >
                <span className="block text-label font-semibold">{c.label}</span>
                <span className="block font-mono text-micro opacity-80">≥ {c.value.toFixed(2)}</span>
                <span className="mt-0.5 block text-micro opacity-80">{c.help}</span>
              </button>
            ))}
        </div>
      </div>
    );
  }

  const id = name.replace(/[^a-z0-9_-]/gi, '-');
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className={labelClass}>
        <span className="flex items-center gap-1.5">
          {label}
          {help}
        </span>
      </label>
      <div className="flex flex-wrap items-center gap-2">
        {extra?.presets?.filter(inRange).map((n) => (
          <button
            key={n}
            type="button"
            onClick={() => onChange(String(n))}
            className={`${choiceClass(Number(value) === n)} px-3 py-1.5 font-mono text-label`}
          >
            {n}
            {extra.unit}
          </button>
        ))}
        <div className="flex items-center gap-1">
          <input
            id={id}
            name={name}
            type="number"
            min={p.min}
            max={p.max}
            step="any"
            required
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className={`${inputClass} w-24 py-1.5`}
          />
          {extra?.unit && <span className="font-mono text-label text-muted-foreground">{extra.unit}</span>}
        </div>
      </div>
      <p className="text-caption text-muted-foreground">
        {extra?.note ?? `Between ${p.min} and ${p.max}. Default ${p.default}.`}
      </p>
    </div>
  );
}

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
  const first = methodologies.find((m) => keyOf(m) === initial?.methodology) ?? methodologies[0];
  const [open, setOpen] = useState(false);
  const [range] = useState(defaultRange);
  const [dateStart, setDateStart] = useState(initial?.dateStart ?? range.start);
  const [dateEnd, setDateEnd] = useState(initial?.dateEnd ?? range.end);
  const [selectedKeys, setSelectedKeys] = useState<string[]>(first ? [keyOf(first)] : []);
  const [values, setValues] = useState<Record<string, Record<string, string>>>(() =>
    first ? { [keyOf(first)]: defaultsFor(first, initial?.parameters) } : {},
  );
  const [mode, setMode] = useState<'single' | 'series'>('single');
  const [windowMonths, setWindowMonths] = useState(1);
  const [intervalMonths, setIntervalMonths] = useState(3);

  const [state, formAction, pending] = useActionState(
    async (prev: SnapshotFormState, formData: FormData) => {
      const result = await runSnapshotAction(prev, formData);
      if (result.ok) {
        setOpen(false);
        window.dispatchEvent(new Event(ACTIVITY_CHANGED));
      }
      return result;
    },
    {},
  );

  const selected = methodologies.filter((m) => selectedKeys.includes(keyOf(m)));
  const combined = selected.length > 1;
  // The API rejects start dates before any chosen methodology's data begins.
  const earliest = selected.map((m) => m.data_start.slice(0, 10)).sort().at(-1);
  // ...and end dates after the latest day any of them has data for (CHIRPS lags a few weeks).
  const lagging = selected
    .filter((m) => m.data_end)
    .sort((a, b) => a.data_end!.localeCompare(b.data_end!))
    .at(0);
  const latest = lagging?.data_end?.slice(0, 10);

  // A series takes one methodology, so there choosing one replaces the other.
  const toggle = (m: Methodology) => {
    const key = keyOf(m);
    setValues((prev) => (prev[key] ? prev : { ...prev, [key]: defaultsFor(m) }));
    setSelectedKeys((prev) =>
      mode === 'series' ? [key] : prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key],
    );
  };
  const chooseMode = (next: 'single' | 'series') => {
    setMode(next);
    if (next === 'series') setSelectedKeys((prev) => prev.slice(0, 1));
  };
  const today = isoDay(new Date());
  const lastEnd = latest && latest < today ? latest : today;
  const pastData = !!latest && dateEnd > latest;
  const windows = mode === 'series' ? seriesWindows(dateStart, dateEnd, windowMonths, intervalMonths) : [];
  const tooMany = windows.length > MAX_WINDOWS;

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
          <input type="hidden" name="mode" value={mode} />
          <input type="hidden" name="window_months" value={windowMonths} />
          <input type="hidden" name="interval_months" value={intervalMonths} />

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

            {!initial && (
              <div className="grid grid-cols-2 gap-1 rounded-md bg-surface-container p-1" role="radiogroup" aria-label="Run type">
                {(['single', 'series'] as const).map((m) => (
                  <button
                    key={m}
                    type="button"
                    role="radio"
                    aria-checked={mode === m}
                    onClick={() => chooseMode(m)}
                    className={`rounded-sm px-3 py-2 text-label font-semibold transition-colors ${
                      mode === m ? 'bg-primary text-primary-foreground' : 'hover:bg-surface-lowest'
                    }`}
                  >
                    {m === 'single' ? 'Single snapshot' : 'Time series'}
                  </button>
                ))}
              </div>
            )}

            <fieldset className="flex flex-col gap-2">
              <legend className="mb-2 flex w-full justify-between text-label font-semibold">
                <span className="flex items-center gap-1.5">
                  {mode === 'series' ? 'Series range' : 'Observation window'}
                  <InfoTip label={mode === 'series' ? 'the series range' : 'the observation window'}>
                    {mode === 'series'
                      ? HELP.series
                      : ((!combined && methodologyView(selected[0]?.slug)?.windowHelp) || HELP.window)}
                  </InfoTip>
                </span>
                <span className="font-mono text-micro font-normal text-muted-foreground">
                  {mode === 'series' ? 'UTC · up to 24 windows' : 'UTC · max one year'}
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
                    min={earliest}
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
                    max={lastEnd}
                  />
                </div>
              </div>
              {pastData && lagging && (
                <p role="alert" className="text-caption text-danger">
                  {lagging.name} has data up to {latest} so far; it is published a few weeks after each month ends.
                  Choose an end on or before {latest}.
                </p>
              )}
            </fieldset>

            {mode === 'series' && (
              <div className="flex flex-col gap-3 rounded-md border border-border p-4">
                <div className="grid grid-cols-2 gap-3">
                  {(
                    [
                      ['Window length', windowMonths, setWindowMonths, HELP.windowLength],
                      ['A window every', intervalMonths, setIntervalMonths, HELP.interval],
                    ] as const
                  ).map(([label, value, set, help]) => (
                    <label key={label} className="flex flex-col gap-1.5">
                      <span className={`${labelClass} flex items-center gap-1.5`}>
                        {label}
                        <InfoTip label={label.toLowerCase()}>{help}</InfoTip>
                      </span>
                      <select value={value} onChange={(e) => set(Number(e.target.value))} className={`${inputClass} appearance-none`}>
                        {MONTH_CHOICES.map((n) => (
                          <option key={n} value={n}>
                            {n} {n === 1 ? 'month' : 'months'}
                          </option>
                        ))}
                      </select>
                    </label>
                  ))}
                </div>
                <div>
                  <span className={labelClass}>
                    {tooMany ? `Over ${MAX_WINDOWS} windows: shorten the range or widen the interval` : `${windows.length} windows`}
                  </span>
                  <ol className="mt-1.5 max-h-32 overflow-y-auto font-mono text-caption text-muted-foreground">
                    {windows.slice(0, MAX_WINDOWS).map(([ws, we], i) => (
                      <li key={ws}>
                        {String(i + 1).padStart(2, '0')}. {ws} → {we}
                      </li>
                    ))}
                  </ol>
                </div>
                <p className="text-caption text-muted-foreground">
                  Each window is its own snapshot, sealed on its own. When all have finished, the series is sealed
                  with one hash over every window&apos;s result, so none can be left out later.
                </p>
              </div>
            )}

            <fieldset className="flex flex-col gap-2">
              <legend className="mb-2 flex w-full justify-between text-label font-semibold">
                <span className="flex items-center gap-1.5">
                  {mode === 'series' ? 'Processing methodology' : 'Processing methodologies'}
                  <InfoTip label="a methodology">{HELP.methodology}</InfoTip>
                </span>
                <span className="font-mono text-micro font-normal text-muted-foreground">
                  {mode === 'series' ? 'one per series' : 'choose one or more'}
                </span>
              </legend>
              {methodologies.map((m) => {
                const checked = selectedKeys.includes(keyOf(m));
                return (
                  <label
                    key={m.id}
                    className={`flex cursor-pointer gap-3 rounded-sm border p-3 transition-colors ${
                      checked ? 'border-primary bg-surface-lowest' : 'border-border hover:border-outline'
                    }`}
                  >
                    <input
                      type={mode === 'series' ? 'radio' : 'checkbox'}
                      name="methodology"
                      value={keyOf(m)}
                      checked={checked}
                      onChange={() => toggle(m)}
                      className="mt-1 size-4 shrink-0 accent-primary"
                    />
                    <span>
                      <span className="block text-label font-semibold">
                        {m.name} <span className="font-mono text-micro text-muted-foreground">v{m.version}</span>
                      </span>
                      <span className="block text-caption text-muted-foreground">
                        {m.description} Data from {m.data_start.slice(0, 10)}
                        {m.data_end ? ` to ${m.data_end.slice(0, 10)}` : ''}.
                      </span>
                    </span>
                  </label>
                );
              })}
              {combined && (
                <p className="text-caption text-muted-foreground">
                  Each methodology runs as its own snapshot for the same window. When all have finished, they are
                  sealed together with one hash, so none can be left out later.
                </p>
              )}
            </fieldset>

            {selected.map((m) => {
              const key = keyOf(m);
              const params = [...m.parameters].sort((a, b) => rank(a.name) - rank(b.name));
              if (params.length === 0) return null;
              return (
                <fieldset key={key} className="flex flex-col gap-4 rounded-md border border-border p-4">
                  <legend className="flex items-center gap-1.5 px-1 text-label font-semibold">
                    <SlidersHorizontal className="size-4" />
                    {combined ? `Settings · ${m.name}` : 'Settings'}
                  </legend>
                  {params.map((p) => (
                    <ParameterField
                      key={p.name}
                      p={p}
                      name={`${PARAM_PREFIX}${key}:${p.name}`}
                      value={values[key]?.[p.name] ?? ''}
                      onChange={(v) => setValues((prev) => ({ ...prev, [key]: { ...prev[key], [p.name]: v } }))}
                    />
                  ))}
                  <p className="font-mono text-micro text-muted-foreground">
                    These settings are saved with the snapshot and sealed into its record hash.
                  </p>
                </fieldset>
              );
            })}

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
            <Button
              type="submit"
              size="lg"
              disabled={pending || selected.length === 0 || tooMany || pastData || (mode === 'series' && !windows.length)}
            >
              <CirclePlay />
              {pending
                ? 'Queuing…'
                : mode === 'series'
                  ? `Start ${windows.length} snapshots`
                  : combined
                    ? `Start ${selected.length} together`
                    : 'Start snapshot'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};

export default RunSnapshotDialog;
