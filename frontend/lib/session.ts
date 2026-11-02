// Session cookies, shared by the auth server actions and proxy.ts.
// No "use server": proxy.ts imports this too.

export const ACCESS_COOKIE = "access_token";
export const REFRESH_COOKIE = "refresh_token";
// Present only when the user ticked "remember me". The proxy can't see a
// cookie's expiry, so this tells it whether a refreshed refresh token should
// persist (30 days) or stay a session cookie.
export const REMEMBER_COOKIE = "remember_session";

// Match the Go API's ACCESS_TOKEN_TTL / REFRESH_TOKEN_TTL defaults.
const ACCESS_MAX_AGE = 15 * 60;
const REFRESH_MAX_AGE = 30 * 24 * 60 * 60;

// Refresh this long before the access token expires, so a request doesn't
// leave with a token that dies on the way.
const EXPIRY_MARGIN_MS = 30_000;

export type TokenPair = {
  access_token: string;
  refresh_token: string;
};

export type SessionCookie = {
  name: string;
  value: string;
  httpOnly: true;
  secure: boolean;
  sameSite: "lax";
  path: "/";
  maxAge?: number;
};

// The cookies to set after login, register or refresh.
export function sessionCookies(tokens: TokenPair, remember: boolean): SessionCookie[] {
  const base = {
    httpOnly: true as const,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax" as const,
    path: "/" as const,
  };
  // Without "remember me" the refresh token is a session cookie and is
  // cleared when the browser closes.
  const persist = remember ? { maxAge: REFRESH_MAX_AGE } : {};

  const cookies: SessionCookie[] = [
    { ...base, name: ACCESS_COOKIE, value: tokens.access_token, maxAge: ACCESS_MAX_AGE },
    { ...base, name: REFRESH_COOKIE, value: tokens.refresh_token, ...persist },
  ];
  if (remember) {
    cookies.push({ ...base, name: REMEMBER_COOKIE, value: "1", ...persist });
  }
  return cookies;
}

// True when the token is missing, unreadable, or expires within the margin.
// This only decides whether to refresh; the Go API still verifies every token.
export function accessTokenNeedsRefresh(token: string | undefined): boolean {
  if (!token) return true;
  try {
    const payload = token.split(".")[1];
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    const { exp } = JSON.parse(json) as { exp?: number };
    return typeof exp !== "number" || exp * 1000 - Date.now() < EXPIRY_MARGIN_MS;
  } catch {
    return true;
  }
}
