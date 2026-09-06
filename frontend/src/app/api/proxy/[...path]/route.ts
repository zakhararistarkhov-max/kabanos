import { NextRequest, NextResponse } from "next/server";
import { backendFetch } from "@/lib/server/session";

// Generic authenticated proxy: /api/proxy/<...> → GET/POST/... /api/v1/<...>.
// Access-token refresh is handled transparently by backendFetch.
async function handle(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  const target = `/api/v1/${path.join("/")}${req.nextUrl.search}`;

  const init: RequestInit = { method: req.method };
  if (req.method !== "GET" && req.method !== "HEAD") {
    const body = await req.text();
    if (body) init.body = body;
  }

  const res = await backendFetch(target, init);
  const text = await res.text();
  return new NextResponse(text || null, {
    status: res.status,
    headers: { "content-type": res.headers.get("content-type") ?? "application/json" },
  });
}

export const GET = handle;
export const POST = handle;
export const PUT = handle;
export const PATCH = handle;
export const DELETE = handle;
