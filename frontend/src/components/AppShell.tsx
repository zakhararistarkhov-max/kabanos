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
  ListChecks,
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

// Navigation is grouped so the (now long) list reads at a glance in the sidebar.
const NAV_GROUPS: { label: string; items: { href: string; label: string; Icon: LucideIcon }[] }[] = [
  {
    label: "Обзор",
    items: [
      { href: "/dashboard", label: "Дашборд", Icon: LayoutDashboard },
      { href: "/gtd", label: "GTD", Icon: ListChecks },
      { href: "/diary", label: "Дневник", Icon: BookText },
      { href: "/reminders", label: "Напоминания", Icon: AlarmClock },
    ],
  },
  {
    label: "Здоровье",
    items: [
      { href: "/water", label: "Вода", Icon: Droplet },
      { href: "/nutrition", label: "Калории", Icon: Flame },
      { href: "/weight", label: "Вес", Icon: Scale },
      { href: "/pressure", label: "Давление", Icon: HeartPulse },
      { href: "/meds", label: "Таблетки", Icon: Pill },
    ],
  },
  {
    label: "Активность и еда",
    items: [
      { href: "/workouts", label: "Тренировки", Icon: ClipboardList },
      { href: "/exercises", label: "Упражнения", Icon: Dumbbell },
      { href: "/dishes", label: "Блюда", Icon: Utensils },
    ],
  },
];

const NAV_FLAT = NAV_GROUPS.flatMap((g) => g.items);

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
    <div className="min-h-screen lg:flex">
      {/* ---- desktop sidebar ---- */}
      <aside className="hidden w-60 shrink-0 border-r border-ink-800 bg-ink-950/60 lg:sticky lg:top-0 lg:flex lg:h-screen lg:flex-col">
        <div className="px-4 py-4">
          <Link href="/dashboard" aria-label="Kabanos — на дашборд">
            <KabanosLogo size={30} wordSize="1.05rem" />
          </Link>
        </div>
        <nav className="flex-1 space-y-4 overflow-y-auto px-3 pb-4">
          {NAV_GROUPS.map((group) => (
            <div key={group.label}>
              <p className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-ink-600">{group.label}</p>
              <div className="space-y-0.5">
                {group.items.map(({ href, label, Icon }) => {
                  const active = isActive(href);
                  return (
                    <Link
                      key={href}
                      href={href}
                      className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
                        active ? "bg-ink-800 text-brand" : "text-ink-400 hover:bg-ink-800/60 hover:text-ink-100"
                      }`}
                    >
                      <Icon size={18} strokeWidth={2} className="shrink-0" />
                      {label}
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>
        <div className="flex items-center justify-between gap-2 border-t border-ink-800 px-3 py-2">
          <Link
            href="/settings"
            className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition ${
              isActive("/settings") ? "bg-ink-800 text-brand" : "text-ink-400 hover:bg-ink-800/60 hover:text-ink-100"
            }`}
          >
            <Settings size={18} strokeWidth={2} />
            Настройки
          </Link>
          <div className="flex items-center gap-1">
            <ThemeToggle />
            <button
              onClick={logout}
              title="Выйти"
              className="grid h-9 w-9 place-items-center rounded-lg text-ink-400 transition hover:bg-ink-800 hover:text-ink-100"
            >
              <LogOut size={18} />
            </button>
          </div>
        </div>
      </aside>

      {/* ---- right column ---- */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* mobile top bar */}
        <header className="sticky top-0 z-20 border-b border-ink-800 bg-ink-950/80 backdrop-blur lg:hidden">
          <div className="flex items-center justify-between gap-2 px-4 py-3">
            <Link href="/dashboard" aria-label="Kabanos — на дашборд" className="shrink-0">
              <KabanosLogo size={30} wordSize="1.05rem" />
            </Link>
            <div className="flex items-center gap-1.5">
              <ThemeToggle />
              <button
                onClick={() => setMenuOpen((o) => !o)}
                className="grid h-9 w-9 place-items-center rounded-lg bg-ink-800 text-ink-100"
                aria-label="Меню"
                aria-expanded={menuOpen}
              >
                {menuOpen ? <X size={18} /> : <Menu size={18} />}
              </button>
            </div>
          </div>

          {menuOpen ? (
            <div className="border-t border-ink-800 bg-ink-950/95">
              <nav className="grid grid-cols-2 gap-1 px-3 py-3 sm:grid-cols-3">
                {NAV_FLAT.map(({ href, label, Icon }) => {
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
                <Link
                  href="/settings"
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                    isActive("/settings") ? "bg-ink-800 text-brand" : "text-ink-300 hover:bg-ink-800/60"
                  }`}
                >
                  <Settings size={18} strokeWidth={2} className="shrink-0" />
                  Настройки
                </Link>
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

        <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">{children}</main>
      </div>
    </div>
  );
}
