"use client";

import Link from "next/link";
import { useState } from "react";
import { Stars } from "@/components/Stars";
import { useTrainingMeta, useWorkouts, useToggleWorkoutFavorite } from "@/hooks/useTraining";
import { DIFFICULTY_LABELS, DIFFICULTY_STYLE, label } from "@/lib/training";
import type { CatalogScope, CatalogSort, Workout } from "@/lib/types";

const TABS: { key: CatalogScope; label: string }[] = [
  { key: "all", label: "Все тренировки" },
  { key: "mine", label: "Мои" },
  { key: "favorites", label: "Избранное" },
];

const SORTS: { key: CatalogSort; label: string }[] = [
  { key: "new", label: "Новые" },
  { key: "rating", label: "По рейтингу" },
  { key: "name", label: "По названию" },
];

export default function WorkoutsPage() {
  const [scope, setScope] = useState<CatalogScope>("all");
  const [sort, setSort] = useState<CatalogSort>("new");
  const [q, setQ] = useState("");
  const [difficulty, setDifficulty] = useState("");
  const [page, setPage] = useState(1);

  const meta = useTrainingMeta();
  const list = useWorkouts({ scope, q, sort, difficulty, page });
  const favorite = useToggleWorkoutFavorite();

  const totalPages = list.data ? Math.max(1, Math.ceil(list.data.total / list.data.limit)) : 1;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">Тренировки</h1>
        <Link href="/workouts/new" className="btn-primary">
          + Составить тренировку
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
          placeholder="Поиск…"
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setPage(1);
          }}
        />
        <select
          className="input max-w-[12rem]"
          value={difficulty}
          onChange={(e) => {
            setDifficulty(e.target.value);
            setPage(1);
          }}
        >
          <option value="">Любая сложность</option>
          {(meta.data?.difficulties ?? []).map((o) => (
            <option key={o.key} value={o.key}>
              {o.label}
            </option>
          ))}
        </select>
        <select
          className="input max-w-[12rem]"
          value={sort}
          onChange={(e) => {
            setSort(e.target.value as CatalogSort);
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
            <div key={i} className="card h-56 animate-pulse bg-ink-800/40" />
          ))}
        </div>
      ) : list.data && list.data.items.length > 0 ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {list.data.items.map((wk) => (
              <WorkoutCard key={wk.id} wk={wk} onToggleFav={() => favorite.mutate({ id: wk.id, favorite: !wk.isFavorite })} />
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
            ? "У вас пока нет своих тренировок. Составьте первую!"
            : scope === "favorites"
              ? "В избранном пусто. Добавляйте тренировки ♥ из каталога."
              : "Ничего не найдено."}
        </div>
      )}
    </div>
  );
}

function WorkoutCard({ wk, onToggleFav }: { wk: Workout; onToggleFav: () => void }) {
  return (
    <div className="card group relative overflow-hidden !p-0">
      <button
        onClick={onToggleFav}
        className="absolute right-3 top-3 z-10 rounded-full bg-ink-950/70 px-2 py-1 text-lg backdrop-blur transition hover:scale-110"
        title={wk.isFavorite ? "Убрать из избранного" : "В избранное"}
      >
        <span className={wk.isFavorite ? "text-bad" : "text-ink-400"}>{wk.isFavorite ? "♥" : "♡"}</span>
      </button>
      <Link href={`/workouts/${wk.id}`}>
        <div className="h-36 w-full bg-ink-800">
          {wk.imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={wk.imageUrl} alt={wk.name} className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full items-center justify-center text-4xl opacity-40">📋</div>
          )}
        </div>
        <div className="p-4">
          <div className="flex items-start justify-between gap-2">
            <h3 className="font-semibold leading-tight">{wk.name}</h3>
            {wk.isMine ? (
              wk.isPublic ? (
                <span className="shrink-0 rounded-full bg-brand/15 px-2 py-0.5 text-xs text-brand">моё</span>
              ) : (
                <span className="shrink-0 rounded-full bg-warn/15 px-2 py-0.5 text-xs text-warn">черновик</span>
              )
            ) : null}
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            <span className={`rounded-md px-2 py-0.5 text-xs ${DIFFICULTY_STYLE[wk.difficulty] ?? "bg-ink-800 text-ink-300"}`}>
              {label(DIFFICULTY_LABELS, wk.difficulty)}
            </span>
            <span className="rounded-md bg-ink-800 px-2 py-0.5 text-xs text-ink-300">{wk.exerciseCount} упр.</span>
          </div>
          <div className="mt-2 flex items-center gap-2">
            <Stars value={Math.round(wk.ratingAvg)} size="text-sm" />
            <span className="text-xs text-ink-500">
              {wk.ratingCount > 0 ? `${wk.ratingAvg.toFixed(1)} (${wk.ratingCount})` : "нет оценок"}
            </span>
          </div>
          <div className="mt-1 text-xs text-ink-500">автор: {wk.authorName || "аноним"}</div>
        </div>
      </Link>
    </div>
  );
}
