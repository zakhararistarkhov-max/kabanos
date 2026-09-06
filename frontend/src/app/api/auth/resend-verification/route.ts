import { NextResponse } from "next/server";
import { backendFetch } from "@/lib/server/session";

export async function POST() {
  const res = await backendFetch("/api/v1/auth/resend-verification", { method: "POST" });
  const data = await res.json().catch(() => ({}));
  return NextResponse.json(data, { status: res.status });
}
