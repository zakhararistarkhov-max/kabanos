"use client";

import Link from "next/link";
import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { browserTZ } from "@/lib/api";
import {
  reminderToInput,
  useCreateReminder,
  useDeleteReminder,
  useReminders,
  useUpdateReminder,
  type ReminderInput,
} from "@/hooks/useReminders";
import type { Reminder } from "@/lib/types";

const WEEKDAYS = ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];
const CONDITION_LABELS: Record<string, string> = {
  "": "",
  water_below_goal: "если воды меньше нормы",
  meds_due: "если есть невыпитые таблетки",
};
const TARGETS = [
  { url: "/water", label: "Вода" },
  { url: "/nutrition", label: "Калории" },
  { url: "/meds", label: "Таблетки" },
  { url: "/weight", label: "Вес" },
  { url: "/pressure", label: "Давление" },
  { url: "/diary", label: "Дневник" },
  { url: "/dashboard", label: "Дашборд" },
];

const PRESETS: { label: string; input: Omit<ReminderInput, "timezone"> }[] = [
  {
    label: "💧 Вода каждые 2 часа",
    input: { title: "Пора попить воды 💧", body: "Сделайте пару глотков", url: "/water", mode: "interval", intervalMinutes: 120, windowStart: "08:00", windowEnd: "22:00", condition: "water_below_goal" },
  },
  {
    label: "💊 Таблетки каждый час (если не выпил)",
    input: { title: "Не забудьте про таблетки 💊", body: "Отметьте приём в разделе «Таблетки»", url: "/meds", mode: "interval", intervalMinutes: 60, windowStart: "09:00", windowEnd: "21:00", condition: "meds_due" },
  },
  {
    label: "🍎 Записать завтрак / обед / ужин",
    input: { title: "Запишите калории 🍎", body: "Не забудьте внести приём пищи", url: "/nutrition", mode: "times", times: ["09:00", "14:00", "20:00"] },
  },
];

function fmtInterval(min: number): string {
  if (min < 60) return `${min} мин`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return m ? `${h} ч ${m} мин` : `${h} ч`;
}

function summary(r: Reminder): string {
  let s = r.mode === "interval" ? `каждые ${fmtInterval(r.intervalMinutes ?? 0)}, ${r.windowStart}–${r.windowEnd}` : `в ${r.times.join(", ")}`;
  if (r.condition) s += ` · ${CONDITION_LABELS[r.condition]}`;
  if (r.days.length > 0 && r.days.length < 7) s += ` · ${r.days.map((d) => WEEKDAYS[d]).join(", ")}`;
  return s;
}

export default function RemindersPage() {
  const list = useReminders();
  const create = useCreateReminder();
  const del = useDeleteReminder();
  const [showForm, setShowForm] = useState(false);

  const items = list.data?.items ?? [];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Напоминания</h1>
          <p className="text-sm text-ink-500">Push‑уведомления по расписанию. Включите уведомления в <Link href="/settings" className="text-brand">настройках</Link>.</p>
        </div>
        <button onClick={() => setShowForm((s) => !s)} className={showForm ? "btn-ghost" : "btn-primary"}>
          {showForm ? "Закрыть" : (<><Plus size={16} /> Своё напоминание</>)}
        </button>
      </div>

      {/* presets */}
      <div className="card">
        <h2 className="mb-3 font-semibold">Быстрые шаблоны</h2>
        <div className="flex flex-wrap gap-2">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              onClick={() => create.mutate({ ...p.input, timezone: browserTZ() })}
              disabled={create.isPending}
              className="rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2 text-sm transition hover:border-brand/50"
            >
              {p.label}
            </button>
          ))}
        </div>
        <p className="mt-2 text-xs text-ink-500">Нажмите — напоминание создастся сразу. Потом можно отключить или удалить ниже.</p>
      </div>

      {showForm ? <CustomForm onCreate={(input) => create.mutate(input, { onSuccess: () => setShowForm(false) })} pending={create.isPending} /> : null}

      {/* list */}
      {list.isLoading ? (
        <div className="card h-24 animate-pulse bg-ink-800/40" />
      ) : items.length > 0 ? (
        <div className="space-y-3">
          {items.map((r) => (
            <ReminderRow key={r.id} r={r} onDelete={() => del.mutate(r.id)} />
          ))}
        </div>
      ) : (
        <div className="card text-center text-ink-500">Пока нет напоминаний — выберите шаблон выше или создайте своё.</div>
      )}
    </div>
  );
}

