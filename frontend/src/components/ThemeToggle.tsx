"use client";

import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";

type Theme = "light" | "dark";

// ThemeToggle flips between the light and dark theme, persisting the choice.
// The initial theme is applied before paint by the inline script in the root
// layout; this component only reflects and updates it.
export function ThemeToggle({ className }: { className?: string }) {
  const [theme, setTheme] = useState<Theme>("dark");

  useEffect(() => {
    const t = document.documentElement.dataset.theme;
    setTheme(t === "light" ? "light" : "dark");
  }, []);

  function toggle() {
    const next: Theme = theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    try {
      localStorage.setItem("kabanos:theme", next);
    } catch {
      /* storage may be unavailable */
    }
    setTheme(next);
  }

  return (
    <button
      onClick={toggle}
      aria-label={theme === "dark" ? "Включить светлую тему" : "Включить тёмную тему"}
      title="Сменить тему"
      className={`grid h-9 w-9 place-items-center rounded-lg text-ink-400 transition hover:bg-ink-800 hover:text-ink-100 ${className ?? ""}`}
    >
      {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
    </button>
  );
}
