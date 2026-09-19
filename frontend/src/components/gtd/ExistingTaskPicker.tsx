"use client";

import { useState } from "react";
import { Search, X } from "lucide-react";
import { useGtdItems } from "@/hooks/useGtd";
import { BUCKET_LABELS } from "@/lib/gtd";

// ExistingTaskPicker lists open tasks not already on the board, so a task
// captured in «Сбор» can be placed on the map as the same entity (no new task
// created — it links the existing gtd_item).
export function ExistingTaskPicker({
  existingItemIds,
  onPick,
  onClose,
  busy,
}: {
  existingItemIds: Set<string>;
  onPick: (itemId: string) => void;
  onClose: () => void;
  busy: boolean;
}) {
  const items = useGtdItems({ done: false });
  const [q, setQ] = useState("");
  const all = items.data?.items ?? [];
  const list = all
    .filter((it) => !existingItemIds.has(it.id))
    .filter((it) => it.title.toLowerCase().includes(q.trim().toLowerCase()));

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4" onClick={onClose}>
      <div className="card flex max-h-[80vh] w-full max-w-md flex-col gap-3" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-semibold">Добавить задачу на карту</h2>
            <p className="text-sm text-ink-500">Уже собранные задачи — та же сущность, без дублей.</p>
          </div>
          <button onClick={onClose} className="text-ink-500 hover:text-ink-100" aria-label="Закрыть">
            <X size={18} />
          </button>
        </div>

        <div className="relative">
          <Search size={15} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-500" />
          <input className="input !pl-9" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Поиск задачи…" autoFocus />
        </div>

        <div className="min-h-0 flex-1 space-y-1.5 overflow-y-auto">
          {items.isLoading ? (
            <div className="h-16 animate-pulse rounded-lg bg-ink-800/40" />
          ) : list.length === 0 ? (
            <p className="py-6 text-center text-sm text-ink-500">
              {all.length === 0 ? "Нет открытых задач. Соберите их на шаге «Сбор»." : "Все задачи уже на карте или не найдены."}
            </p>
          ) : (
            list.map((it) => (
              <button
                key={it.id}
                onClick={() => onPick(it.id)}
                disabled={busy}
                className="flex w-full items-center gap-2 rounded-lg border border-ink-800 bg-ink-950/40 px-3 py-2 text-left text-sm transition hover:border-brand/50 disabled:opacity-50"
              >
                <span className="min-w-0 flex-1 truncate">{it.title}</span>
                <span className="shrink-0 chip">{BUCKET_LABELS[it.bucket] ?? it.bucket}</span>
                {it.dueOn ? <span className="shrink-0 text-xs text-ink-500">{it.dueOn}</span> : null}
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
