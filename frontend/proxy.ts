import { NextResponse, type NextRequest } from "next/server";

import {
  ACCESS_COOKIE,
  REFRESH_COOKIE,
  REMEMBER_COOKIE,
  accessTokenNeedsRefresh,
  sessionCookies,
  type TokenPair,
} from "@/lib/session";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

// Keeps the session alive. Access tokens last 15 minutes and Server Components
// can't set cookies, so the refresh happens here, before the page renders.
//
// This only refreshes tokens; it doesn't decide who may see a page. Pages and
// layouts check the session, and the Go API checks every request.
export async function proxy(request: NextRequest) {
  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  const refresh = request.cookies.get(REFRESH_COOKIE)?.value;

  if (!refresh || !accessTokenNeedsRefresh(access)) {
    return NextResponse.next();
  }

  let res: Response;
  try {
    res = await fetch(`${API_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refresh }),
    });
  } catch {
    // API unreachable: leave the cookies alone and let the page show the error.
    return NextResponse.next();
  }

  if (res.status === 401) {
    // Refresh token expired, revoked or reused too late: the session is over.
    return endSession(request);
  }
  if (!res.ok) {
    // Server trouble, not a bad session: keep the cookies and try next request.
    return NextResponse.next();
  }

  const tokens = (await res.json()) as TokenPair;
  const cookies = sessionCookies(tokens, request.cookies.has(REMEMBER_COOKIE));

  // Two copies of the new tokens: on the forwarded request, so the page being
  // rendered right now uses them, and on the response, so the browser keeps them.
  for (const cookie of cookies) {
    request.cookies.set(cookie.name, cookie.value);
  }
  const response = NextResponse.next({ request: { headers: request.headers } });
  for (const cookie of cookies) {
    response.cookies.set(cookie);
  }
  return response;
}

function endSession(request: NextRequest) {
  for (const name of [ACCESS_COOKIE, REFRESH_COOKIE, REMEMBER_COOKIE]) {
    request.cookies.delete(name);
  }
  const response = NextResponse.next({ request: { headers: request.headers } });
  for (const name of [ACCESS_COOKIE, REFRESH_COOKIE, REMEMBER_COOKIE]) {
    response.cookies.delete(name);
  }
  return response;
}

export const config = {
  matcher: [
    {
      // Pages only: skip static files, image optimisation and metadata files.
      source: "/((?!_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico)$).*)",
      // Skip prefetches: links on a page are prefetched in parallel, and each
      // would otherwise try to refresh with the same single-use token.
      missing: [
        { type: "header", key: "next-router-prefetch" },
        { type: "header", key: "purpose", value: "prefetch" },
      ],
    },
  ],
};
