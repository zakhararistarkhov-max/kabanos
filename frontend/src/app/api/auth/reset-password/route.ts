import { NextRequest, NextResponse } from "next/server";
import { rawBackend } from "@/lib/server/session";

export async function POST(req: NextRequest) {
  const body = await req.text();
  const res = await rawBackend("/api/v1/auth/reset-password", { method: "POST", body });
  const data = await res.json().catch(() => ({}));
  return NextResponse.json(data, { status: res.status });
}
