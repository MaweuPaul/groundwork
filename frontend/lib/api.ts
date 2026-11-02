import "server-only";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { ACCESS_COOKIE } from "@/lib/session";
import type {
  NewParcel,
  NewSnapshot,
  Parcel,
  QueuedSnapshot,
  Snapshot,
} from "@/lib/types";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; error: string };

type Options = {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  public?: boolean;
};

export async function api<T>(path: string, options: Options = {}): Promise<ApiResult<T>> {
  const { method = "GET", body, public: isPublic = false } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";

  if (!isPublic) {
    const token = (await cookies()).get(ACCESS_COOKIE)?.value;
    if (!token) redirect("/login");
    headers.Authorization = `Bearer ${token}`;
  }

  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return { ok: false, status: 0, error: "Can't reach the server. Try again in a moment." };
  }

  // redirect() throws, so it must stay outside the try above.
  if (res.status === 401 && !isPublic) redirect("/login");

  const json = await res.json().catch(() => null);
  if (!res.ok) {
    return { ok: false, status: res.status, error: json?.error ?? "Something went wrong." };
  }
  return { ok: true, data: json as T };
}

const id = (value: string) => encodeURIComponent(value);

export const getParcels = () => api<Parcel[]>("/parcels");

export const getParcel = (parcelId: string) => api<Parcel>(`/parcels/${id(parcelId)}`);

export const createParcel = (parcel: NewParcel) =>
  api<Parcel>("/parcels", { method: "POST", body: parcel });

export const createSnapshot = (parcelId: string, snapshot: NewSnapshot) =>
  api<QueuedSnapshot>(`/parcels/${id(parcelId)}/snapshots`, { method: "POST", body: snapshot });

export const getSnapshot = (snapshotId: string) => api<Snapshot>(`/snapshots/${id(snapshotId)}`);
