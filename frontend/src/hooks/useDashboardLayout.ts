"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "kabanos:dashboard:layout:v2";
const LEGACY_KEY = "kabanos:dashboard:layout:v1";

// The default set and order of dashboard widgets for a new user.
export const DEFAULT_LAYOUT = ["water", "nutrition", "weight", "pressure", "macros", "training", "meds"];

interface Stored {
  order: string[]; // enabled widgets, in display order
  known: string[]; // every widget the user has already seen (shown or hidden)
}

// useDashboardLayout persists the user's chosen widgets in localStorage. It also
// tracks which widgets the user has already *seen* so that a widget added in a
// later release appears automatically once — without un-hiding widgets the user
// deliberately removed.
export function useDashboardLayout(allIds: string[]) {
  const [order, setOrder] = useState<string[]>(DEFAULT_LAYOUT);
  const [known, setKnown] = useState<string[]>(DEFAULT_LAYOUT);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    let ord = DEFAULT_LAYOUT;
    let kn = DEFAULT_LAYOUT;
    try {
      const rawV2 = localStorage.getItem(STORAGE_KEY);
      if (rawV2) {
        const p = JSON.parse(rawV2) as Stored;
        if (p && Array.isArray(p.order) && Array.isArray(p.known)) {
          ord = p.order.filter((id) => allIds.includes(id));
          kn = p.known.filter((id) => allIds.includes(id));
        }
      } else {
        const rawV1 = localStorage.getItem(LEGACY_KEY);
        if (rawV1) {
          const arr = JSON.parse(rawV1);
          if (Array.isArray(arr)) {
            ord = arr.filter((id): id is string => typeof id === "string" && allIds.includes(id));
            kn = [...ord]; // only what they had counts as "seen"; newer widgets append below
          }
        }
      }
    } catch {
      /* ignore malformed/absent storage */
    }

    // Append widgets the user has never seen (new since their last save).
    const unseen = allIds.filter((id) => !kn.includes(id));
    if (unseen.length > 0) {
      ord = [...ord, ...unseen];
      kn = [...kn, ...unseen];
    }

    setOrder(ord);
    setKnown(kn);
    setMounted(true);
    persist(ord, kn);
    // allIds is a stable module constant; run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function persist(ord: string[], kn: string[]) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ order: ord, known: kn }));
    } catch {
      /* storage may be unavailable (private mode) — keep in-memory */
    }
  }

  function update(ord: string[], kn: string[]) {
    setOrder(ord);
    setKnown(kn);
    persist(ord, kn);
  }

  const hidden = allIds.filter((id) => !order.includes(id));
  const withKnown = (id: string) => (known.includes(id) ? known : [...known, id]);

  return {
    mounted,
    enabled: order,
    hidden,
    isEnabled: (id: string) => order.includes(id),
    toggle: (id: string) =>
      order.includes(id) ? update(order.filter((x) => x !== id), known) : update([...order, id], withKnown(id)),
    move: (id: string, dir: -1 | 1) => {
      const i = order.indexOf(id);
      const j = i + dir;
      if (i < 0 || j < 0 || j >= order.length) return;
      const next = [...order];
      [next[i], next[j]] = [next[j], next[i]];
      update(next, known);
    },
    reset: () => update(DEFAULT_LAYOUT, allIds),
  };
}
