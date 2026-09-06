"use client";

import Link from "next/link";
import { useState } from "react";
import { Stars } from "@/components/Stars";
import { useExercises, useToggleExerciseFavorite, useTrainingMeta } from "@/hooks/useTraining";
import {
  CATEGORY_LABELS,
  DIFFICULTY_LABELS,
  DIFFICULTY_STYLE,
  EQUIPMENT_LABELS,
  label,
} from "@/lib/training";
import type { CatalogScope, CatalogSort, Exercise } from "@/lib/types";

const TABS: { key: CatalogScope; label: string }[] = [
  { key: "all", label: "Все упражнения" },
  { key: "mine", label: "Мои" },
  { key: "favorites", label: "Избранное" },
];

const SORTS: { key: CatalogSort; label: string }[] = [
  { key: "new", label: "Новые" },
  { key: "rating", label: "По рейтингу" },
  { key: "name", label: "По названию" },
];

export default function ExercisesPage() {
  const [scope, setScope] = useState<CatalogScope>("all");
  const [sort, setSort] = useState<CatalogSort>("new");
  const [q, setQ] = useState("");
  const [category, setCategory] = useState("");
  const [difficulty, setDifficulty] = useState("");
  const [equipment, setEquipment] = useState("");
  const [page, setPage] = useState(1);

  const meta = useTrainingMeta();
  const list = useExercises({ scope, q, sort, category, difficulty, equipment, page });
  const favorite = useToggleExerciseFavorite();

  const totalPages = list.data ? Math.max(1, Math.ceil(list.data.total / list.data.limit)) : 1;

  function resetPage<T>(setter: (v: T) => void) {
    return (v: T) => {
      setter(v);
      setPage(1);
    };
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">Упражнения</h1>
        <Link href="/exercises/new" className="btn-primary">
          + Создать упражнение
        </Link>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <div className="flex rounded-xl bg-ink-900 p-1">
          {TABS.map((t) => (
            <button
              key={t.key}
              onClick={() => resetPage(setScope)(t.key)}
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
          placeholder="Поиск…"
          value={q}
          onChange={(e) => resetPage(setQ)(e.target.value)}
        />
        <select className="input max-w-[11rem]" value={category} onChange={(e) => resetPage(setCategory)(e.target.value)}>
          <option value="">Все типы</option>
          {(meta.data?.categories ?? []).map((o) => (
            <option key={o.key} value={o.key}>
              {o.label}
            </option>
          ))}
        </select>
        <select className="input max-w-[11rem]" value={difficulty} onChange={(e) => resetPage(setDifficulty)(e.target.value)}>
          <option value="">Любая сложность</option>
          {(meta.data?.difficulties ?? []).map((o) => (
            <option key={o.key} value={o.key}>
              {o.label}
            </option>
          ))}
        </select>
        <select className="input max-w-[11rem]" value={equipment} onChange={(e) => resetPage(setEquipment)(e.target.value)}>
          <option value="">Любой инвентарь</option>
          {(meta.data?.equipment ?? []).map((o) => (
            <option key={o.key} value={o.key}>
              {o.label}
            </option>
          ))}
        </select>
        <select className="input max-w-[11rem]" value={sort} onChange={(e) => resetPage(setSort)(e.target.value as CatalogSort)}>
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
            <div key={i} className="card h-56 animate-pulse bg-ink-800/40" />
          ))}
        </div>
      ) : list.data && list.data.items.length > 0 ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {list.data.items.map((ex) => (
              <ExerciseCard key={ex.id} ex={ex} onToggleFav={() => favorite.mutate({ id: ex.id, favorite: !ex.isFavorite })} />
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
            ? "У вас пока нет своих упражнений. Создайте первое!"
            : scope === "favorites"
              ? "В избранном пусто. Добавляйте упражнения ♥ из каталога."
              : "Ничего не найдено."}
        </div>
      )}
    </div>
  );
}

function ExerciseCard({ ex, onToggleFav }: { ex: Exercise; onToggleFav: () => void }) {
  return (
    <div className="card group relative overflow-hidden !p-0">
      <button
        onClick={onToggleFav}
        className="absolute right-3 top-3 z-10 rounded-full bg-ink-950/70 px-2 py-1 text-lg backdrop-blur transition hover:scale-110"
        title={ex.isFavorite ? "Убрать из избранного" : "В избранное"}
      >
        <span className={ex.isFavorite ? "text-bad" : "text-ink-400"}>{ex.isFavorite ? "♥" : "♡"}</span>
      </button>
      <Link href={`/exercises/${ex.id}`}>
        <div className="h-36 w-full bg-ink-800">
          {ex.imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={ex.imageUrl} alt={ex.name} className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full items-center justify-center text-4xl opacity-40">🏋️</div>
          )}
        </div>
        <div className="p-4">
          <div className="flex items-start justify-between gap-2">
            <h3 className="font-semibold leading-tight">{ex.name}</h3>
            {ex.isMine ? <span className="rounded-full bg-brand/15 px-2 py-0.5 text-xs text-brand">моё</span> : null}
          </div>
          <div className="mt-2 flex flex-wrap gap-1.5">
            <span className="rounded-md bg-ink-800 px-2 py-0.5 text-xs text-ink-300">{label(CATEGORY_LABELS, ex.category)}</span>
            <span className={`rounded-md px-2 py-0.5 text-xs ${DIFFICULTY_STYLE[ex.difficulty] ?? "bg-ink-800 text-ink-300"}`}>
              {label(DIFFICULTY_LABELS, ex.difficulty)}
            </span>
          </div>
          {ex.equipment.length > 0 ? (
            <div className="mt-1.5 truncate text-xs text-ink-500">
              🧰 {ex.equipment.map((e) => label(EQUIPMENT_LABELS, e)).join(", ")}
            </div>
          ) : null}
          <div className="mt-2 flex items-center gap-2">
            <Stars value={Math.round(ex.ratingAvg)} size="text-sm" />
            <span className="text-xs text-ink-500">
              {ex.ratingCount > 0 ? `${ex.ratingAvg.toFixed(1)} (${ex.ratingCount})` : "нет оценок"}
            </span>
          </div>
        </div>
      </Link>
    </div>
  );
}
