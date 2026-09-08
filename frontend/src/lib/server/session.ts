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

// Result of a completed rotation, keyed by the OLD refresh token. Because
// refresh tokens rotate (each use revokes the previous), a request that reaches
// the refresh step just AFTER the single-flight promise resolved would still be
// holding the old token in its cookie and, without this, would call /refresh
// again with a now-revoked token — which the backend flags as reuse/theft and
// rejects (401). Caching the rotation briefly lets these stragglers reuse the
// freshly minted tokens instead of re-refreshing. TTL only needs to outlast a
// single page-load burst.
const ROTATED_TTL_MS = 30_000;
const recentlyRotated = new Map<string, { tokens: Tokens; at: number }>();

function rememberRotation(oldRefreshToken: string, tokens: Tokens): void {
  const now = Date.now();
  recentlyRotated.set(oldRefreshToken, { tokens, at: now });
  for (const [k, v] of recentlyRotated) {
    if (now - v.at > ROTATED_TTL_MS) recentlyRotated.delete(k);
  }
}

function recentRotation(oldRefreshToken: string): Tokens | null {
  const hit = recentlyRotated.get(oldRefreshToken);
  if (!hit) return null;
  if (Date.now() - hit.at > ROTATED_TTL_MS) {
    recentlyRotated.delete(oldRefreshToken);
    return null;
  }
  return hit.tokens;
}

function sharedRefresh(refreshToken: string): Promise<Tokens | null> {
  const existing = inflightRefresh.get(refreshToken);
  if (existing) return existing;

  const p = (async (): Promise<Tokens | null> => {
    const res = await rawBackend("/api/v1/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return null;
    const tokens = (await res.json()) as Tokens;
    rememberRotation(refreshToken, tokens);
    return tokens;
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

  // If this refresh token was just rotated by a concurrent request, reuse that
  // result instead of calling /refresh again with a now-revoked token.
  const cached = recentRotation(rt);
  if (cached) {
    await setSession(cached);
    return doFetch(cached.accessToken);
  }

  const tokens = await sharedRefresh(rt);
  if (!tokens) {
    // Lost a rotation race after our single-flight resolved? Retry the cache
    // once before giving up, so a straggler doesn't needlessly clear a session
    // that a sibling request just refreshed.
    const raced = recentRotation(rt);
    if (raced) {
      await setSession(raced);
      return doFetch(raced.accessToken);
    }
    await clearSession();
    return res;
  }
  // Update this request's cookies and retry with the freshly minted access
  // token (not the stale one still in the cookie store).
  await setSession(tokens);
  return doFetch(tokens.accessToken);
}
