"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Field } from "@/components/Field";
import { ApiRequestError } from "@/lib/api";
import {
  useExercises,
  useTrainingMeta,
  uploadWorkoutImage,
  type WorkoutInput,
} from "@/hooks/useTraining";
import { CATEGORY_LABELS, DIFFICULTY_LABELS, label } from "@/lib/training";
import type { Workout } from "@/lib/types";

interface BuilderItem {
  key: string;
  exerciseId: string;
  name: string;
  imageUrl: string | null;
  category: string;
  difficulty: string;
  sets: string;
  reps: string;
  durationSec: string;
  restSec: string;
  weightKg: string;
  note: string;
}

interface Props {
  initial?: Workout;
  submitLabel: string;
  onSubmit: (input: WorkoutInput) => Promise<Workout>;
}

let keySeq = 0;
const nextKey = () => `it-${keySeq++}`;

function numOrNull(s: string): number | null {
  const n = parseFloat(s.replace(",", "."));
  return Number.isFinite(n) && n > 0 ? n : null;
}

export function WorkoutBuilder({ initial, submitLabel, onSubmit }: Props) {
  const router = useRouter();
  const meta = useTrainingMeta();

  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [difficulty, setDifficulty] = useState<string>(initial?.difficulty ?? "medium");
  const [items, setItems] = useState<BuilderItem[]>(
    (initial?.items ?? []).map((it) => ({
      key: nextKey(),
      exerciseId: it.exerciseId,
      name: it.exercise.name,
      imageUrl: it.exercise.imageUrl,
      category: it.exercise.category,
      difficulty: it.exercise.difficulty,
      sets: it.sets?.toString() ?? "",
      reps: it.reps?.toString() ?? "",
      durationSec: it.durationSec?.toString() ?? "",
      restSec: it.restSec?.toString() ?? "",
      weightKg: it.weightKg?.toString() ?? "",
      note: it.note ?? "",
    })),
  );

  const [imageKey, setImageKey] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(initial?.imageUrl ?? null);
  const [uploading, setUploading] = useState(false);

  const [search, setSearch] = useState("");
  const results = useExercises({ scope: "all", q: search, sort: "new", page: 1 });

  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  function addExercise(exId: string, exName: string, imageUrl: string | null, category: string, diff: string) {
    setItems((prev) => [
      ...prev,
      { key: nextKey(), exerciseId: exId, name: exName, imageUrl, category, difficulty: diff, sets: "", reps: "", durationSec: "", restSec: "", weightKg: "", note: "" },
    ]);
  }

  function patch(key: string, field: keyof BuilderItem, value: string) {
    setItems((prev) => prev.map((it) => (it.key === key ? { ...it, [field]: value } : it)));
  }

  function move(index: number, dir: -1 | 1) {
    setItems((prev) => {
      const next = [...prev];
      const j = index + dir;
      if (j < 0 || j >= next.length) return prev;
      [next[index], next[j]] = [next[j], next[index]];
      return next;
    });
  }

  function removeItem(key: string) {
    setItems((prev) => prev.filter((it) => it.key !== key));
  }

  async function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      const key = await uploadWorkoutImage(file);
      setImageKey(key);
      setPreview(URL.createObjectURL(file));
    } catch {
      setError("Не удалось загрузить изображение");
    } finally {
      setUploading(false);
    }
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setFields({});
    if (items.length === 0) {
      setError("Добавьте хотя бы одно упражнение");
      return;
    }
    setBusy(true);
    const input: WorkoutInput = {
      name: name.trim(),
      description: description.trim(),
      difficulty,
      imageKey: imageKey ?? (initial ? undefined : null),
      items: items.map((it) => ({
        exerciseId: it.exerciseId,
        sets: numOrNull(it.sets),
        reps: numOrNull(it.reps),
        durationSec: numOrNull(it.durationSec),
        restSec: numOrNull(it.restSec),
        weightKg: numOrNull(it.weightKg),
        note: it.note.trim(),
      })),
    };
    try {
      const wk = await onSubmit(input);
      router.push(`/workouts/${wk.id}`);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
        if (err.fields) setFields(err.fields);
      } else {
        setError("Не удалось сохранить тренировку");
      }
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-6">
      <div className="card space-y-4">
        <Field label="Название тренировки" name="name" value={name} onChange={(e) => setName(e.target.value)} error={fields.name} />
        <div>
          <label className="label">Описание</label>
          <textarea className="input min-h-20" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <div>
            <label className="label">Сложность</label>
            <select className="input" value={difficulty} onChange={(e) => setDifficulty(e.target.value)}>
              {(meta.data?.difficulties ?? []).map((o) => (
                <option key={o.key} value={o.key}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Обложка</label>
            <div className="flex items-center gap-3">
              <div className="h-12 w-12 overflow-hidden rounded-lg border border-ink-700 bg-ink-800">
                {preview ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={preview} alt="" className="h-full w-full object-cover" />
                ) : (
                  <div className="flex h-full items-center justify-center opacity-40">🏋️</div>
                )}
              </div>
              <label className="btn-ghost cursor-pointer !py-2">
                {uploading ? "Загрузка…" : preview ? "Заменить" : "Загрузить"}
                <input type="file" accept="image/png,image/jpeg,image/webp" className="hidden" onChange={onFile} disabled={uploading} />
              </label>
            </div>
          </div>
        </div>
      </div>

      {/* selected exercises */}
      <div className="card space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold">Упражнения ({items.length})</h2>
        </div>
        {items.length === 0 ? (
          <p className="text-sm text-ink-500">Добавьте упражнения из каталога ниже.</p>
        ) : (
          <ul className="space-y-3">
            {items.map((it, i) => (
              <li key={it.key} className="rounded-xl border border-ink-800 bg-ink-950/40 p-3">
                <div className="flex items-start gap-3">
                  <div className="flex flex-col items-center gap-1 pt-1">
                    <button type="button" onClick={() => move(i, -1)} disabled={i === 0} className="text-ink-500 disabled:opacity-30 hover:text-ink-100">
                      ▲
                    </button>
                    <span className="text-xs text-ink-500">{i + 1}</span>
                    <button type="button" onClick={() => move(i, 1)} disabled={i === items.length - 1} className="text-ink-500 disabled:opacity-30 hover:text-ink-100">
                      ▼
                    </button>
                  </div>
                  <div className="h-12 w-12 shrink-0 overflow-hidden rounded-lg bg-ink-800">
                    {it.imageUrl ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={it.imageUrl} alt="" className="h-full w-full object-cover" />
                    ) : (
                      <div className="flex h-full items-center justify-center opacity-40">🏋️</div>
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <p className="truncate font-medium">{it.name}</p>
                      <button type="button" onClick={() => removeItem(it.key)} className="shrink-0 text-sm text-ink-500 hover:text-bad">
                        убрать
                      </button>
                    </div>
                    <p className="text-xs text-ink-500">
                      {label(CATEGORY_LABELS, it.category)} · {label(DIFFICULTY_LABELS, it.difficulty)}
                    </p>
                    <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-5">
                      <MiniInput label="Подходы" value={it.sets} onChange={(v) => patch(it.key, "sets", v)} />
                      <MiniInput label="Повт." value={it.reps} onChange={(v) => patch(it.key, "reps", v)} />
                      <MiniInput label="Время,с" value={it.durationSec} onChange={(v) => patch(it.key, "durationSec", v)} />
                      <MiniInput label="Отдых,с" value={it.restSec} onChange={(v) => patch(it.key, "restSec", v)} />
                      <MiniInput label="Вес,кг" value={it.weightKg} onChange={(v) => patch(it.key, "weightKg", v)} />
                    </div>
                    <input
                      className="input mt-2 !py-1.5 text-sm"
                      placeholder="Заметка (например: до отказа)"
                      value={it.note}
                      onChange={(e) => patch(it.key, "note", e.target.value)}
                    />
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
        {fields.items ? <p className="field-error">{fields.items}</p> : null}
      </div>

      {/* exercise picker */}
      <div className="card space-y-3">
        <h2 className="font-semibold">Добавить упражнение</h2>
        <input className="input" placeholder="Поиск упражнений…" value={search} onChange={(e) => setSearch(e.target.value)} />
        <div className="grid gap-2 sm:grid-cols-2">
          {(results.data?.items ?? []).map((ex) => (
            <button
              type="button"
              key={ex.id}
              onClick={() => addExercise(ex.id, ex.name, ex.imageUrl, ex.category, ex.difficulty)}
              className="flex items-center gap-3 rounded-xl border border-ink-800 bg-ink-950/40 p-2 text-left transition hover:border-brand/50"
            >
              <div className="h-10 w-10 shrink-0 overflow-hidden rounded-lg bg-ink-800">
                {ex.imageUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={ex.imageUrl} alt="" className="h-full w-full object-cover" />
                ) : (
                  <div className="flex h-full items-center justify-center text-sm opacity-40">🏋️</div>
                )}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{ex.name}</p>
                <p className="text-xs text-ink-500">
                  {label(CATEGORY_LABELS, ex.category)} · {label(DIFFICULTY_LABELS, ex.difficulty)}
                </p>
              </div>
              <span className="shrink-0 text-brand">＋</span>
            </button>
          ))}
          {results.data && results.data.items.length === 0 ? (
            <p className="text-sm text-ink-500">Ничего не найдено.</p>
          ) : null}
        </div>
      </div>

      {error ? <p className="field-error">{error}</p> : null}
      <div className="flex gap-3">
        <button type="submit" disabled={busy || uploading} className="btn-primary">
          {busy ? "Сохраняем…" : submitLabel}
        </button>
        <button type="button" onClick={() => router.back()} className="btn-ghost">
          Отмена
        </button>
      </div>
    </form>
  );
}

function MiniInput({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="block">
      <span className="mb-1 block text-[11px] text-ink-500">{label}</span>
      <input className="input !py-1.5 text-sm" inputMode="decimal" value={value} onChange={(e) => onChange(e.target.value)} />
    </label>
  );
}
