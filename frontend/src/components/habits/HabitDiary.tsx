"use client";

import { useState } from "react";
import { NotebookPen, ThumbsDown, ThumbsUp, Trash2 } from "lucide-react";
import { useAddHabitLog, useDeleteHabitLog, useHabitLogs } from "@/hooks/useHabits";
import type { HabitKind, HabitStatus } from "@/lib/types";

function statusDot(s: HabitStatus): string {
  return s === "positive" ? "bg-good" : s === "negative" ? "bg-bad" : "bg-ink-600";
}
function fmtWhen(iso: string): string {
  return new Date(iso).toLocaleString("ru-RU", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

// HabitDiary is the free-text status journal for a habit (separate from the
// day-by-day tracker check-ins).
export function HabitDiary({ habitId, kind }: { habitId: string; kind: HabitKind }) {
  const list = useHabitLogs(habitId, true);
  const add = useAddHabitLog(habitId);
  const del = useDeleteHabitLog(habitId);
  const [note, setNote] = useState("");

  const items = list.data?.items ?? [];

  function log(status: HabitStatus) {
    if (!note.trim() && status === "") return;
    add.mutate({ note: note.trim(), status }, { onSuccess: () => setNote("") });
  }

  return (
    <div className="space-y-2">
      <h3 className="flex items-center gap-1.5 font-semibold"><NotebookPen size={16} /> Дневник</h3>

      <div className="space-y-2 rounded-xl border border-ink-800 bg-ink-950/40 p-2.5">
        <input
          className="input !py-1.5"
          placeholder="Как прошло? Заметка о статусе…"
          value={note}
          maxLength={2000}
          onChange={(e) => setNote(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") log(""); }}
        />
        <div className="flex flex-wrap items-center gap-2">
          <button onClick={() => log("positive")} disabled={add.isPending} className="inline-flex items-center gap-1 rounded-lg bg-good/15 px-2.5 py-1.5 text-sm text-good transition hover:bg-good/25">
            <ThumbsUp size={14} /> {kind === "good" ? "Получилось" : "Удержался"}
          </button>
          <button onClick={() => log("negative")} disabled={add.isPending} className="inline-flex items-center gap-1 rounded-lg bg-bad/15 px-2.5 py-1.5 text-sm text-bad transition hover:bg-bad/25">
            <ThumbsDown size={14} /> {kind === "good" ? "Не вышло" : "Сорвался"}
          </button>
          <button onClick={() => log("")} disabled={add.isPending} className="btn-ghost !py-1.5 text-sm">
            Просто заметка
          </button>
        </div>
      </div>

      {items.length > 0 ? (
        <ul className="space-y-1.5">
          {items.map((l) => (
            <li key={l.id} className="flex items-start gap-2 rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2 text-sm">
              <span className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${statusDot(l.status)}`} />
              <div className="min-w-0 flex-1">
                {l.note ? <p className="whitespace-pre-wrap text-ink-200">{l.note}</p> : <p className="text-ink-500">{l.status === "positive" ? "Отметка: успех" : l.status === "negative" ? "Отметка: срыв" : "—"}</p>}
                <span className="text-xs text-ink-500">{fmtWhen(l.createdAt)}</span>
              </div>
              <button onClick={() => del.mutate(l.id)} className="text-ink-600 hover:text-bad" aria-label="Удалить">
                <Trash2 size={13} />
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-xs text-ink-500">Записей пока нет.</p>
      )}
    </div>
  );
}
