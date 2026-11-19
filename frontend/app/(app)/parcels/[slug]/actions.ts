'use server';

import { revalidatePath } from 'next/cache';

import { createGroup, createSeries, createSnapshot } from '@/lib/api';
import { PARAM_PREFIX } from '@/lib/snapshot-form';

export type SnapshotFormState = {
  ok?: boolean;
  error?: string;
  values?: { methodology: string; date_start: string; date_end: string };
};

const field = (formData: FormData, name: string) => String(formData.get(name) ?? '').trim();

export async function runSnapshotAction(
  _prev: SnapshotFormState,
  formData: FormData,
): Promise<SnapshotFormState> {
  const parcelId = field(formData, 'parcel_id');
  // Each chosen methodology is "slug@version".
  const keys = formData.getAll('methodology').map(String);
  const values = {
    methodology: keys.join(','),
    date_start: field(formData, 'date_start'),
    date_end: field(formData, 'date_end'),
  };
  const series = field(formData, 'mode') === 'series';

  if (keys.length === 0) return { error: 'Choose a methodology.', values };
  if (series && keys.length > 1) return { error: 'A series takes one methodology.', values };
  if (!values.date_start || !values.date_end) return { error: 'Choose a start and end date.', values };

  // Inputs named "param:<slug@version>:<name>" are each methodology's settings. Range checks
  // happen in the API, which owns the rules and returns a readable message.
  const parameters: Record<string, Record<string, number>> = {};
  for (const [name, raw] of formData) {
    if (!name.startsWith(PARAM_PREFIX)) continue;
    const [key, param] = name.slice(PARAM_PREFIX.length).split(':');
    const text = String(raw).trim();
    const value = Number(text);
    if (text === '' || !Number.isFinite(value)) return { error: 'Settings must be numbers.', values };
    (parameters[key] ??= {})[param] = value;
  }

  const members = keys.map((key) => {
    const [methodology, methodologyVersion] = key.split('@');
    return { methodology, methodology_version: methodologyVersion, parameters: parameters[key] ?? {} };
  });
  const dates = { date_start: values.date_start, date_end: values.date_end };

  const result = series
    ? await createSeries(parcelId, {
        ...members[0],
        ...dates,
        window_months: Number(field(formData, 'window_months')),
        interval_months: Number(field(formData, 'interval_months')),
      })
    : members.length > 1
      ? await createGroup(parcelId, { ...dates, members })
      : await createSnapshot(parcelId, { ...members[0], ...dates });
  if (!result.ok) return { error: result.error, values };

  revalidatePath('/parcels/[slug]', 'page');
  return { ok: true };
}
