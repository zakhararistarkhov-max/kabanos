// Server-side configuration and cookie contract for the BFF layer. The browser
// never sees the backend URL or the tokens — only httpOnly cookies set here.

/** Base URL of the Go API, reachable from the Next.js server (not the browser). */
export function backendBaseURL(): string {
  return process.env.BACKEND_INTERNAL_URL ?? "http://localhost:8080";
}

export const COOKIE_ACCESS = "kab_at";
export const COOKIE_REFRESH = "kab_rt";

const isProd = process.env.NODE_ENV === "production";

// The Secure flag defaults to on in production, but can be forced off for an
// HTTP-only deployment (plain-IP box without TLS, or TLS terminated elsewhere).
// Over plain HTTP browsers drop Secure cookies, which would break login — so
// set COOKIE_SECURE=false in that case.
const cookieSecure =
  process.env.COOKIE_SECURE != null && process.env.COOKIE_SECURE !== ""
    ? process.env.COOKIE_SECURE === "true"
    : isProd;

/** Cookie options shared by both tokens. httpOnly keeps them out of JS reach. */
export function cookieOptions(maxAgeSeconds: number) {
  return {
    httpOnly: true,
    secure: cookieSecure,
    sameSite: "lax" as const,
    path: "/",
    maxAge: maxAgeSeconds,
  };
}