function ReminderRow({ r, onDelete }: { r: Reminder; onDelete: () => void }) {
  const update = useUpdateReminder(r.id);
  return (
    <div className="card flex items-center justify-between gap-3">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className={`font-medium ${r.enabled ? "" : "text-ink-500 line-through"}`}>{r.title}</span>
        </div>
        <div className="text-sm text-ink-500">{summary(r)}</div>
      </div>
      <div className="flex shrink-0 items-center gap-3">
        <label className="flex cursor-pointer items-center gap-2 text-sm text-ink-400">
          <input
            type="checkbox"
            checked={r.enabled}
            onChange={(e) => update.mutate({ ...reminderToInput(r), enabled: e.target.checked })}
            className="h-4 w-4 accent-brand"
          />
          вкл
        </label>
        <button onClick={onDelete} className="text-ink-500 transition hover:text-bad" aria-label="Удалить">
          <Trash2 size={16} />
        </button>
      </div>
    </div>
  );
}

function CustomForm({ onCreate, pending }: { onCreate: (input: ReminderInput) => void; pending: boolean }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [url, setUrl] = useState("/dashboard");
  const [mode, setMode] = useState<"interval" | "times">("interval");
  const [intervalMinutes, setIntervalMinutes] = useState(120);
  const [windowStart, setWindowStart] = useState("08:00");
  const [windowEnd, setWindowEnd] = useState("22:00");
  const [times, setTimes] = useState<string[]>(["09:00"]);
  const [condition, setCondition] = useState<ReminderInput["condition"]>("");

  function submit() {
    if (!title.trim()) return;
    onCreate({
      title: title.trim(),
      body: body.trim(),
      url,
      mode,
      intervalMinutes: mode === "interval" ? intervalMinutes : null,
      windowStart,
      windowEnd,
      times: mode === "times" ? times.filter(Boolean) : [],
      condition,
      timezone: browserTZ(),
      enabled: true,
    });
  }

  return (
    <div className="card space-y-4">
      <h2 className="font-semibold">Своё напоминание</h2>
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <label className="label">Заголовок</label>
          <input className="input" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Например: Попить воды" />
        </div>
        <div>
          <label className="label">Куда ведёт</label>
          <select className="input" value={url} onChange={(e) => setUrl(e.target.value)}>
            {TARGETS.map((t) => (
              <option key={t.url} value={t.url}>{t.label}</option>
            ))}
          </select>
        </div>
      </div>
      <div>
        <label className="label">Текст (необязательно)</label>
        <input className="input" value={body} onChange={(e) => setBody(e.target.value)} placeholder="Короткое сообщение" />
      </div>

      <div className="flex gap-2">
        <button onClick={() => setMode("interval")} className={`flex-1 rounded-xl px-3 py-2 text-sm ${mode === "interval" ? "bg-ink-700 text-ink-100" : "bg-ink-800/50 text-ink-400"}`}>
          Через интервал
        </button>
        <button onClick={() => setMode("times")} className={`flex-1 rounded-xl px-3 py-2 text-sm ${mode === "times" ? "bg-ink-700 text-ink-100" : "bg-ink-800/50 text-ink-400"}`}>
          В заданное время
        </button>
      </div>

      {mode === "interval" ? (
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="label">Каждые</label>
            <select className="input" value={intervalMinutes} onChange={(e) => setIntervalMinutes(Number(e.target.value))}>
              {[30, 60, 90, 120, 180, 240, 360].map((m) => (
                <option key={m} value={m}>{fmtInterval(m)}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">С</label>
            <input type="time" className="input" value={windowStart} onChange={(e) => setWindowStart(e.target.value)} />
          </div>
          <div>
            <label className="label">До</label>
            <input type="time" className="input" value={windowEnd} onChange={(e) => setWindowEnd(e.target.value)} />
          </div>
        </div>
      ) : (
        <div>
          <label className="label">Времена</label>
          <div className="flex flex-wrap items-center gap-2">
            {times.map((t, i) => (
              <div key={i} className="flex items-center gap-1">
                <input type="time" className="input !w-auto" value={t} onChange={(e) => setTimes((prev) => prev.map((x, j) => (j === i ? e.target.value : x)))} />
                {times.length > 1 ? (
                  <button onClick={() => setTimes((prev) => prev.filter((_, j) => j !== i))} className="text-ink-500 hover:text-bad">✕</button>
                ) : null}
              </div>
            ))}
            <button onClick={() => setTimes((prev) => [...prev, "12:00"])} className="btn-ghost !py-1.5">+ время</button>
          </div>
        </div>
      )}

      <div>
        <label className="label">Условие (не беспокоить, если уже сделано)</label>
        <select className="input" value={condition} onChange={(e) => setCondition(e.target.value as ReminderInput["condition"])}>
          <option value="">Без условия — всегда</option>
          <option value="water_below_goal">Только если воды меньше дневной нормы</option>
          <option value="meds_due">Только если есть невыпитые таблетки</option>
        </select>
      </div>

      <button onClick={submit} disabled={pending} className="btn-primary">
        {pending ? "Создаём…" : "Создать напоминание"}
      </button>
    </div>
  );
}
