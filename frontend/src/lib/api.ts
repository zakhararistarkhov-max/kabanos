// Client-side API helper. Everything goes through the same-origin BFF
// (/api/*), so there are no tokens or CORS to manage here.

export class ApiRequestError extends Error {
  status: number;
  code: string;
  fields?: Record<string, string>;
  constructor(message: string, status: number, code: string, fields?: Record<string, string>) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "content-type": "application/json", ...(init?.headers ?? {}) },
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const err = data?.error ?? { code: "error", message: "Request failed" };
    throw new ApiRequestError(err.message, res.status, err.code, err.fields);
  }
  return data as T;
}

/** Calls an authenticated domain endpoint via the proxy. */
export function api<T>(path: string, init?: RequestInit): Promise<T> {
  return request<T>(`/api/proxy${path}`, init);
}

/** Calls a BFF auth endpoint (login, session, …). */
export function authApi<T>(path: string, init?: RequestInit): Promise<T> {
  return request<T>(`/api/auth${path}`, init);
}

/** The browser's IANA timezone, sent so the server buckets days correctly. */
export function browserTZ(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

/** Today's date as YYYY-MM-DD in the browser's timezone. */
export function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

/** Returns YYYY-MM-DD for `days` ago from today (local). */
export function isoDaysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
