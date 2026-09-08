"use client";

import { useState } from "react";
import { Field } from "@/components/Field";
import { PillBar } from "@/components/PillBar";
import { todayISO } from "@/lib/api";
import { ApiRequestError } from "@/lib/api";
import {
  useCreateMed,
  useDeleteMed,
  useMedHistory,
  useMeds,
  useTakeIntake,
  useUndoIntake,
  useUpdateMed,
  type MedInput,
} from "@/hooks/useMeds";
import type { Medication } from "@/lib/types";

export default function MedsPage() {
  const meds = useMeds();
  const take = useTakeIntake();
  const undo = useUndoIntake();
  const del = useDeleteMed();

  const [editing, setEditing] = useState<Medication | null>(null);
  const [showForm, setShowForm] = useState(false);

  const items = meds.data?.items ?? [];
  const active = items.filter((m) => m.status !== "finished");
  const finished = items.filter((m) => m.status === "finished");

  const doneToday = active.filter((m) => m.takenToday >= m.timesPerDay).length;

  function onDelete(id: string) {
    if (confirm("Удалить курс? История приёмов тоже удалится.")) del.mutate(id);
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Таблетки и витамины</h1>
          <p className="text-sm text-ink-500">
            {active.length > 0
              ? `Выполнено сегодня: ${doneToday} из ${active.length}`
              : "Добавьте курс — и отмечайте приёмы каждый день."}
          </p>
        </div>
        <button
          onClick={() => {
            setEditing(null);
            setShowForm((s) => !s);
          }}
          className="btn-primary"
        >
          {showForm && !editing ? "Закрыть" : "+ Добавить"}
        </button>
      </div>

      {(showForm || editing) && (
        <MedForm
          key={editing?.id ?? "new"}
          initial={editing}
          onDone={() => {
            setEditing(null);
            setShowForm(false);
          }}
        />
      )}

      {meds.isLoading ? (
        <div className="card h-40 animate-pulse bg-ink-800/40" />
      ) : active.length > 0 ? (
        <div className="space-y-4">
          {active.map((m) => (
            <div key={m.id} className="card">
              <PillBar
                med={m}
                onTake={() => take.mutate(m.id)}
                onUndo={() => undo.mutate(m.id)}
                busy={take.isPending || undo.isPending}
              />
              <div className="mt-3 flex gap-3 border-t border-ink-800 pt-2 text-xs text-ink-500">
                <button onClick={() => setEditing(m)} className="hover:text-ink-100">
                  Изменить
                </button>
                <button onClick={() => onDelete(m.id)} className="hover:text-bad">
                  Удалить
                </button>
                {m.notes ? <span className="ml-auto italic">{m.notes}</span> : null}
              </div>
              <MedHistorySection med={m} />
            </div>
          ))}
        </div>
      ) : (
        <div className="card text-center text-ink-500">Пока нет активных курсов.</div>
      )}

      {finished.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-sm font-semibold text-ink-500">Завершённые курсы</h2>
          {finished.map((m) => (
            <div key={m.id} className="card">
              <div className="flex items-center justify-between opacity-80">
                <div>
                  <div className="font-medium">{m.name}</div>
                  <div className="text-xs text-ink-500">
                    {m.dose} {m.unit} × {m.timesPerDay}/день · курс {m.courseTotal ?? "—"} дн.
                  </div>
                </div>
                <div className="flex gap-3 text-xs text-ink-500">
                  <button onClick={() => setEditing(m)} className="hover:text-ink-100">
                    Возобновить
                  </button>
                  <button onClick={() => onDelete(m.id)} className="hover:text-bad">
                    Удалить
                  </button>
                </div>
              </div>
              <MedHistorySection med={m} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function num(v: string): number {
  const n = parseFloat(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

const UNITS = ["таблетка", "капсула", "мг", "мл", "шт", "капля", "порция"];

function MedForm({ initial, onDone }: { initial: Medication | null; onDone: () => void }) {
  const create = useCreateMed();
  const update = useUpdateMed(initial?.id ?? "");
  const editing = Boolean(initial);

  const [f, setF] = useState({
    name: initial?.name ?? "",
    unit: initial?.unit ?? "таблетка",
    dose: String(initial?.dose ?? 1),
    timesPerDay: String(initial?.timesPerDay ?? 1),
    startDate: initial?.startDate ?? todayISO(),
    durationDays: initial?.durationDays != null ? String(initial.durationDays) : "",
    notes: initial?.notes ?? "",
  });
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setFields({});
    const input: MedInput = {
      name: f.name.trim(),
      unit: f.unit.trim() || "таблетка",
      dose: num(f.dose),
      timesPerDay: Math.round(num(f.timesPerDay)),
      startDate: f.startDate,
      durationDays: f.durationDays ? Math.round(num(f.durationDays)) : null,
      notes: f.notes.trim(),
      active: true,
    };
    try {
      if (editing) await update.mutateAsync(input);
      else await create.mutateAsync(input);
      onDone();
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
        if (err.fields) setFields(err.fields);
      } else {
        setError("Не удалось сохранить");
      }
    }
  }

  const pending = create.isPending || update.isPending;

  return (
    <form onSubmit={submit} className="card space-y-4">
      <h2 className="font-semibold">{editing ? "Изменить курс" : "Новый курс"}</h2>
      <Field label="Название" name="name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} error={fields.name} placeholder="Витамин D3, Магний B6…" />

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Field label="Доза за приём" name="dose" inputMode="decimal" value={f.dose} onChange={(e) => setF({ ...f, dose: e.target.value })} error={fields.dose} />
        <div>
          <label className="label">Единица</label>
          <select className="input" value={f.unit} onChange={(e) => setF({ ...f, unit: e.target.value })}>
            {UNITS.map((u) => (
              <option key={u} value={u}>
                {u}
              </option>
            ))}
          </select>
        </div>
        <Field label="Раз в день" name="times" inputMode="numeric" value={f.timesPerDay} onChange={(e) => setF({ ...f, timesPerDay: e.target.value })} error={fields.timesPerDay} />
        <Field label="Длит., дней" name="duration" inputMode="numeric" value={f.durationDays} onChange={(e) => setF({ ...f, durationDays: e.target.value })} hint="пусто = бессрочно" error={fields.durationDays} />
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label className="label">Начало курса</label>
          <input type="date" className="input" value={f.startDate} onChange={(e) => setF({ ...f, startDate: e.target.value })} />
          {fields.startDate ? <p className="field-error">{fields.startDate}</p> : null}
        </div>
        <Field label="Заметка (необязательно)" name="notes" value={f.notes} onChange={(e) => setF({ ...f, notes: e.target.value })} placeholder="После еды" />
      </div>

      {error ? <p className="field-error">{error}</p> : null}

      <div className="flex gap-3">
        <button type="submit" disabled={pending} className="btn-primary">
          {pending ? "Сохраняем…" : editing ? "Сохранить" : "Добавить курс"}
        </button>
        <button type="button" onClick={onDone} className="btn-ghost">
          Отмена
        </button>
      </div>
    </form>
  );
}

// "YYYY-MM-DD" -> "DD.MM.YYYY"
function fmtDate(iso: string): string {
  if (iso.length < 10) return iso;
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`;
}

// Collapsible per-course intake log: when and how consistently the course was
// taken. The history is fetched lazily the first time the panel is opened.
function MedHistorySection({ med }: { med: Medication }) {
  const [open, setOpen] = useState(false);
  const hist = useMedHistory(med.id, open);
  const h = hist.data;

  const adherence =
    h && med.courseTotal && med.timesPerDay > 0
      ? Math.round((h.totalTaken / (med.timesPerDay * med.courseTotal)) * 100)
      : null;

  return (
    <div className="mt-2 border-t border-ink-800 pt-2">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1 text-xs text-ink-500 transition hover:text-ink-100"
        aria-expanded={open}
      >
        <span className="inline-block w-3">{open ? "▾" : "▸"}</span> История приёмов
      </button>

      {open ? (
        hist.isLoading ? (
          <div className="mt-2 h-16 animate-pulse rounded-lg bg-ink-800/40" />
        ) : h && h.days.length > 0 ? (
          <div className="mt-2 space-y-2">
            <p className="text-xs text-ink-400">
              Отмечено приёмов: <span className="font-semibold text-ink-100">{h.totalTaken}</span> · дней: {h.activeDays}
              {h.firstDate ? ` · ${fmtDate(h.firstDate)} — ${fmtDate(h.lastDate)}` : ""}
              {adherence != null ? <span className="text-ink-500"> · {adherence}% курса</span> : null}
            </p>
            <ul className="max-h-48 space-y-0.5 overflow-auto pr-1 text-sm">
              {[...h.days].reverse().map((d) => {
                const done = d.count >= h.timesPerDay;
                return (
                  <li key={d.date} className="flex items-center justify-between rounded-lg px-2 py-1 odd:bg-ink-800/30">
                    <span className="text-ink-300">{fmtDate(d.date)}</span>
                    <span className={done ? "font-medium text-good" : "text-ink-400"}>
                      {d.count} / {h.timesPerDay}
                      {done ? " ✓" : ""}
                    </span>
                  </li>
                );
              })}
            </ul>
          </div>
        ) : (
          <p className="mt-2 text-xs text-ink-500">Пока нет ни одного отмеченного приёма.</p>
        )
      ) : null}
    </div>
  );
}
