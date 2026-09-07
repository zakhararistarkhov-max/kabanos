"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "kabanos:dashboard:layout:v1";

// The default set and order of dashboard widgets for a new user.
export const DEFAULT_LAYOUT = ["water", "nutrition", "weight", "pressure", "macros", "training", "meds"];

// useDashboardLayout persists the user's chosen widgets (which ones and in what
// order) in localStorage — a per-browser preference, no backend round-trip.
export function useDashboardLayout(allIds: string[]) {
  const [order, setOrder] = useState<string[]>(DEFAULT_LAYOUT);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as unknown;
        if (Array.isArray(parsed)) {
          const cleaned = parsed.filter((id): id is string => typeof id === "string" && allIds.includes(id));
          if (cleaned.length > 0) setOrder(cleaned);
        }
      }
    } catch {
      /* ignore malformed/absent storage */
    }
    setMounted(true);
    // allIds is a stable module constant; run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function persist(next: string[]) {
    setOrder(next);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    } catch {
      /* storage may be unavailable (private mode) — keep in-memory */
    }
  }

  const hidden = allIds.filter((id) => !order.includes(id));

  return {
    mounted,
    enabled: order,
    hidden,
    isEnabled: (id: string) => order.includes(id),
    toggle: (id: string) => persist(order.includes(id) ? order.filter((x) => x !== id) : [...order, id]),
    move: (id: string, dir: -1 | 1) => {
      const i = order.indexOf(id);
      const j = i + dir;
      if (i < 0 || j < 0 || j >= order.length) return;
      const next = [...order];
      [next[i], next[j]] = [next[j], next[i]];
      persist(next);
    },
    reset: () => persist(DEFAULT_LAYOUT),
  };
}
