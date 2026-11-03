import { cookies } from 'next/headers';

import { ACCESS_COOKIE } from '@/lib/session';

const API_URL = process.env.API_URL ?? 'http://localhost:8080';

// The map loads the NDVI image straight from the browser, which can't send the API's bearer
// token, so this passes the request on with it. No ".png" in the path keeps it inside the
// proxy matcher, so an expiring session is refreshed here too.
export async function GET(_request: Request, { params }: { params: Promise<{ ref: string }> }) {
  const { ref } = await params;
  const token = (await cookies()).get(ACCESS_COOKIE)?.value;
  if (!token) return new Response(null, { status: 401 });

  let res: Response;
  try {
    res = await fetch(`${API_URL}/snapshots/${encodeURIComponent(ref)}/ndvi.png`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
    });
  } catch {
    return new Response(null, { status: 502 });
  }
  if (!res.ok) return new Response(null, { status: res.status });

  return new Response(res.body, {
    headers: { 'Content-Type': 'image/png', 'Cache-Control': 'private, max-age=86400' },
  });
}
