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

/** Attempts to rotate the refresh token; returns true on success. */
async function tryRefresh(): Promise<boolean> {
  const store = await cookies();
  const rt = store.get(COOKIE_REFRESH)?.value;
  if (!rt) return false;

  const res = await rawBackend("/api/v1/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refreshToken: rt }),
  });
  if (!res.ok) {
    await clearSession();
    return false;
  }
  await setSession((await res.json()) as Tokens);
  return true;
}

/**
 * backendFetch calls an authenticated API endpoint, transparently refreshing
 * the access token once on 401 and retrying. Returns the raw upstream Response
 * so callers can forward status and body verbatim.
 */
export async function backendFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const doFetch = async (): Promise<Response> => {
    const store = await cookies();
    const at = store.get(COOKIE_ACCESS)?.value;
    const headers = new Headers(init.headers);
    if (at) headers.set("Authorization", `Bearer ${at}`);
    if (init.body && !headers.has("content-type")) headers.set("content-type", "application/json");
    return fetch(backendBaseURL() + path, { ...init, headers, cache: "no-store" });
  };

  let res = await doFetch();
  if (res.status === 401 && (await tryRefresh())) {
    res = await doFetch();
  }
  return res;
}
