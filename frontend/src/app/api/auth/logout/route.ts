import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { COOKIE_REFRESH } from "@/lib/config";
import { clearSession, rawBackend } from "@/lib/server/session";

export async function POST() {
  const store = await cookies();
  const rt = store.get(COOKIE_REFRESH)?.value;
  if (rt) {
    // Best-effort server-side revocation; ignore failures.
    await rawBackend("/api/v1/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refreshToken: rt }),
    }).catch(() => undefined);
  }
  await clearSession();
  return NextResponse.json({ ok: true });
}
