"use client";

import Link from "next/link";
import { useState } from "react";
import { Stars } from "@/components/Stars";
import { useDishes, useToggleFavorite } from "@/hooks/useDishes";
import type { Dish, DishScope, DishSort } from "@/lib/types";

const TABS: { key: DishScope; label: string }[] = [
  { key: "all", label: "Все блюда" },
  { key: "mine", label: "Мои" },
  { key: "favorites", label: "Избранное" },
];

const SORTS: { key: DishSort; label: string }[] = [
  { key: "new", label: "Новые" },
  { key: "rating", label: "По рейтингу" },
  { key: "name", label: "По названию" },
];

export default function DishesPage() {
  const [scope, setScope] = useState<DishScope>("all");
  const [sort, setSort] = useState<DishSort>("new");
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);

  const list = useDishes({ scope, q, sort, page });
  const favorite = useToggleFavorite();

  const totalPages = list.data ? Math.max(1, Math.ceil(list.data.total / list.data.limit)) : 1;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">Блюда</h1>
        <Link href="/dishes/new" className="btn-primary">
          + Создать блюдо
        </Link>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <div className="flex rounded-xl bg-ink-900 p-1">
          {TABS.map((t) => (
            <button
              key={t.key}
              onClick={() => {
                setScope(t.key);
                setPage(1);
              }}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                scope === t.key ? "bg-ink-700 text-ink-100" : "text-ink-400 hover:text-ink-100"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
        <input
          className="input max-w-xs"
          placeholder="Поиск по названию…"
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setPage(1);
          }}
        />
        <select
          className="input max-w-[12rem]"
          value={sort}
          onChange={(e) => {
            setSort(e.target.value as DishSort);
            setPage(1);
          }}
        >
          {SORTS.map((s) => (
            <option key={s.key} value={s.key}>
              {s.label}
            </option>
          ))}
        </select>
      </div>

      {list.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="card h-64 animate-pulse bg-ink-800/40" />
          ))}
        </div>
      ) : list.data && list.data.items.length > 0 ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {list.data.items.map((dish) => (
              <DishCard key={dish.id} dish={dish} onToggleFav={() => favorite.mutate({ id: dish.id, favorite: !dish.isFavorite })} />
            ))}
          </div>
          {totalPages > 1 ? (
            <div className="flex items-center justify-center gap-3 text-sm">
              <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="btn-ghost !py-1.5">
                ← Назад
              </button>
              <span className="text-ink-500">
                {page} / {totalPages}
              </span>
              <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="btn-ghost !py-1.5">
                Вперёд →
              </button>
            </div>
          ) : null}
        </>
      ) : (
        <div className="card text-center text-ink-500">
          {scope === "mine"
            ? "У вас пока нет своих блюд. Создайте первое!"
            : scope === "favorites"
              ? "В избранном пусто. Добавляйте блюда ♥ из общего каталога."
              : "Ничего не найдено."}
        </div>
      )}
    </div>
  );
}

function DishCard({ dish, onToggleFav }: { dish: Dish; onToggleFav: () => void }) {
  return (
    <div className="card group relative overflow-hidden !p-0">
      <button
        onClick={onToggleFav}
        className="absolute right-3 top-3 z-10 rounded-full bg-ink-950/70 px-2 py-1 text-lg backdrop-blur transition hover:scale-110"
        title={dish.isFavorite ? "Убрать из избранного" : "В избранное"}
      >
        <span className={dish.isFavorite ? "text-bad" : "text-ink-400"}>{dish.isFavorite ? "♥" : "♡"}</span>
      </button>
      <Link href={`/dishes/${dish.id}`}>
        <div className="h-36 w-full bg-ink-800">
          {dish.imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={dish.imageUrl} alt={dish.name} className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full items-center justify-center text-4xl opacity-40">🍽️</div>
          )}
        </div>
        <div className="p-4">
          <div className="flex items-start justify-between gap-2">
            <h3 className="font-semibold leading-tight">{dish.name}</h3>
            {dish.isMine ? <span className="rounded-full bg-brand/15 px-2 py-0.5 text-xs text-brand">моё</span> : null}
          </div>
          <div className="mt-1 text-sm text-ink-500">
            {Math.round(dish.per100g.kcal)} ккал / 100 г · Б {dish.per100g.protein} Ж {dish.per100g.fat} У {dish.per100g.carbs}
          </div>
          <div className="mt-2 flex items-center gap-2">
            <Stars value={Math.round(dish.ratingAvg)} size="text-sm" />
            <span className="text-xs text-ink-500">
              {dish.ratingCount > 0 ? `${dish.ratingAvg.toFixed(1)} (${dish.ratingCount})` : "нет оценок"}
            </span>
          </div>
          <div className="mt-1 text-xs text-ink-500">автор: {dish.authorName || "аноним"}</div>
        </div>
      </Link>
    </div>
  );
}
