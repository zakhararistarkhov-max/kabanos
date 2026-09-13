"use client";

import { useState } from "react";
import { useGtdReview } from "@/hooks/useGtd";
import { ItemRow } from "@/components/gtd/ItemRow";
import type { GtdBucket } from "@/lib/types";

const CHECKLIST = [
  "Разобрать все входящие до нуля",
  "Просмотреть списки следующих действий",
  "Просмотреть список «Ожидание» — напомнить, если нужно",
  "Проверить календарь: прошедшее и ближайшее",
  "Пройтись по проектам — у каждого есть следующее действие",
  "Заглянуть в «Когда‑нибудь/Может быть»",
];

// Step 4 — Reflect: the weekly review. Surfaces what needs attention and a
// checklist to bring the whole system up to date.
export function ReviewTab({ onJump }: { onJump: (tab: string, bucket?: GtdBucket) => void }) {
  const review = useGtdReview();
  const [checked, setChecked] = useState<boolean[]>(() => CHECKLIST.map(() => false));
  const r = review.data;

  if (review.isLoading || !r) return <div className="card h-40 animate-pulse bg-ink-800/40" />;

  const tiles: { label: string; value: number; warn?: boolean; onClick?: () => void }[] = [
    { label: "Входящие", value: r.inboxCount, warn: r.inboxCount > 0, onClick: () => onJump("clarify") },
    { label: "Следующие", value: r.nextCount, onClick: () => onJump("organize") },
    { label: "Ожидание", value: r.waitingCount, onClick: () => onJump("organize") },
    { label: "Календарь", value: r.calendarUpcoming, onClick: () => onJump("organize") },
    { label: "Когда‑нибудь", value: r.somedayCount, onClick: () => onJump("organize") },
    { label: "Сделано за неделю", value: r.completedThisWeek },
  ];

  return (
    <div className="space-y-5">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {tiles.map((t) => (
          <button
            key={t.label}
            onClick={t.onClick}
            disabled={!t.onClick}
            className={`card text-left transition ${t.onClick ? "hover:border-brand/50" : ""} ${t.warn ? "border-warn/40" : ""}`}
          >
            <div className={`text-2xl font-black ${t.warn ? "text-warn" : ""}`}>{t.value}</div>
            <div className="text-xs text-ink-500">{t.label}</div>
          </button>
        ))}
      </div>

      {r.inboxCount > 0 ? (
        <div className="rounded-xl border border-warn/30 bg-warn/10 px-4 py-3 text-sm text-warn">
          Во входящих {r.inboxCount} — начните с их разбора.{" "}
          <button onClick={() => onJump("clarify")} className="font-semibold underline">
            Обработать
          </button>
        </div>
      ) : null}

      {r.stalledProjects.length > 0 ? (
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-ink-400">Проекты без следующего действия</h3>
          {r.stalledProjects.map((p) => (
            <div key={p.id} className="flex items-center justify-between rounded-xl border border-warn/30 bg-ink-950/40 px-3 py-2.5">
              <span className="text-sm">{p.title}</span>
              <button onClick={() => onJump("organize")} className="text-xs text-brand hover:text-brand-soft">
                добавить действие
              </button>
            </div>
          ))}
        </div>
      ) : null}

      {r.overdueCalendar.length > 0 ? (
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-ink-400">Просрочено в календаре</h3>
          {r.overdueCalendar.map((it) => (
            <ItemRow key={it.id} item={it} />
          ))}
        </div>
      ) : null}

      {r.byContext.length > 0 ? (
        <div className="card space-y-2">
          <h3 className="text-sm font-semibold text-ink-400">Следующие действия по контекстам</h3>
          {r.byContext.map((c) => (
            <div key={c.context} className="flex items-center gap-3 text-sm">
              <span className="w-32 shrink-0 truncate text-ink-300">{c.context}</span>
              <div className="h-2 flex-1 overflow-hidden rounded-full bg-ink-800">
                <div className="h-full rounded-full bg-brand" style={{ width: `${Math.min(100, c.count * 12)}%` }} />
              </div>
              <span className="w-6 text-right text-ink-500">{c.count}</span>
            </div>
          ))}
        </div>
      ) : null}

      <div className="card space-y-2">
        <h3 className="font-semibold">Чек‑лист еженедельного обзора</h3>
        {CHECKLIST.map((step, i) => (
          <label key={i} className="flex cursor-pointer items-center gap-2 text-sm text-ink-300">
            <input
              type="checkbox"
              checked={checked[i]}
              onChange={(e) => setChecked((prev) => prev.map((v, j) => (j === i ? e.target.checked : v)))}
              className="h-4 w-4 accent-brand"
            />
            <span className={checked[i] ? "text-ink-600 line-through" : ""}>{step}</span>
          </label>
        ))}
        {checked.every(Boolean) ? <p className="text-sm text-good">Обзор завершён — система в порядке ✅</p> : null}
      </div>
    </div>
  );
}
