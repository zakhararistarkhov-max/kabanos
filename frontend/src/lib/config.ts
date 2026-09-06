// Server-side configuration and cookie contract for the BFF layer. The browser
// never sees the backend URL or the tokens — only httpOnly cookies set here.

/** Base URL of the Go API, reachable from the Next.js server (not the browser). */
export function backendBaseURL(): string {
  return process.env.BACKEND_INTERNAL_URL ?? "http://localhost:8080";
}

export const COOKIE_ACCESS = "kab_at";
export const COOKIE_REFRESH = "kab_rt";

const isProd = process.env.NODE_ENV === "production";

/** Cookie options shared by both tokens. httpOnly keeps them out of JS reach. */
export function cookieOptions(maxAgeSeconds: number) {
  return {
    httpOnly: true,
    secure: isProd,
    sameSite: "lax" as const,
    path: "/",
    maxAge: maxAgeSeconds,
  };
}
