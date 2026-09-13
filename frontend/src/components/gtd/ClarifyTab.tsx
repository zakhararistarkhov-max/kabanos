"use client";

import { useState } from "react";
import { Check } from "lucide-react";
import { itemToInput, useDeleteItem, useGtdItems, useToggleDone, useUpdateItem } from "@/hooks/useGtd";
import { ItemEditor } from "@/components/gtd/ItemEditor";
import type { GtdBucket, GtdItem } from "@/lib/types";

// Step 2 — Clarify: decide what each inbox item is and where it goes. Simple
// decisions are one click; anything needing detail (a context, a time, who we're
// waiting on) opens the full editor prefilled with the chosen bucket.
export function ClarifyTab() {
  const inbox = useGtdItems({ bucket: "inbox", done: false });
  const update = useUpdateItem();
  const toggle = useToggleDone();
  const del = useDeleteItem();
  const [editing, setEditing] = useState<GtdItem | null>(null);

  const items = inbox.data?.items ?? [];
  const current = items[0];
  const btn = "rounded-lg bg-ink-800 px-3 py-2 text-sm text-ink-200 transition hover:bg-brand/15";

  function quickMove(item: GtdItem, bucket: GtdBucket) {
    update.mutate({ id: item.id, input: { ...itemToInput(item), bucket } });
  }

  if (editing) {
    return (
      <div className="space-y-3">
        <p className="text-sm text-ink-500">Уточните детали и сохраните — пункт уйдёт из входящих.</p>
        <ItemEditor initial={editing} onDone={() => setEditing(null)} onCancel={() => setEditing(null)} />
      </div>
    );
  }

  if (inbox.isLoading) return <div className="card h-40 animate-pulse bg-ink-800/40" />;

  if (!current) {
    return (
      <div className="card text-center text-ink-500">
        <div className="mb-1 text-2xl">✅</div>
        Входящие разобраны. Загляните в «Организацию» или «Выполнение».
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="text-sm text-ink-500">Осталось разобрать: {items.length}</div>

      <div className="card space-y-4">
        <div>
          <div className="text-lg font-semibold">{current.title}</div>
          {current.notes ? <p className="mt-1 text-sm text-ink-400">{current.notes}</p> : null}
        </div>

        <div>
          <div className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-500">Требует действия?</div>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => toggle.mutate({ id: current.id, done: true })}
              className="flex items-center gap-1.5 rounded-lg bg-good/15 px-3 py-2 text-sm font-medium text-good transition hover:bg-good/25"
              title="Правило 2 минут: если быстро — сделайте сейчас"
            >
              <Check size={15} /> Сделал (≤2 мин)
            </button>
            <button onClick={() => setEditing({ ...current, bucket: "next" })} className={btn}>
              ⚡ Следующее действие
            </button>
            <button onClick={() => setEditing({ ...current, bucket: "calendar" })} className={btn}>
              📅 Запланировать
            </button>
            <button onClick={() => setEditing({ ...current, bucket: "waiting" })} className={btn}>
              ⏳ Делегировать
            </button>
          </div>
        </div>

        <div>
          <div className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-500">Не требует действия</div>
          <div className="flex flex-wrap gap-2">
            <button onClick={() => quickMove(current, "reference")} className={btn}>
              📚 В справку
            </button>
            <button onClick={() => quickMove(current, "someday")} className={btn}>
              💭 Когда‑нибудь
            </button>
            <button
              onClick={() => del.mutate(current.id)}
              className="rounded-lg bg-ink-800 px-3 py-2 text-sm text-ink-400 transition hover:bg-bad/20 hover:text-bad"
            >
              🗑 Удалить
            </button>
          </div>
        </div>

        <button onClick={() => setEditing(current)} className="text-sm text-brand hover:text-brand-soft">
          Настроить подробно…
        </button>
      </div>

      {/* peek at the queue */}
      {items.length > 1 ? (
        <div>
          <div className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-500">Дальше в очереди</div>
          <ul className="space-y-1 text-sm text-ink-400">
            {items.slice(1, 6).map((it) => (
              <li key={it.id} className="truncate rounded-lg bg-ink-800/40 px-3 py-1.5">
                {it.title}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
