import { NextRequest, NextResponse } from "next/server";
import { rawBackend, setSession } from "@/lib/server/session";

// Exchanges credentials for a session. On success the tokens are stored in
// httpOnly cookies and only the user object is returned to the browser.
export async function POST(req: NextRequest) {
  const body = await req.text();
  const res = await rawBackend("/api/v1/auth/login", { method: "POST", body });
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    return NextResponse.json(data ?? { error: { code: "error", message: "login failed" } }, { status: res.status });
  }
  const { user, ...tokens } = data;
  await setSession(tokens);
  return NextResponse.json({ user });
}
