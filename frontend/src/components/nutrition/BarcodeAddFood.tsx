"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiRequestError } from "@/lib/api";
import { useLookupBarcode } from "@/hooks/useNutrition";
import { useCreateDish } from "@/hooks/useDishes";
import { BarcodeScanner } from "@/components/nutrition/BarcodeScanner";
import type { FoodProduct, Meal } from "@/lib/types";

const MEALS: { value: Meal; label: string }[] = [
  { value: "breakfast", label: "Завтрак" },
  { value: "lunch", label: "Обед" },
  { value: "dinner", label: "Ужин" },
  { value: "snack", label: "Перекус" },
];

type Added = { name: string; kcal: number; protein: number; fat: number; carbs: number; fiber: number; meal?: Meal };

// BarcodeAddFood drives the whole scan→lookup→confirm→log flow. On confirm it
// hands an absolute-macro entry to onAdd (same shape the manual form uses).
export function BarcodeAddFood({ onAdd, onClose }: { onAdd: (v: Added) => void; onClose: () => void }) {
  const router = useRouter();
  const lookup = useLookupBarcode();
  const createDish = useCreateDish();
  const [product, setProduct] = useState<FoodProduct | null>(null);
  const [grams, setGrams] = useState("100");
  const [meal, setMeal] = useState<Meal | "">("");
  const [error, setError] = useState<string | null>(null);

  const dishName = (p: FoodProduct) => (p.brand && !p.name.includes(p.brand) ? `${p.name} (${p.brand})` : p.name);

  async function saveAsDish() {
    if (!product) return;
    setError(null);
    try {
      const dish = await createDish.mutateAsync({
        name: dishName(product),
        description: `Из штрихкода ${product.barcode}`,
        recipe: "",
        imageKey: null,
        kcalPer100: product.per100g.kcal,
        proteinPer100: product.per100g.protein,
        fatPer100: product.per100g.fat,
        carbsPer100: product.per100g.carbs,
        fiberPer100: product.per100g.fiber,
        servingGrams: product.servingGrams ?? null,
      });
      router.push(`/dishes/${dish.id}`);
    } catch {
      setError("Не удалось сохранить блюдо.");
    }
  }

  async function onDetected(code: string) {
    setError(null);
    try {
      const p = await lookup.mutateAsync(code);
      setProduct(p);
      setGrams(p.servingGrams ? String(p.servingGrams) : "100");
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 404) {
        setError(`Штрихкод ${code} не найден в базе. Попробуйте другой или добавьте вручную.`);
      } else {
        setError("Не удалось получить данные о продукте. Попробуйте ещё раз.");
      }
    }
  }

  const g = parseFloat(grams.replace(",", ".")) || 0;
  const factor = g / 100;
  const preview = product
    ? {
        kcal: Math.round(product.per100g.kcal * factor),
        protein: +(product.per100g.protein * factor).toFixed(1),
        fat: +(product.per100g.fat * factor).toFixed(1),
        carbs: +(product.per100g.carbs * factor).toFixed(1),
        fiber: +(product.per100g.fiber * factor).toFixed(1),
      }
    : null;

  function confirm() {
    if (!product || !preview || g <= 0) return;
    onAdd({ name: dishName(product), kcal: preview.kcal, protein: preview.protein, fat: preview.fat, carbs: preview.carbs, fiber: preview.fiber, meal: meal || undefined });
    onClose();
  }

  return (
    <div className="card space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold">Сканировать штрихкод</h2>
        <button onClick={onClose} className="text-sm text-ink-500 hover:text-ink-100">
          Закрыть
        </button>
      </div>

      {!product ? (
        <>
          <BarcodeScanner onDetected={onDetected} />
          {lookup.isPending ? <p className="text-center text-sm text-ink-400">Ищем продукт…</p> : null}
          {error ? <p className="field-error">{error}</p> : null}
        </>
      ) : (
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <div className="h-16 w-16 shrink-0 overflow-hidden rounded-xl border border-ink-800 bg-ink-800">
              {product.imageUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={product.imageUrl} alt="" className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-2xl opacity-40">🏷️</div>
              )}
            </div>
            <div className="min-w-0">
              <div className="truncate font-medium">{product.name}</div>
              <div className="text-xs text-ink-500">
                {product.brand ? product.brand + " · " : ""}
                {product.per100g.kcal} ккал/100г · Б{product.per100g.protein} Ж{product.per100g.fat} У{product.per100g.carbs} Кл{product.per100g.fiber}
              </div>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Количество, г</label>
              <input className="input" inputMode="decimal" value={grams} onChange={(e) => setGrams(e.target.value)} />
            </div>
            <div>
              <label className="label">Приём пищи</label>
              <select className="input" value={meal} onChange={(e) => setMeal(e.target.value as Meal | "")}>
                <option value="">—</option>
                {MEALS.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {preview ? (
            <div className="rounded-xl bg-ink-800/50 px-3 py-2 text-sm text-ink-300">
              ≈ <span className="font-semibold text-ink-100">{preview.kcal} ккал</span> · Б {preview.protein} · Ж {preview.fat} · У {preview.carbs} · Кл {preview.fiber}
            </div>
          ) : null}

          {product.per100g.kcal === 0 ? (
            <p className="text-xs text-warn">У продукта нет данных о калорийности в базе — проверьте и при необходимости добавьте вручную.</p>
          ) : null}

          <div className="flex flex-wrap gap-2">
            <button onClick={confirm} disabled={g <= 0} className="btn-primary">
              Добавить в рацион
            </button>
            <button onClick={saveAsDish} disabled={createDish.isPending} className="btn-ghost">
              {createDish.isPending ? "Сохраняем…" : "Сохранить как блюдо"}
            </button>
            <button
              onClick={() => {
                setProduct(null);
                setError(null);
              }}
              className="btn-ghost"
            >
              Сканировать другой
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
