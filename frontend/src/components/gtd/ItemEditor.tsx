"use client";

import { useState } from "react";
import { ApiRequestError } from "@/lib/api";
import {
  itemToInput,
  useCaptureItem,
  useGtdProjects,
  useUpdateItem,
  type GtdItemInput,
} from "@/hooks/useGtd";
import { BUCKET_LABELS, CONTEXT_PRESETS, ENERGY_LABELS, PRIORITY_LABELS, isoToLocalInput, localToISO } from "@/lib/gtd";
import type { GtdBucket, GtdEnergy, GtdItem } from "@/lib/types";

const BUCKETS: GtdBucket[] = ["inbox", "next", "waiting", "calendar", "someday", "reference"];

// Full create/edit form for a GTD item. `initial` = edit; otherwise create.
export function ItemEditor({
  initial,
  defaultBucket = "next",
  onDone,
  onCancel,
}: {
  initial?: GtdItem | null;
  defaultBucket?: GtdBucket;
  onDone: () => void;
  onCancel: () => void;
}) {
  const create = useCaptureItem();
  const update = useUpdateItem();
  const projects = useGtdProjects();
  const editing = Boolean(initial);
  const base = initial ? itemToInput(initial) : null;

  const [f, setF] = useState<GtdItemInput>(
    base ?? {
      title: "",
      notes: "",
      bucket: defaultBucket,
      context: "",
      waitingFor: "",
      scheduledAt: null,
      endAt: null,
      allDay: false,
      dueOn: null,
      energy: "",
      timeMinutes: null,
      priority: 0,
      projectId: null,
    },
  );
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  const bucket = f.bucket ?? "next";
  const set = <K extends keyof GtdItemInput>(k: K, v: GtdItemInput[K]) => setF((s) => ({ ...s, [k]: v }));

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setFields({});
    if (!f.title.trim()) {
      setFields({ title: "Введите название" });
      return;
    }
    const input: GtdItemInput = { ...f, title: f.title.trim() };
    try {
      if (editing && initial) await update.mutateAsync({ id: initial.id, input });
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
    <form onSubmit={submit} className="card space-y-3">
      <div>
        <label className="label">Название</label>
        <input className="input" value={f.title} onChange={(e) => set("title", e.target.value)} placeholder="Что нужно сделать?" autoFocus />
        {fields.title ? <p className="field-error">{fields.title}</p> : null}
      </div>

      <div>
        <label className="label">Куда</label>
        <div className="flex flex-wrap gap-1.5">
          {BUCKETS.map((b) => (
            <button
              type="button"
              key={b}
              onClick={() => set("bucket", b)}
              className={`rounded-lg px-2.5 py-1.5 text-xs font-medium transition ${
                bucket === b ? "bg-brand text-ink-950" : "bg-ink-800 text-ink-300 hover:bg-ink-700"
              }`}
            >
              {BUCKET_LABELS[b]}
            </button>
          ))}
        </div>
      </div>

      {/* bucket-specific fields */}
      {bucket === "next" ? (
        <div>
          <label className="label">Контекст</label>
          <input
            className="input"
            list="gtd-contexts"
            value={f.context ?? ""}
            onChange={(e) => set("context", e.target.value)}
            placeholder="@компьютер, @звонки…"
          />
          <datalist id="gtd-contexts">
            {CONTEXT_PRESETS.map((c) => (
              <option key={c} value={c} />
            ))}
          </datalist>
        </div>
      ) : null}

      {bucket === "waiting" ? (
        <div>
          <label className="label">Ждём от кого / чего</label>
          <input className="input" value={f.waitingFor ?? ""} onChange={(e) => set("waitingFor", e.target.value)} placeholder="Например: ответа от Иры" />
        </div>
      ) : null}

      {bucket === "calendar" ? (
        <div className="space-y-2">
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <div>
              <label className="label">Начало</label>
              <input
                type="datetime-local"
                className="input"
                value={isoToLocalInput(f.scheduledAt ?? null)}
                onChange={(e) => set("scheduledAt", localToISO(e.target.value))}
              />
              {fields.scheduledAt ? <p className="field-error">{fields.scheduledAt}</p> : null}
            </div>
            <div>
              <label className="label">Конец (необязательно)</label>
              <input
                type="datetime-local"
                className="input"
                value={isoToLocalInput(f.endAt ?? null)}
                onChange={(e) => set("endAt", localToISO(e.target.value))}
              />
            </div>
          </div>
          <label className="flex cursor-pointer items-center gap-2 text-sm text-ink-400">
            <input type="checkbox" checked={f.allDay ?? false} onChange={(e) => set("allDay", e.target.checked)} className="h-4 w-4 accent-brand" />
            Весь день
          </label>
        </div>
      ) : null}

      {/* soft due date for actionable, non-calendar buckets */}
      {bucket === "next" || bucket === "waiting" ? (
        <div>
          <label className="label">Срок (необязательно)</label>
          <input type="date" className="input" value={f.dueOn ?? ""} onChange={(e) => set("dueOn", e.target.value || null)} />
        </div>
      ) : null}

      {/* engage attributes for actionable buckets */}
      {bucket === "next" || bucket === "calendar" ? (
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          <div>
            <label className="label">Приоритет</label>
            <select className="input" value={f.priority ?? 0} onChange={(e) => set("priority", Number(e.target.value))}>
              {[0, 1, 2, 3, 4, 5].map((p) => (
                <option key={p} value={p}>
                  {p === 0 ? PRIORITY_LABELS[0] : `${p} — ${PRIORITY_LABELS[p]}`}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Энергия</label>
            <select className="input" value={f.energy ?? ""} onChange={(e) => set("energy", e.target.value as GtdEnergy)}>
              {(["", "low", "medium", "high"] as GtdEnergy[]).map((en) => (
                <option key={en} value={en}>
                  {ENERGY_LABELS[en]}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Минут</label>
            <input
              type="number"
              min={1}
              className="input"
              value={f.timeMinutes ?? ""}
              onChange={(e) => set("timeMinutes", e.target.value ? Number(e.target.value) : null)}
              placeholder="15"
            />
          </div>
        </div>
      ) : null}

      <div>
        <label className="label">Проект (необязательно)</label>
        <select className="input" value={f.projectId ?? ""} onChange={(e) => set("projectId", e.target.value || null)}>
          <option value="">— без проекта —</option>
          {(projects.data?.items ?? [])
            .filter((p) => p.status === "active" || p.id === f.projectId)
            .map((p) => (
              <option key={p.id} value={p.id}>
                {p.title}
              </option>
            ))}
        </select>
      </div>

      <div>
        <label className="label">Заметки</label>
        <textarea className="input min-h-16" value={f.notes ?? ""} onChange={(e) => set("notes", e.target.value)} />
      </div>

      {error ? <p className="field-error">{error}</p> : null}

      <div className="flex gap-2">
        <button type="submit" disabled={pending} className="btn-primary">
          {pending ? "Сохраняем…" : editing ? "Сохранить" : "Добавить"}
        </button>
        <button type="button" onClick={onCancel} className="btn-ghost">
          Отмена
        </button>
      </div>
    </form>
  );
}
