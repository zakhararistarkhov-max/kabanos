import { NextRequest, NextResponse } from "next/server";
import { rawBackend, setSession } from "@/lib/server/session";

export async function POST(req: NextRequest) {
  const body = await req.text();
  const res = await rawBackend("/api/v1/auth/register", { method: "POST", body });
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    return NextResponse.json(data ?? { error: { code: "error", message: "registration failed" } }, { status: res.status });
  }
  const { user, ...tokens } = data;
  await setSession(tokens);
  return NextResponse.json({ user }, { status: 201 });
}
