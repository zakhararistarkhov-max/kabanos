"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { Field } from "@/components/Field";
import { ApiRequestError } from "@/lib/api";
import { uploadDishImage, useCreateDish, useDishes, type DishInput } from "@/hooks/useDishes";
import type { Macros } from "@/lib/types";

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

export default function NewDishPage() {
  const router = useRouter();
  const create = useCreateDish();

  const [f, setF] = useState({ name: "", description: "", recipe: "", kcal: "", protein: "", fat: "", carbs: "", serving: "" });
  const [imageKey, setImageKey] = useState<string | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  const [ingredients, setIngredients] = useState<IngRow[]>([]);
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
    const input: DishInput = {
      name: f.name.trim(),
      description: f.description.trim(),
      recipe: f.recipe.trim(),
      imageKey,
      kcalPer100: composed ? 0 : num(f.kcal),
      proteinPer100: composed ? 0 : num(f.protein),
      fatPer100: composed ? 0 : num(f.fat),
      carbsPer100: composed ? 0 : num(f.carbs),
      servingGrams: composed ? null : f.serving ? num(f.serving) : null,
      ingredients: composed ? ingredients.map((i) => ({ dishId: i.dishId, grams: num(i.grams) })) : undefined,
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
        <p className="text-sm text-ink-500">
          Укажите КБЖУ вручную (на 100 г) — или соберите блюдо из ингредиентов, и КБЖУ посчитается само.
        </p>
      </div>

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
                .filter((d) => !ingredients.some((i) => i.dishId === d.id))
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
              {uploading ? "Загрузка…" : imageKey ? "Заменить фото" : "Выбрать фото"}
              <input type="file" accept="image/png,image/jpeg,image/webp" className="hidden" onChange={onFile} disabled={uploading} />
            </label>
          </div>
        </div>

        {error ? <p className="field-error">{error}</p> : null}
        {fields.ingredients ? <p className="field-error">{fields.ingredients}</p> : null}

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
