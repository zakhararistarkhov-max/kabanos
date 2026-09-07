"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  AlarmClock,
  BookText,
  ClipboardList,
  Droplet,
  Dumbbell,
  Flame,
  HeartPulse,
  LayoutDashboard,
  LogOut,
  Menu,
  Pill,
  Scale,
  Settings,
  Utensils,
  X,
  type LucideIcon,
} from "lucide-react";
import { authApi } from "@/lib/api";
import { useSession, useInvalidateSession } from "@/hooks/useSession";
import { KabanosLogo } from "@/components/Logo";
import { ThemeToggle } from "@/components/ThemeToggle";

const NAV: { href: string; label: string; Icon: LucideIcon }[] = [
  { href: "/dashboard", label: "Дашборд", Icon: LayoutDashboard },
  { href: "/water", label: "Вода", Icon: Droplet },
  { href: "/nutrition", label: "Калории", Icon: Flame },
  { href: "/dishes", label: "Блюда", Icon: Utensils },
  { href: "/workouts", label: "Тренировки", Icon: ClipboardList },
  { href: "/exercises", label: "Упражнения", Icon: Dumbbell },
  { href: "/meds", label: "Таблетки", Icon: Pill },
  { href: "/weight", label: "Вес", Icon: Scale },
  { href: "/pressure", label: "Давление", Icon: HeartPulse },
  { href: "/diary", label: "Дневник", Icon: BookText },
  { href: "/reminders", label: "Напоминания", Icon: AlarmClock },
  { href: "/settings", label: "Настройки", Icon: Settings },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { data } = useSession();
  const invalidate = useInvalidateSession();
  const [resent, setResent] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const user = data?.user;

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
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-2 px-4 py-3">
          <Link href="/dashboard" aria-label="Kabanos — на дашборд" className="shrink-0">
            <KabanosLogo size={32} wordSize="1.05rem" />
          </Link>

          {/* desktop nav */}
          <nav className="hidden items-center gap-0.5 lg:flex">
            {NAV.map(({ href, label, Icon }) => {
              const active = isActive(href);
              return (
                <Link
                  key={href}
                  href={href}
                  title={label}
                  className={`flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm font-medium transition ${
                    active ? "bg-ink-800 text-brand" : "text-ink-400 hover:bg-ink-800/60 hover:text-ink-100"
                  }`}
                >
                  <Icon size={17} strokeWidth={2} className="shrink-0" />
                  <span className="hidden 2xl:inline">{label}</span>
                </Link>
              );
            })}
          </nav>

          <div className="flex items-center gap-1.5">
            <ThemeToggle />
            <button
              onClick={logout}
              title="Выйти"
              className="hidden h-9 w-9 place-items-center rounded-lg text-ink-400 transition hover:bg-ink-800 hover:text-ink-100 lg:grid"
            >
              <LogOut size={18} />
            </button>
            {/* mobile menu toggle */}
            <button
              onClick={() => setMenuOpen((o) => !o)}
              className="grid h-9 w-9 place-items-center rounded-lg bg-ink-800 text-ink-100 lg:hidden"
              aria-label="Меню"
              aria-expanded={menuOpen}
            >
              {menuOpen ? <X size={18} /> : <Menu size={18} />}
            </button>
          </div>
        </div>

        {/* mobile dropdown menu */}
        {menuOpen ? (
          <div className="border-t border-ink-800 bg-ink-950/95 lg:hidden">
            <nav className="mx-auto grid max-w-6xl grid-cols-2 gap-1 px-3 py-3 sm:grid-cols-3">
              {NAV.map(({ href, label, Icon }) => {
                const active = isActive(href);
                return (
                  <Link
                    key={href}
                    href={href}
                    className={`flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                      active ? "bg-ink-800 text-brand" : "text-ink-300 hover:bg-ink-800/60"
                    }`}
                  >
                    <Icon size={18} strokeWidth={2} className="shrink-0" />
                    {label}
                  </Link>
                );
              })}
              <button
                onClick={logout}
                className="flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-sm font-medium text-ink-400 hover:bg-ink-800/60"
              >
                <LogOut size={18} strokeWidth={2} className="shrink-0" />
                Выйти
              </button>
            </nav>
          </div>
        ) : null}
      </header>

      {user && !user.emailVerified ? (
        <div className="border-b border-warn/30 bg-warn/10">
          <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2 px-4 py-2 text-sm text-warn">
            <span>Подтвердите email ({user.email}), чтобы не потерять доступ к аккаунту.</span>
            <button onClick={resend} disabled={resent} className="font-semibold underline disabled:no-underline">
              {resent ? "Письмо отправлено" : "Отправить письмо ещё раз"}
            </button>
          </div>
        </div>
      ) : null}

      <main className="mx-auto max-w-6xl px-4 py-6">{children}</main>
    </div>
  );
}
