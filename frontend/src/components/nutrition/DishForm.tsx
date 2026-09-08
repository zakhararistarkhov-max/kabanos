"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { Field } from "@/components/Field";
import { ApiRequestError } from "@/lib/api";
import { uploadDishImage, useDishes, type DishInput } from "@/hooks/useDishes";
import type { Dish, Macros } from "@/lib/types";

function num(v: string): number {
  const n = parseFloat(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

interface IngRow {
  dishId: string;
  name: string;
  per100g: Macros;
  grams: string;
}

interface Props {
  initial?: Dish;
  submitLabel: string;
  onSubmit: (input: DishInput) => Promise<Dish>;
}

// Shared create/edit form for a dish. Handles both a manually entered dish
// (KБЖУ per 100 g) and a composed one (built from other dishes, macros derived).
export function DishForm({ initial, submitLabel, onSubmit }: Props) {
  const router = useRouter();

  const [f, setF] = useState({
    name: initial?.name ?? "",
    description: initial?.description ?? "",
    recipe: initial?.recipe ?? "",
    kcal: initial ? String(initial.per100g.kcal) : "",
    protein: initial ? String(initial.per100g.protein) : "",
    fat: initial ? String(initial.per100g.fat) : "",
    carbs: initial ? String(initial.per100g.carbs) : "",
    serving: initial?.servingGrams != null ? String(initial.servingGrams) : "",
  });
  const [imageKey, setImageKey] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(initial?.imageUrl ?? null);
  const [uploading, setUploading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  const [ingredients, setIngredients] = useState<IngRow[]>(
    (initial?.ingredients ?? []).map((i) => ({ dishId: i.dishId, name: i.name, per100g: i.per100g, grams: String(i.grams) })),
  );
  const [search, setSearch] = useState("");
  const results = useDishes({ scope: "all", q: search, sort: "new", page: 1 });
  const composed = ingredients.length > 0;

  function set(key: keyof typeof f) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setF({ ...f, [key]: e.target.value });
  }

  // Live totals computed from ingredients, mirroring the server calculation.
  const totals = useMemo(() => {
    let g = 0, kcal = 0, p = 0, fat = 0, c = 0;
    for (const it of ingredients) {
      const gr = num(it.grams);
      const k = gr / 100;
      g += gr;
      kcal += it.per100g.kcal * k;
      p += it.per100g.protein * k;
      fat += it.per100g.fat * k;
      c += it.per100g.carbs * k;
    }
    const per = (x: number) => (g > 0 ? +((x / g) * 100).toFixed(1) : 0);
    return {
      grams: +g.toFixed(1),
      kcal: Math.round(kcal),
      protein: +p.toFixed(1),
      fat: +fat.toFixed(1),
      carbs: +c.toFixed(1),
      per100: { kcal: Math.round(per(kcal)), protein: per(p), fat: per(fat), carbs: per(c) },
    };
  }, [ingredients]);

  function addIngredient(id: string, name: string, per100g: Macros) {
    if (id === initial?.id) return; // a dish can't contain itself
    if (ingredients.some((i) => i.dishId === id)) return;
    setIngredients([...ingredients, { dishId: id, name, per100g, grams: "100" }]);
    setSearch("");
  }
  function patchGrams(id: string, grams: string) {
    setIngredients((prev) => prev.map((i) => (i.dishId === id ? { ...i, grams } : i)));
  }
  function removeIngredient(id: string) {
    setIngredients((prev) => prev.filter((i) => i.dishId !== id));
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
    setBusy(true);
    const input: DishInput = {
      name: f.name.trim(),
      description: f.description.trim(),
      recipe: f.recipe.trim(),
      // On edit, only override the image when a new one was uploaded; sending
      // null back would clear it (the server keeps the existing key otherwise).
      imageKey: imageKey ?? (initial ? undefined : null),
      kcalPer100: composed ? 0 : num(f.kcal),
      proteinPer100: composed ? 0 : num(f.protein),
      fatPer100: composed ? 0 : num(f.fat),
      carbsPer100: composed ? 0 : num(f.carbs),
      servingGrams: composed ? null : f.serving ? num(f.serving) : null,
      ingredients: composed ? ingredients.map((i) => ({ dishId: i.dishId, grams: num(i.grams) })) : undefined,
    };
    try {
      const dish = await onSubmit(input);
      router.push(`/dishes/${dish.id}`);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
        if (err.fields) setFields(err.fields);
      } else {
        setError("Не удалось сохранить блюдо");
      }
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="card space-y-4">
      <Field label="Название" name="name" value={f.name} onChange={set("name")} error={fields.name} />

      <div>
        <label className="label">Описание</label>
        <textarea className="input min-h-20" value={f.description} onChange={set("description")} />
      </div>

      {/* ingredients */}
      <div className="rounded-xl border border-ink-800 bg-ink-950/40 p-3">
        <div className="mb-2 flex items-center justify-between">
          <h2 className="font-semibold">Ингредиенты (блюда)</h2>
          {composed ? <span className="text-xs text-ink-500">итог: {totals.grams} г</span> : null}
        </div>

        {ingredients.length > 0 ? (
          <ul className="mb-3 space-y-2">
            {ingredients.map((it) => {
              const gr = num(it.grams) / 100;
              return (
                <li key={it.dishId} className="flex items-center gap-2 rounded-lg bg-ink-900/60 px-2 py-1.5">
                  <span className="min-w-0 flex-1 truncate text-sm">{it.name}</span>
                  <span className="hidden shrink-0 text-xs text-ink-500 sm:inline">
                    {Math.round(it.per100g.kcal * gr)} ккал · Б{+(it.per100g.protein * gr).toFixed(1)} Ж{+(it.per100g.fat * gr).toFixed(1)} У{+(it.per100g.carbs * gr).toFixed(1)}
                  </span>
                  <input
                    className="input !w-20 !py-1 text-sm"
                    inputMode="decimal"
                    value={it.grams}
                    onChange={(e) => patchGrams(it.dishId, e.target.value)}
                    aria-label="граммы"
                  />
                  <span className="text-xs text-ink-500">г</span>
                  <button type="button" onClick={() => removeIngredient(it.dishId)} className="text-ink-500 hover:text-bad">
                    ✕
                  </button>
                </li>
              );
            })}
          </ul>
        ) : (
          <p className="mb-3 text-xs text-ink-500">Найдите блюдо ниже и добавьте — например «Кабачок», «Морковь». Или оставьте пусто и введите КБЖУ вручную.</p>
        )}

        <input className="input" placeholder="Поиск блюда для добавления…" value={search} onChange={(e) => setSearch(e.target.value)} />
        {search ? (
          <div className="mt-2 max-h-44 space-y-1 overflow-auto">
            {(results.data?.items ?? [])
              .filter((d) => d.id !== initial?.id && !ingredients.some((i) => i.dishId === d.id))
              .slice(0, 8)
              .map((d) => (
                <button
                  type="button"
                  key={d.id}
                  onClick={() => addIngredient(d.id, d.name, d.per100g)}
                  className="flex w-full items-center justify-between rounded-lg px-2 py-1.5 text-left text-sm hover:bg-ink-800/60"
                >
                  <span className="truncate">{d.name}</span>
                  <span className="shrink-0 text-xs text-ink-500">{Math.round(d.per100g.kcal)} ккал/100г ＋</span>
                </button>
              ))}
            {results.data && results.data.items.length === 0 ? <p className="px-2 text-xs text-ink-500">Ничего не найдено.</p> : null}
          </div>
        ) : null}

        {composed ? (
          <div className="mt-3 rounded-lg bg-brand/10 px-3 py-2 text-sm text-ink-200">
            Рассчитано: <span className="font-semibold">{totals.per100.kcal} ккал / 100 г</span> · Б {totals.per100.protein} Ж {totals.per100.fat} У {totals.per100.carbs}
            <span className="text-ink-500"> · всего {totals.kcal} ккал на {totals.grams} г</span>
          </div>
        ) : null}
      </div>

      {/* manual macros only when not composed */}
      {!composed ? (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Field label="Ккал/100г" name="kcal" inputMode="decimal" value={f.kcal} onChange={set("kcal")} error={fields.kcalPer100} />
            <Field label="Белки/100г" name="protein" inputMode="decimal" value={f.protein} onChange={set("protein")} />
            <Field label="Жиры/100г" name="fat" inputMode="decimal" value={f.fat} onChange={set("fat")} />
            <Field label="Углев./100г" name="carbs" inputMode="decimal" value={f.carbs} onChange={set("carbs")} />
          </div>
          <Field label="Размер порции, г (необязательно)" name="serving" inputMode="decimal" value={f.serving} onChange={set("serving")} hint="Позволит добавлять блюдо порциями" />
        </>
      ) : null}

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
            {uploading ? "Загрузка…" : preview ? "Заменить фото" : "Выбрать фото"}
            <input type="file" accept="image/png,image/jpeg,image/webp" className="hidden" onChange={onFile} disabled={uploading} />
          </label>
        </div>
      </div>

      {error ? <p className="field-error">{error}</p> : null}
      {fields.ingredients ? <p className="field-error">{fields.ingredients}</p> : null}

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
