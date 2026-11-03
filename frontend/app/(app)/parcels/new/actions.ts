'use server';

import { redirect } from 'next/navigation';

import { createParcel } from '@/lib/api';
import { parcelPath } from '@/lib/paths';
import type { MultiPolygon, Polygon } from '@/lib/types';

export type ParcelFormState = {
  error?: string;
  fieldErrors?: { name?: string; geometry?: string };
  values?: { name: string; region: string; description: string };
};

function parseGeometry(raw: string): Polygon | MultiPolygon | null {
  try {
    const geometry = JSON.parse(raw);
    return geometry?.type === 'Polygon' || geometry?.type === 'MultiPolygon' ? geometry : null;
  } catch {
    return null;
  }
}

const field = (formData: FormData, name: string) => String(formData.get(name) ?? '').trim();

export async function createParcelAction(
  _prev: ParcelFormState,
  formData: FormData,
): Promise<ParcelFormState> {
  const values = {
    name: field(formData, 'name'),
    region: field(formData, 'region'),
    description: field(formData, 'description'),
  };
  const geometry = parseGeometry(field(formData, 'geometry'));

  const fieldErrors: ParcelFormState['fieldErrors'] = {};
  if (!values.name) fieldErrors.name = 'Give the parcel a name.';
  if (!geometry) fieldErrors.geometry = 'Draw the boundary on the map or upload a GeoJSON file.';
  if (fieldErrors.name || !geometry) return { fieldErrors, values };

  const metadata: Record<string, string> = {};
  if (values.region) metadata.region = values.region;

  const result = await createParcel({
    name: values.name,
    description: values.description,
    geometry,
    metadata,
  });
  if (!result.ok) return { error: result.error, values };

  redirect(parcelPath(result.data));
}
