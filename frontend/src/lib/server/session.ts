// Server-only session layer (the "BFF"). The browser talks only to Next.js
// route handlers; those handlers hold the access/refresh tokens in httpOnly
// cookies and forward authenticated requests to the Go API. Tokens never reach
// client JavaScript, which closes the door on XSS token theft.

import { cookies } from "next/headers";
import {
  backendBaseURL,
  COOKIE_ACCESS,
  COOKIE_REFRESH,
  cookieOptions,
} from "@/lib/config";

export interface Tokens {
  accessToken: string;
  accessExpiresAt: string;
  refreshToken: string;
  refreshExpiresAt: string;
}

function ttlSeconds(iso: string): number {
  return Math.max(1, Math.floor((new Date(iso).getTime() - Date.now()) / 1000));
}

export async function setSession(tokens: Tokens): Promise<void> {
  const store = await cookies();
  store.set(COOKIE_ACCESS, tokens.accessToken, cookieOptions(ttlSeconds(tokens.accessExpiresAt)));
  store.set(COOKIE_REFRESH, tokens.refreshToken, cookieOptions(ttlSeconds(tokens.refreshExpiresAt)));
}

export async function clearSession(): Promise<void> {
  const store = await cookies();
  store.delete(COOKIE_ACCESS);
  store.delete(COOKIE_REFRESH);
}

export async function hasRefreshToken(): Promise<boolean> {
  const store = await cookies();
  return Boolean(store.get(COOKIE_REFRESH)?.value);
}

/** rawBackend calls a public (unauthenticated) API endpoint. */
export async function rawBackend(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("content-type")) headers.set("content-type", "application/json");
  return fetch(backendBaseURL() + path, { ...init, headers, cache: "no-store" });
}

// Single-flight refresh, keyed by the refresh token. Refresh tokens ROTATE
// (each use revokes the previous one and the backend treats reuse as theft), so
// when several authenticated requests hit 401 at once — e.g. the widget-heavy
// dashboard after the access token expires — they must NOT each call /refresh.
// Concurrent requests carrying the same refresh token coalesce onto one call
// and share its result. Keying by token keeps different users' refreshes
// separate even though the map is process-global.
const inflightRefresh = new Map<string, Promise<Tokens | null>>();

function sharedRefresh(refreshToken: string): Promise<Tokens | null> {
  const existing = inflightRefresh.get(refreshToken);
  if (existing) return existing;

  const p = (async (): Promise<Tokens | null> => {
    const res = await rawBackend("/api/v1/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return null;
    return (await res.json()) as Tokens;
  })().finally(() => inflightRefresh.delete(refreshToken));

  inflightRefresh.set(refreshToken, p);
  return p;
}

/**
 * backendFetch calls an authenticated API endpoint, transparently refreshing
 * the access token once on 401 and retrying. Returns the raw upstream Response
 * so callers can forward status and body verbatim.
 */
export async function backendFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const doFetch = async (accessToken?: string): Promise<Response> => {
    const headers = new Headers(init.headers);
    const at = accessToken ?? (await cookies()).get(COOKIE_ACCESS)?.value;
    if (at) headers.set("Authorization", `Bearer ${at}`);
    if (init.body && !headers.has("content-type")) headers.set("content-type", "application/json");
    return fetch(backendBaseURL() + path, { ...init, headers, cache: "no-store" });
  };

  const res = await doFetch();
  if (res.status !== 401) return res;

  const rt = (await cookies()).get(COOKIE_REFRESH)?.value;
  if (!rt) return res;

  const tokens = await sharedRefresh(rt);
  if (!tokens) {
    await clearSession();
    return res;
  }
  // Update this request's cookies and retry with the freshly minted access
  // token (not the stale one still in the cookie store).
  await setSession(tokens);
  return doFetch(tokens.accessToken);
}
