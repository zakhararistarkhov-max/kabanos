"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "kabanos:dashboard:layout:v3";
const LEGACY_V2 = "kabanos:dashboard:layout:v2";
const LEGACY_V1 = "kabanos:dashboard:layout:v1";

// The default set and order of dashboard widgets for a new user.
export const DEFAULT_LAYOUT = ["water", "nutrition", "fasting", "weight", "pressure", "diary", "macros", "gtd", "training", "meds"];

export type Span = 1 | 2 | 3;

interface Stored {
  order: string[]; // enabled widgets, in display order
  known: string[]; // every widget the user has already seen (shown or hidden)
  spans: Record<string, Span>; // per-widget width override (else the widget's default)
}

// useDashboardLayout persists the user's chosen widgets, their order and their
// per-widget width in localStorage. It also tracks which widgets the user has
// already *seen* so a widget added in a later release appears automatically once.
export function useDashboardLayout(allIds: string[]) {
  const [order, setOrder] = useState<string[]>(DEFAULT_LAYOUT);
  const [known, setKnown] = useState<string[]>(DEFAULT_LAYOUT);
  const [spans, setSpans] = useState<Record<string, Span>>({});
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    let ord = DEFAULT_LAYOUT;
    let kn = DEFAULT_LAYOUT;
    let sp: Record<string, Span> = {};
    try {
      const rawV3 = localStorage.getItem(STORAGE_KEY);
      const rawV2 = localStorage.getItem(LEGACY_V2);
      const rawV1 = localStorage.getItem(LEGACY_V1);
      if (rawV3) {
        const p = JSON.parse(rawV3) as Stored;
        if (p && Array.isArray(p.order) && Array.isArray(p.known)) {
          ord = p.order.filter((id) => allIds.includes(id));
          kn = p.known.filter((id) => allIds.includes(id));
          sp = sanitizeSpans(p.spans, allIds);
        }
      } else if (rawV2) {
        const p = JSON.parse(rawV2) as { order: string[]; known: string[] };
        if (p && Array.isArray(p.order) && Array.isArray(p.known)) {
          ord = p.order.filter((id) => allIds.includes(id));
          kn = p.known.filter((id) => allIds.includes(id));
        }
      } else if (rawV1) {
        const arr = JSON.parse(rawV1);
        if (Array.isArray(arr)) {
          ord = arr.filter((id): id is string => typeof id === "string" && allIds.includes(id));
          kn = [...ord];
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
    setSpans(sp);
    setMounted(true);
    persist(ord, kn, sp);
    // allIds is a stable module constant; run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function persist(ord: string[], kn: string[], sp: Record<string, Span>) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ order: ord, known: kn, spans: sp }));
    } catch {
      /* storage may be unavailable (private mode) — keep in-memory */
    }
  }

  function commit(ord: string[], kn: string[], sp: Record<string, Span>) {
    setOrder(ord);
    setKnown(kn);
    setSpans(sp);
    persist(ord, kn, sp);
  }

  const hidden = allIds.filter((id) => !order.includes(id));
  const withKnown = (id: string) => (known.includes(id) ? known : [...known, id]);

  return {
    mounted,
    enabled: order,
    hidden,
    spans,
    isEnabled: (id: string) => order.includes(id),
    toggle: (id: string) =>
      order.includes(id)
        ? commit(order.filter((x) => x !== id), known, spans)
        : commit([...order, id], withKnown(id), spans),
    // reorder replaces the whole order (used by drag-and-drop).
    reorder: (nextOrder: string[]) => {
      const filtered = nextOrder.filter((id) => allIds.includes(id));
      // guard: keep the same set, just reordered
      if (filtered.length !== order.length) return;
      commit(filtered, known, spans);
    },
    setSpan: (id: string, span: Span) => commit(order, known, { ...spans, [id]: span }),
    reset: () => commit(DEFAULT_LAYOUT, allIds, {}),
  };
}

function sanitizeSpans(sp: unknown, allIds: string[]): Record<string, Span> {
  const out: Record<string, Span> = {};
  if (sp && typeof sp === "object") {
    for (const [k, v] of Object.entries(sp as Record<string, unknown>)) {
      if (allIds.includes(k) && (v === 1 || v === 2 || v === 3)) out[k] = v;
    }
  }
  return out;
}
