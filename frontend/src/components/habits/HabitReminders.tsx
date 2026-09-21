"use client";

import { useState } from "react";
import { Bell, BellOff, Plus, Trash2 } from "lucide-react";
import { browserTZ } from "@/lib/api";
import {
  useAddHabitReminder,
  useDeleteHabitReminder,
  useHabitReminders,
  useToggleHabitReminder,
} from "@/hooks/useHabits";
import type { HabitReminder } from "@/lib/types";

const WEEKDAYS = ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];

export function HabitReminders({ habitId }: { habitId: string }) {
  const list = useHabitReminders(habitId, true);
  const add = useAddHabitReminder(habitId);
  const toggle = useToggleHabitReminder(habitId);
  const del = useDeleteHabitReminder(habitId);

  const [text, setText] = useState("");
  const [time, setTime] = useState("09:00");
  const [days, setDays] = useState<number[]>([]);

  const items = list.data?.items ?? [];

  function submit() {
    add.mutate(
      { text: text.trim(), times: [time], days, timezone: browserTZ() },
      { onSuccess: () => { setText(""); setDays([]); } },
    );
  }

  return (
    <div className="space-y-2">
      <h3 className="flex items-center gap-1.5 font-semibold"><Bell size={16} /> Напоминания</h3>

      {items.map((r) => (
        <ReminderRow key={r.id} r={r} onToggle={(en) => toggle.mutate({ id: r.id, enabled: en })} onDelete={() => del.mutate(r.id)} />
      ))}

      <div className="space-y-2 rounded-xl border border-ink-800 bg-ink-950/40 p-2.5">
        <div className="flex flex-wrap items-center gap-2">
          <input type="time" className="input !w-auto !py-1.5" value={time} onChange={(e) => setTime(e.target.value)} />
          <input
            className="input !py-1.5 min-w-40 flex-1"
            placeholder="Текст напоминания (напр. «Не забудь!»)"
            value={text}
            maxLength={300}
            onChange={(e) => setText(e.target.value)}
          />
        </div>
        <div className="flex flex-wrap items-center gap-1">
          {WEEKDAYS.map((d, i) => {
            const on = days.includes(i);
            return (
              <button
                key={i}
                onClick={() => setDays((prev) => (on ? prev.filter((x) => x !== i) : [...prev, i]))}
                className={`rounded-md px-2 py-1 text-xs transition ${on ? "bg-brand text-ink-950" : "bg-ink-800 text-ink-400 hover:bg-ink-700"}`}
              >
                {d}
              </button>
            );
          })}
          <span className="ml-1 text-xs text-ink-500">{days.length === 0 ? "каждый день" : ""}</span>
          <button onClick={submit} disabled={add.isPending} className="btn-primary ml-auto !py-1.5">
            <Plus size={14} /> Добавить
          </button>
        </div>
      </div>
    </div>
  );
}

function ReminderRow({ r, onToggle, onDelete }: { r: HabitReminder; onToggle: (enabled: boolean) => void; onDelete: () => void }) {
  return (
    <div className="flex items-center gap-2 rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2 text-sm">
      <span className="font-mono tabular-nums text-ink-200">{r.times.join(", ")}</span>
      <span className="min-w-0 flex-1 truncate text-ink-300">
        {r.text || "напоминание"}
        {r.days.length > 0 ? <span className="text-ink-500"> · {r.days.map((d) => WEEKDAYS[d]).join(",")}</span> : null}
      </span>
      <button onClick={() => onToggle(!r.enabled)} className={r.enabled ? "text-brand" : "text-ink-600"} title={r.enabled ? "Включено" : "Выключено"}>
        {r.enabled ? <Bell size={15} /> : <BellOff size={15} />}
      </button>
      <button onClick={onDelete} className="text-ink-600 hover:text-bad" aria-label="Удалить">
        <Trash2 size={14} />
      </button>
    </div>
  );
}
