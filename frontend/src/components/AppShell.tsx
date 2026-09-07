"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { authApi } from "@/lib/api";
import { useSession, useInvalidateSession } from "@/hooks/useSession";
import { KabanosLogo } from "@/components/Logo";

const NAV = [
  { href: "/dashboard", label: "Дашборд", icon: "◆" },
  { href: "/water", label: "Вода", icon: "💧" },
  { href: "/nutrition", label: "Калории", icon: "🍎" },
  { href: "/dishes", label: "Блюда", icon: "🍽️" },
  { href: "/workouts", label: "Тренировки", icon: "🏋️" },
  { href: "/exercises", label: "Упражнения", icon: "💪" },
  { href: "/meds", label: "Таблетки", icon: "💊" },
  { href: "/weight", label: "Вес", icon: "⚖️" },
  { href: "/pressure", label: "Давление", icon: "🩺" },
  { href: "/settings", label: "Настройки", icon: "⚙️" },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { data } = useSession();
  const invalidate = useInvalidateSession();
  const [resent, setResent] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const user = data?.user;

  // Close the mobile menu whenever the route changes.
  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  async function logout() {
    await authApi("/logout", { method: "POST" }).catch(() => undefined);
    invalidate();
    router.push("/login");
  }

  async function resend() {
    await authApi("/resend-verification", { method: "POST" }).catch(() => undefined);
    setResent(true);
  }

  const isActive = (href: string) => pathname === href || pathname.startsWith(href + "/");

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-20 border-b border-ink-800 bg-ink-950/80 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-2 px-4 py-3">
          <Link href="/dashboard" aria-label="Kabanos — на дашборд">
            <KabanosLogo size={32} wordSize="1.05rem" />
          </Link>

          {/* desktop nav */}
          <nav className="hidden items-center gap-1 lg:flex">
            {NAV.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={`rounded-lg px-2.5 py-1.5 text-sm font-medium transition ${
                  isActive(item.href) ? "bg-ink-800 text-ink-100" : "text-ink-300 hover:bg-ink-800/60"
                }`}
              >
                <span className="mr-1">{item.icon}</span>
                <span className="hidden xl:inline">{item.label}</span>
              </Link>
            ))}
          </nav>

          <div className="flex items-center gap-2">
            <button onClick={logout} className="hidden text-sm text-ink-500 hover:text-ink-100 lg:block">
              Выйти
            </button>
            {/* mobile menu toggle */}
            <button
              onClick={() => setMenuOpen((o) => !o)}
              className="grid h-9 w-9 place-items-center rounded-lg bg-ink-800 text-ink-100 lg:hidden"
              aria-label="Меню"
              aria-expanded={menuOpen}
            >
              <span className="text-lg leading-none">{menuOpen ? "✕" : "☰"}</span>
            </button>
          </div>
        </div>

        {/* mobile dropdown menu */}
        {menuOpen ? (
          <div className="border-t border-ink-800 bg-ink-950/95 lg:hidden">
            <nav className="mx-auto grid max-w-5xl grid-cols-2 gap-1 px-3 py-3 sm:grid-cols-3">
              {NAV.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center gap-2 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                    isActive(item.href) ? "bg-ink-800 text-ink-100" : "text-ink-300 hover:bg-ink-800/60"
                  }`}
                >
                  <span className="text-base">{item.icon}</span>
                  {item.label}
                </Link>
              ))}
              <button
                onClick={logout}
                className="flex items-center gap-2 rounded-lg px-3 py-2.5 text-left text-sm font-medium text-ink-400 hover:bg-ink-800/60"
              >
                <span className="text-base">🚪</span>
                Выйти
              </button>
            </nav>
          </div>
        ) : null}
      </header>

      {user && !user.emailVerified ? (
        <div className="border-b border-warn/30 bg-warn/10">
          <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2 px-4 py-2 text-sm text-warn">
            <span>Подтвердите email ({user.email}), чтобы не потерять доступ к аккаунту.</span>
            <button onClick={resend} disabled={resent} className="font-semibold underline disabled:no-underline">
              {resent ? "Письмо отправлено" : "Отправить письмо ещё раз"}
            </button>
          </div>
        </div>
      ) : null}

      <main className="mx-auto max-w-5xl px-4 py-6">{children}</main>
    </div>
  );
}
