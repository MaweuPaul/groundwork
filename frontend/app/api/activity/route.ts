import { cookies } from 'next/headers';

import { ACCESS_COOKIE } from '@/lib/session';

const API_URL = process.env.API_URL ?? 'http://localhost:8080';

// The snapshot notifier polls from the browser, which can't send the API's bearer token,
// so this passes the request on with it.
export async function GET() {
  const token = (await cookies()).get(ACCESS_COOKIE)?.value;
  if (!token) return Response.json({ error: 'signed out' }, { status: 401 });

  let res: Response;
  try {
    res = await fetch(`${API_URL}/activity?limit=10`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
    });
  } catch {
    return Response.json({ error: 'api unreachable' }, { status: 502 });
  }
  return new Response(res.body, {
    status: res.status,
    headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' },
  });
}
