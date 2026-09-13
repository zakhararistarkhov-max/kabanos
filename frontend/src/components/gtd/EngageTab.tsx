"use client";

import { useState } from "react";
import { useGtdContexts, useGtdItems } from "@/hooks/useGtd";
import { ItemRow } from "@/components/gtd/ItemRow";
import { ENERGY_LABELS } from "@/lib/gtd";
import type { GtdEnergy } from "@/lib/types";

const TIME_OPTIONS = [
  { label: "Любое время", value: 0 },
  { label: "≤ 15 мин", value: 15 },
  { label: "≤ 30 мин", value: 30 },
  { label: "≤ 1 час", value: 60 },
];

// Step 5 — Engage: pick what to do now by context, available time and energy.
// The list is already priority-sorted by the API.
export function EngageTab() {
  const contexts = useGtdContexts();
  const [context, setContext] = useState("");
  const [maxTime, setMaxTime] = useState(0);
  const [energy, setEnergy] = useState<GtdEnergy>("");

  const q = useGtdItems({
    bucket: "next",
    done: false,
    context: context || undefined,
    maxTime: maxTime || undefined,
    energy: energy || undefined,
  });
  const items = q.data?.items ?? [];

  // Calendar items scheduled for today, shown as a heads-up.
  const cal = useGtdItems({ bucket: "calendar", done: false });
  const todayStr = new Date().toDateString();
  const todayCal = (cal.data?.items ?? []).filter((it) => it.scheduledAt && new Date(it.scheduledAt).toDateString() === todayStr);

  return (
    <div className="space-y-5">
      <div className="card space-y-3">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div>
            <label className="label">Контекст</label>
            <select className="input" value={context} onChange={(e) => setContext(e.target.value)}>
              <option value="">Все</option>
              {(contexts.data?.items ?? []).map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Есть времени</label>
            <select className="input" value={maxTime} onChange={(e) => setMaxTime(Number(e.target.value))}>
              {TIME_OPTIONS.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Энергия</label>
            <select className="input" value={energy} onChange={(e) => setEnergy(e.target.value as GtdEnergy)}>
              {(["", "low", "medium", "high"] as GtdEnergy[]).map((en) => (
                <option key={en} value={en}>
                  {en === "" ? "Любая" : ENERGY_LABELS[en]}
                </option>
              ))}
            </select>
          </div>
        </div>
        <p className="text-xs text-ink-500">Список отсортирован по приоритету. Отметьте галочкой — и он исчезнет из списка.</p>
      </div>

      {todayCal.length > 0 ? (
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-ink-400">📅 Сегодня в календаре</h3>
          {todayCal.map((it) => (
            <ItemRow key={it.id} item={it} />
          ))}
        </div>
      ) : null}

      <div className="space-y-2">
        <h3 className="text-sm font-semibold text-ink-400">⚡ Что можно сделать сейчас</h3>
        {q.isLoading ? (
          <div className="card h-20 animate-pulse bg-ink-800/40" />
        ) : items.length > 0 ? (
          items.map((it) => <ItemRow key={it.id} item={it} />)
        ) : (
          <div className="card text-center text-sm text-ink-500">Под эти условия ничего нет. Смягчите фильтры или добавьте действия.</div>
        )}
      </div>
    </div>
  );
}
