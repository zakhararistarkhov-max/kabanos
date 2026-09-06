"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Field } from "@/components/Field";
import { ApiRequestError } from "@/lib/api";
import { uploadDishImage, useCreateDish, type DishInput } from "@/hooks/useDishes";

function num(v: string): number {
  const n = parseFloat(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

export default function NewDishPage() {
  const router = useRouter();
  const create = useCreateDish();

  const [f, setF] = useState({
    name: "",
    description: "",
    recipe: "",
    kcal: "",
    protein: "",
    fat: "",
    carbs: "",
    serving: "",
  });
  const [imageKey, setImageKey] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  function set(key: keyof typeof f) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setF({ ...f, [key]: e.target.value });
  }

  async function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setError(null);
    try {
      const key = await uploadDishImage(file);
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
    const input: DishInput = {
      name: f.name.trim(),
      description: f.description.trim(),
      recipe: f.recipe.trim(),
      imageKey,
      kcalPer100: num(f.kcal),
      proteinPer100: num(f.protein),
      fatPer100: num(f.fat),
      carbsPer100: num(f.carbs),
      servingGrams: f.serving ? num(f.serving) : null,
    };
    try {
      const dish = await create.mutateAsync(input);
      router.push(`/dishes/${dish.id}`);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
        if (err.fields) setFields(err.fields);
      } else {
        setError("Не удалось создать блюдо");
      }
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Новое блюдо</h1>
        <p className="text-sm text-ink-500">КБЖУ указываются на 100 грамм.</p>
      </div>

      <form onSubmit={submit} className="card space-y-4">
        <Field label="Название" name="name" value={f.name} onChange={set("name")} error={fields.name} />

        <div>
          <label className="label">Описание</label>
          <textarea className="input min-h-20" value={f.description} onChange={set("description")} />
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Field label="Ккал/100г" name="kcal" inputMode="decimal" value={f.kcal} onChange={set("kcal")} error={fields.kcalPer100} />
          <Field label="Белки/100г" name="protein" inputMode="decimal" value={f.protein} onChange={set("protein")} />
          <Field label="Жиры/100г" name="fat" inputMode="decimal" value={f.fat} onChange={set("fat")} />
          <Field label="Углев./100г" name="carbs" inputMode="decimal" value={f.carbs} onChange={set("carbs")} />
        </div>

        <Field label="Размер порции, г (необязательно)" name="serving" inputMode="decimal" value={f.serving} onChange={set("serving")} hint="Позволит добавлять блюдо порциями" />

        <div>
          <label className="label">Рецепт</label>
          <textarea className="input min-h-32" value={f.recipe} onChange={set("recipe")} placeholder="Ингредиенты и шаги приготовления…" />
        </div>

        <div>
          <label className="label">Фото</label>
          <div className="flex items-center gap-4">
            <div className="h-24 w-24 overflow-hidden rounded-xl border border-ink-700 bg-ink-800">
              {preview ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={preview} alt="preview" className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-2xl opacity-40">🍽️</div>
              )}
            </div>
            <label className="btn-ghost cursor-pointer">
              {uploading ? "Загрузка…" : imageKey ? "Заменить фото" : "Выбрать фото"}
              <input type="file" accept="image/png,image/jpeg,image/webp" className="hidden" onChange={onFile} disabled={uploading} />
            </label>
          </div>
        </div>

        {error ? <p className="field-error">{error}</p> : null}

        <div className="flex gap-3">
          <button type="submit" disabled={create.isPending || uploading} className="btn-primary">
            {create.isPending ? "Создаём…" : "Создать блюдо"}
          </button>
          <button type="button" onClick={() => router.push("/dishes")} className="btn-ghost">
            Отмена
          </button>
        </div>
      </form>
    </div>
  );
}
