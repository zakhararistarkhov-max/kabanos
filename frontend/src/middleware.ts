import { NextRequest, NextResponse } from "next/server";
import { COOKIE_REFRESH } from "@/lib/config";

// Route protection at the edge: unauthenticated users hitting an app route are
// bounced to /login; already-authenticated users are kept out of the auth
// pages. Presence of the refresh cookie is a cheap first gate — the API still
// enforces real authorization on every request.
const PROTECTED = ["/dashboard", "/water", "/weight", "/settings", "/nutrition", "/dishes"];
const AUTH_PAGES = ["/login", "/register"];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  const hasSession = Boolean(req.cookies.get(COOKIE_REFRESH)?.value);

  if (PROTECTED.some((p) => pathname === p || pathname.startsWith(p + "/")) && !hasSession) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }

  if (AUTH_PAGES.includes(pathname) && hasSession) {
    const url = req.nextUrl.clone();
    url.pathname = "/dashboard";
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/water/:path*",
    "/weight/:path*",
    "/settings/:path*",
    "/nutrition/:path*",
    "/dishes/:path*",
    "/login",
    "/register",
  ],
};
