import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { COOKIE_REFRESH } from "@/lib/config";

// Entry point: send the visitor to their dashboard or to the login screen.
export default async function Home() {
  const store = await cookies();
  redirect(store.get(COOKIE_REFRESH)?.value ? "/dashboard" : "/login");
}
