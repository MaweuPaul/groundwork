'use server';

import { revalidatePath } from 'next/cache';

import { createSeries, createSnapshot } from '@/lib/api';

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
  const values = {
    methodology: field(formData, 'methodology'),
    date_start: field(formData, 'date_start'),
    date_end: field(formData, 'date_end'),
  };

  // The select's value is "slug@version".
  const [methodology, methodologyVersion] = values.methodology.split('@');
  if (!methodology || !methodologyVersion) return { error: 'Choose a methodology.', values };
  if (!values.date_start || !values.date_end) return { error: 'Choose a start and end date.', values };

  // Range checks happen in the API, which owns the rules and returns a readable message.
  const cloudPct = Number(field(formData, 'max_scene_cloud_pct'));
  const threshold = Number(field(formData, 'cloud_score_threshold'));
  if (!Number.isFinite(cloudPct) || !Number.isFinite(threshold)) {
    return { error: 'Cloud settings must be numbers.', values };
  }

  const request = {
    methodology,
    methodology_version: methodologyVersion,
    date_start: values.date_start,
    date_end: values.date_end,
    max_scene_cloud_pct: cloudPct,
    cloud_score_threshold: threshold,
  };
  const result =
    field(formData, 'mode') === 'series'
      ? await createSeries(parcelId, {
          ...request,
          window_months: Number(field(formData, 'window_months')),
          interval_months: Number(field(formData, 'interval_months')),
        })
      : await createSnapshot(parcelId, request);
  if (!result.ok) return { error: result.error, values };

  revalidatePath('/parcels/[slug]', 'page');
  return { ok: true };
}
