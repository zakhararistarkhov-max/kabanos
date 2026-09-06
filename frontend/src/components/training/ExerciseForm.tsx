"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Field } from "@/components/Field";
import { ApiRequestError } from "@/lib/api";
import { useTrainingMeta, uploadExerciseImage, type ExerciseInput } from "@/hooks/useTraining";
import { label as lbl, EQUIPMENT_LABELS } from "@/lib/training";
import type { Exercise, Option } from "@/lib/types";

interface Props {
  initial?: Exercise;
  submitLabel: string;
  onSubmit: (input: ExerciseInput) => Promise<Exercise>;
}

// Shared create/edit form for an exercise.
export function ExerciseForm({ initial, submitLabel, onSubmit }: Props) {
  const router = useRouter();
  const meta = useTrainingMeta();

  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [category, setCategory] = useState<string>(initial?.category ?? "strength");
  const [difficulty, setDifficulty] = useState<string>(initial?.difficulty ?? "medium");
  const [jointImpact, setJointImpact] = useState<string>(initial?.jointImpact ?? "medium");
  const [equipment, setEquipment] = useState<string[]>(initial?.equipment ?? []);
  const [muscles, setMuscles] = useState<string[]>(initial?.muscles ?? []);
  const [videoUrl, setVideoUrl] = useState(initial?.videoUrl ?? "");
  const [customEquip, setCustomEquip] = useState("");

  const [imageKey, setImageKey] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(initial?.imageUrl ?? null);
  const [uploading, setUploading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  function toggle(list: string[], set: (v: string[]) => void, key: string) {
    set(list.includes(key) ? list.filter((k) => k !== key) : [...list, key]);
  }

  function addCustomEquip() {
    const k = customEquip.trim().toLowerCase();
    if (k && !equipment.includes(k)) setEquipment([...equipment, k]);
    setCustomEquip("");
  }

  async function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setError(null);
    try {
      const key = await uploadExerciseImage(file);
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
    setBusy(true);
    const input: ExerciseInput = {
      name: name.trim(),
      description: description.trim(),
      category,
      difficulty,
      jointImpact,
      equipment,
      muscles,
      videoUrl: videoUrl.trim(),
      // On edit, only override the image when a new one was uploaded; sending
      // the existing key back is unnecessary because the server keeps it.
      imageKey: imageKey ?? (initial ? undefined : null),
    };
    try {
      const ex = await onSubmit(input);
      router.push(`/exercises/${ex.id}`);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
        if (err.fields) setFields(err.fields);
      } else {
        setError("Не удалось сохранить упражнение");
      }
      setBusy(false);
    }
  }

  const equipOptions: Option[] = meta.data?.equipment ?? [];
  const muscleOptions: Option[] = meta.data?.muscles ?? [];

  return (
    <form onSubmit={submit} className="card space-y-4">
      <Field label="Название" name="name" value={name} onChange={(e) => setName(e.target.value)} error={fields.name} />

      <div>
        <label className="label">Описание / техника выполнения</label>
        <textarea className="input min-h-24" value={description} onChange={(e) => setDescription(e.target.value)} />
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div>
          <label className="label">Тип</label>
          <select className="input" value={category} onChange={(e) => setCategory(e.target.value)}>
            {(meta.data?.categories ?? []).map((o) => (
              <option key={o.key} value={o.key}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
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
          <label className="label">Нагрузка на суставы</label>
          <select className="input" value={jointImpact} onChange={(e) => setJointImpact(e.target.value)}>
            {(meta.data?.jointImpacts ?? []).map((o) => (
              <option key={o.key} value={o.key}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <label className="label">Инвентарь</label>
        <div className="flex flex-wrap gap-2">
          {equipOptions.map((o) => (
            <Chip key={o.key} active={equipment.includes(o.key)} onClick={() => toggle(equipment, setEquipment, o.key)}>
              {o.label}
            </Chip>
          ))}
          {equipment
            .filter((k) => !equipOptions.some((o) => o.key === k))
            .map((k) => (
              <Chip key={k} active onClick={() => toggle(equipment, setEquipment, k)}>
                {lbl(EQUIPMENT_LABELS, k)} ✕
              </Chip>
            ))}
        </div>
        <div className="mt-2 flex gap-2">
          <input
            className="input max-w-xs"
            placeholder="Свой инвентарь…"
            value={customEquip}
            onChange={(e) => setCustomEquip(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                addCustomEquip();
              }
            }}
          />
          <button type="button" onClick={addCustomEquip} className="btn-ghost shrink-0">
            Добавить
          </button>
        </div>
      </div>

      <div>
        <label className="label">Мышечные группы</label>
        <div className="flex flex-wrap gap-2">
          {muscleOptions.map((o) => (
            <Chip key={o.key} active={muscles.includes(o.key)} onClick={() => toggle(muscles, setMuscles, o.key)}>
              {o.label}
            </Chip>
          ))}
        </div>
      </div>

      <Field
        label="Ссылка на видео (необязательно)"
        name="videoUrl"
        value={videoUrl}
        onChange={(e) => setVideoUrl(e.target.value)}
        placeholder="https://youtu.be/…"
        error={fields.videoUrl}
        hint="Например, ссылка на YouTube с демонстрацией"
      />

      <div>
        <label className="label">Фото</label>
        <div className="flex items-center gap-4">
          <div className="h-24 w-24 overflow-hidden rounded-xl border border-ink-700 bg-ink-800">
            {preview ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={preview} alt="preview" className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full items-center justify-center text-2xl opacity-40">🏋️</div>
            )}
          </div>
          <label className="btn-ghost cursor-pointer">
            {uploading ? "Загрузка…" : preview ? "Заменить фото" : "Выбрать фото"}
            <input type="file" accept="image/png,image/jpeg,image/webp" className="hidden" onChange={onFile} disabled={uploading} />
          </label>
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

function Chip({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full px-3 py-1 text-sm transition ${
        active ? "bg-brand text-ink-950" : "bg-ink-800 text-ink-300 hover:bg-ink-700"
      }`}
    >
      {children}
    </button>
  );
}
