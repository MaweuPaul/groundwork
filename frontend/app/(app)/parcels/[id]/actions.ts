'use server';

import { revalidatePath } from 'next/cache';

import { createSnapshot } from '@/lib/api';

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

  const result = await createSnapshot(parcelId, {
    methodology,
    methodology_version: methodologyVersion,
    date_start: values.date_start,
    date_end: values.date_end,
  });
  if (!result.ok) return { error: result.error, values };

  revalidatePath(`/parcels/${parcelId}`);
  return { ok: true };
}
