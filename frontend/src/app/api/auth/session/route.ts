import { NextResponse } from "next/server";
import { backendFetch } from "@/lib/server/session";

// Returns the current user (refreshing the access token if needed), or 401.
export async function GET() {
  const res = await backendFetch("/api/v1/me");
  if (!res.ok) {
    return NextResponse.json({ user: null }, { status: 401 });
  }
  const user = await res.json();
  return NextResponse.json({ user });
}
