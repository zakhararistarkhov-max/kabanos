"use client";

import { useState } from "react";
import { Bell, BellOff, ChevronDown, ChevronRight, NotebookPen, Plus, ThumbsDown, ThumbsUp, Trash2 } from "lucide-react";
import { browserTZ } from "@/lib/api";
import {
  useAddHabitLog,
  useAddHabitReminder,
  useDeleteHabit,
  useDeleteHabitLog,
  useDeleteHabitReminder,
  useHabitLogs,
  useHabitReminders,
  useToggleHabitReminder,
} from "@/hooks/useHabits";
import type { Habit, HabitReminder, HabitStatus } from "@/lib/types";

const WEEKDAYS = ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];

function statusDot(s: HabitStatus): string {
  return s === "positive" ? "bg-good" : s === "negative" ? "bg-bad" : "bg-ink-600";
}
function fmtWhen(iso: string): string {
  return new Date(iso).toLocaleString("ru-RU", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export function HabitCard({ habit }: { habit: Habit }) {
  const [open, setOpen] = useState(false);
  const del = useDeleteHabit();
  const good = habit.kind === "good";

  return (
    <div className={`rounded-2xl border bg-ink-950/40 ${good ? "border-good/30" : "border-bad/30"}`}>
      <div className="flex items-start gap-2 p-3">
        <button onClick={() => setOpen((v) => !v)} className="mt-0.5 text-ink-500 hover:text-ink-100" aria-label="Развернуть">
          {open ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
        </button>
        <button onClick={() => setOpen((v) => !v)} className="min-w-0 flex-1 text-left">
          <div className="flex items-center gap-2">
            <span className="font-semibold">{habit.name}</span>
            {habit.lastStatus ? <span className={`h-2 w-2 rounded-full ${statusDot(habit.lastStatus)}`} title="последняя отметка" /> : null}
          </div>
          {habit.description ? <p className="mt-0.5 line-clamp-1 text-sm text-ink-500">{habit.description}</p> : null}
          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-500">
            {habit.reminderCount > 0 ? <span className="inline-flex items-center gap-1"><Bell size={12} /> {habit.reminderCount}</span> : null}
            {habit.logCount > 0 ? <span className="inline-flex items-center gap-1"><NotebookPen size={12} /> {habit.logCount}</span> : null}
            {habit.reminderCount === 0 && habit.logCount === 0 ? <span>нажмите, чтобы настроить</span> : null}
          </div>
        </button>
        <button
          onClick={() => {
            if (confirm(`Удалить привычку «${habit.name}»?`)) del.mutate(habit.id);
          }}
          className="text-ink-600 transition hover:text-bad"
          aria-label="Удалить привычку"
        >
          <Trash2 size={15} />
        </button>
      </div>

      {open ? (
        <div className="space-y-4 border-t border-ink-800 p-3">
          <Reminders habitId={habit.id} />
          <Diary habitId={habit.id} kind={habit.kind} />
        </div>
      ) : null}
    </div>
  );
}

function Reminders({ habitId }: { habitId: string }) {
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
      <h4 className="flex items-center gap-1.5 text-sm font-semibold text-ink-300"><Bell size={14} /> Напоминания</h4>

      {items.map((r) => (
        <ReminderRow key={r.id} r={r} onToggle={(en) => toggle.mutate({ id: r.id, enabled: en })} onDelete={() => del.mutate(r.id)} />
      ))}

      <div className="rounded-xl border border-ink-800 bg-ink-950/40 p-2.5 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <input type="time" className="input !w-auto !py-1.5" value={time} onChange={(e) => setTime(e.target.value)} />
          <input
            className="input !py-1.5 min-w-40 flex-1"
            placeholder="Текст напоминания (напр. «Выпей воды»)"
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

function Diary({ habitId, kind }: { habitId: string; kind: "good" | "bad" }) {
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
      <h4 className="flex items-center gap-1.5 text-sm font-semibold text-ink-300"><NotebookPen size={14} /> Дневник</h4>

      <div className="rounded-xl border border-ink-800 bg-ink-950/40 p-2.5 space-y-2">
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
            <ThumbsUp size={14} /> {kind === "good" ? "Сделал" : "Удержался"}
          </button>
          <button onClick={() => log("negative")} disabled={add.isPending} className="inline-flex items-center gap-1 rounded-lg bg-bad/15 px-2.5 py-1.5 text-sm text-bad transition hover:bg-bad/25">
            <ThumbsDown size={14} /> {kind === "good" ? "Пропустил" : "Сорвался"}
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
        <p className="text-xs text-ink-500">Записей пока нет — отметьте первый статус выше.</p>
      )}
    </div>
  );
}
