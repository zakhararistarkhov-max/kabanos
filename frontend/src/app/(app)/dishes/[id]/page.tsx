"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { Stars } from "@/components/Stars";
import { Field } from "@/components/Field";
import {
  useAddComment,
  useAddDishToDiet,
  useDeleteComment,
  useDeleteDish,
  useDish,
  useDishComments,
  useRateDish,
  useToggleFavorite,
} from "@/hooks/useDishes";
import type { Meal } from "@/lib/types";

const MEALS: { value: Meal; label: string }[] = [
  { value: "breakfast", label: "Завтрак" },
  { value: "lunch", label: "Обед" },
  { value: "dinner", label: "Ужин" },
  { value: "snack", label: "Перекус" },
];

function num(v: string): number {
  const n = parseFloat(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

export default function DishDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const router = useRouter();

  const dish = useDish(id);
  const comments = useDishComments(id);
  const rate = useRateDish(id);
  const favorite = useToggleFavorite();
  const addToDiet = useAddDishToDiet(id);
  const addComment = useAddComment(id);
  const deleteComment = useDeleteComment(id);
  const deleteDish = useDeleteDish();

  const [unit, setUnit] = useState<"g" | "serving">("g");
  const [amount, setAmount] = useState("100");
  const [meal, setMeal] = useState<Meal | "">("");
  const [added, setAdded] = useState(false);
  const [commentBody, setCommentBody] = useState("");

  const d = dish.data;

  const previewGrams = useMemo(() => {
    if (!d) return 0;
    if (unit === "serving") return num(amount) * (d.servingGrams ?? 0);
    return num(amount);
  }, [d, unit, amount]);

  const preview = useMemo(() => {
    if (!d) return null;
    const f = previewGrams / 100;
    return {
      kcal: Math.round(d.per100g.kcal * f),
      protein: +(d.per100g.protein * f).toFixed(1),
      fat: +(d.per100g.fat * f).toFixed(1),
      carbs: +(d.per100g.carbs * f).toFixed(1),
    };
  }, [d, previewGrams]);

  if (dish.isLoading) return <div className="card h-64 animate-pulse bg-ink-800/40" />;
  if (dish.isError || !d)
    return (
      <div className="card text-center text-ink-500">
        Блюдо не найдено.{" "}
        <Link href="/dishes" className="text-brand">
          К каталогу
        </Link>
      </div>
    );

  function submitToDiet() {
    const payload =
      unit === "serving" ? { servings: num(amount), meal: meal || undefined } : { grams: num(amount), meal: meal || undefined };
    addToDiet.mutate(payload, {
      onSuccess: () => {
        setAdded(true);
        setTimeout(() => setAdded(false), 2500);
      },
    });
  }

  async function onDelete() {
    if (!confirm("Удалить это блюдо?")) return;
    await deleteDish.mutateAsync(id);
    router.push("/dishes");
  }

  return (
    <div className="space-y-6">
      <Link href="/dishes" className="text-sm text-ink-500 hover:text-ink-100">
        ← Все блюда
      </Link>

      <div className="grid gap-6 md:grid-cols-[1.2fr_1fr]">
        {/* left: media + info */}
        <div className="space-y-4">
          <div className="card !p-0 overflow-hidden">
            <div className="h-56 w-full bg-ink-800">
              {d.imageUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={d.imageUrl} alt={d.name} className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-6xl opacity-40">🍽️</div>
              )}
            </div>
            <div className="p-5">
              <div className="flex items-start justify-between gap-3">
                <h1 className="text-2xl font-bold">{d.name}</h1>
                <button onClick={() => favorite.mutate({ id: d.id, favorite: !d.isFavorite })} className="text-2xl">
                  <span className={d.isFavorite ? "text-bad" : "text-ink-400"}>{d.isFavorite ? "♥" : "♡"}</span>
                </button>
              </div>
              <p className="mt-1 text-sm text-ink-500">автор: {d.authorName || "аноним"}</p>
              {d.description ? <p className="mt-3 text-ink-300">{d.description}</p> : null}

              <div className="mt-4 grid grid-cols-4 gap-2 text-center">
                <Nutri label="Ккал" value={d.per100g.kcal} />
                <Nutri label="Белки" value={d.per100g.protein} />
                <Nutri label="Жиры" value={d.per100g.fat} />
                <Nutri label="Углев." value={d.per100g.carbs} />
              </div>
              <p className="mt-1 text-center text-xs text-ink-500">на 100 г{d.servingGrams ? ` · порция ${d.servingGrams} г` : ""}</p>

              {d.isMine ? (
                <div className="mt-4 flex gap-2">
                  <button onClick={onDelete} className="btn-ghost text-bad">
                    Удалить блюдо
                  </button>
                </div>
              ) : null}
            </div>
          </div>

          {d.recipe ? (
            <div className="card">
              <h2 className="mb-2 font-semibold">Рецепт</h2>
              <p className="whitespace-pre-wrap text-ink-300">{d.recipe}</p>
            </div>
          ) : null}
        </div>

        {/* right: rating, add-to-diet */}
        <div className="space-y-4">
          <div className="card">
            <h2 className="mb-2 font-semibold">Оценка</h2>
            <div className="flex items-center gap-3">
              <Stars value={d.myRating ?? 0} onRate={(r) => rate.mutate(r)} size="text-2xl" />
              <div className="text-sm text-ink-500">
                {d.ratingCount > 0 ? (
                  <>
                    средняя <span className="text-ink-100">{d.ratingAvg.toFixed(1)}</span> ({d.ratingCount})
                  </>
                ) : (
                  "оцените первым"
                )}
              </div>
            </div>
            {d.myRating ? <p className="mt-1 text-xs text-ink-500">ваша оценка: {d.myRating} ★</p> : null}
          </div>

          <div className="card space-y-3">
            <h2 className="font-semibold">Добавить в рацион</h2>
            <div className="flex gap-1 text-xs">
              <button onClick={() => setUnit("g")} className={`rounded-lg px-2 py-1 ${unit === "g" ? "bg-ink-700" : "bg-ink-800/50"}`}>
                Граммы
              </button>
              <button
                onClick={() => setUnit("serving")}
                disabled={!d.servingGrams}
                className={`rounded-lg px-2 py-1 disabled:opacity-40 ${unit === "serving" ? "bg-ink-700" : "bg-ink-800/50"}`}
              >
                Порции {d.servingGrams ? `(${d.servingGrams} г)` : "(нет)"}
              </button>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Field
                label={unit === "g" ? "Количество, г" : "Порций"}
                name="amount"
                inputMode="decimal"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
              />
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
                ≈ <span className="font-semibold text-ink-100">{preview.kcal} ккал</span> · Б {preview.protein} · Ж {preview.fat} · У{" "}
                {preview.carbs} <span className="text-ink-500">({Math.round(previewGrams)} г)</span>
              </div>
            ) : null}
            <button onClick={submitToDiet} disabled={addToDiet.isPending || previewGrams <= 0} className="btn-primary w-full">
              {added ? "Добавлено ✓" : "Добавить в рацион"}
            </button>
            {added ? (
              <Link href="/nutrition" className="block text-center text-sm text-brand hover:text-brand-soft">
                Открыть дневник питания →
              </Link>
            ) : null}
          </div>
        </div>
      </div>

      {/* comments */}
      <div className="card">
        <h2 className="mb-3 font-semibold">Комментарии</h2>
        <div className="mb-4 flex gap-2">
          <input
            className="input"
            placeholder="Оставьте отзыв…"
            value={commentBody}
            onChange={(e) => setCommentBody(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && commentBody.trim()) {
                addComment.mutate(commentBody.trim());
                setCommentBody("");
              }
            }}
          />
          <button
            onClick={() => {
              if (commentBody.trim()) {
                addComment.mutate(commentBody.trim());
                setCommentBody("");
              }
            }}
            className="btn-ghost shrink-0"
          >
            Отправить
          </button>
        </div>
        {comments.data && comments.data.items.length > 0 ? (
          <ul className="space-y-3">
            {comments.data.items.map((c) => (
              <li key={c.id} className="rounded-xl bg-ink-800/40 px-4 py-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">{c.authorName || "аноним"}</span>
                  <span className="text-xs text-ink-500">
                    {new Date(c.createdAt).toLocaleDateString("ru-RU")}
                    {c.isMine ? (
                      <button onClick={() => deleteComment.mutate(c.id)} className="ml-2 text-ink-500 hover:text-bad">
                        удалить
                      </button>
                    ) : null}
                  </span>
                </div>
                <p className="mt-1 text-ink-300">{c.body}</p>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-ink-500">Пока нет комментариев.</p>
        )}
      </div>
    </div>
  );
}

function Nutri({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl bg-ink-800/50 py-2">
      <div className="text-lg font-bold">{value}</div>
      <div className="text-xs text-ink-500">{label}</div>
    </div>
  );
}
