"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { Stars } from "@/components/Stars";
import {
  useAddWorkoutComment,
  useDeleteWorkout,
  useDeleteWorkoutComment,
  usePublishWorkout,
  useRateWorkout,
  useToggleWorkoutFavorite,
  useWorkout,
  useWorkoutComments,
} from "@/hooks/useTraining";
import { PublishBadge, PublishControl } from "@/components/PublishControl";
import { CATEGORY_LABELS, DIFFICULTY_LABELS, DIFFICULTY_STYLE, label, prescription } from "@/lib/training";

export default function WorkoutDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();

  const wk = useWorkout(id);
  const comments = useWorkoutComments(id);
  const rate = useRateWorkout(id);
  const favorite = useToggleWorkoutFavorite();
  const addComment = useAddWorkoutComment(id);
  const deleteComment = useDeleteWorkoutComment(id);
  const remove = useDeleteWorkout();
  const publish = usePublishWorkout(id);

  const [commentBody, setCommentBody] = useState("");

  const w = wk.data;
  if (wk.isLoading) return <div className="card h-64 animate-pulse bg-ink-800/40" />;
  if (wk.isError || !w)
    return (
      <div className="card text-center text-ink-500">
        Тренировка не найдена. <Link href="/workouts" className="text-brand">К каталогу</Link>
      </div>
    );

  async function onDelete() {
    if (!confirm("Удалить эту тренировку?")) return;
    await remove.mutateAsync(id);
    router.push("/workouts");
  }

  function sendComment() {
    if (commentBody.trim()) {
      addComment.mutate(commentBody.trim());
      setCommentBody("");
    }
  }

  return (
    <div className="space-y-6">
      <Link href="/workouts" className="text-sm text-ink-500 hover:text-ink-100">
        ← Все тренировки
      </Link>

      <div className="card !p-0 overflow-hidden">
        <div className="h-48 w-full bg-ink-800">
          {w.imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={w.imageUrl} alt={w.name} className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full items-center justify-center text-6xl opacity-40">📋</div>
          )}
        </div>
        <div className="p-5">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h1 className="text-2xl font-bold">{w.name}</h1>
              <p className="mt-1 text-sm text-ink-500">автор: {w.authorName || "аноним"}</p>
              {w.isMine ? (
                <div className="mt-2">
                  <PublishBadge isPublic={w.isPublic} />
                </div>
              ) : null}
            </div>
            <button onClick={() => favorite.mutate({ id: w.id, favorite: !w.isFavorite })} className="text-2xl">
              <span className={w.isFavorite ? "text-bad" : "text-ink-400"}>{w.isFavorite ? "♥" : "♡"}</span>
            </button>
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <span className={`rounded-md px-2 py-1 text-xs ${DIFFICULTY_STYLE[w.difficulty] ?? "bg-ink-800 text-ink-300"}`}>
              {label(DIFFICULTY_LABELS, w.difficulty)}
            </span>
            <span className="rounded-md bg-ink-800 px-2 py-1 text-xs text-ink-300">{w.exerciseCount} упражнений</span>
            <div className="flex items-center gap-2">
              <Stars value={w.myRating ?? 0} onRate={(r) => rate.mutate(r)} size="text-lg" />
              <span className="text-xs text-ink-500">
                {w.ratingCount > 0 ? `${w.ratingAvg.toFixed(1)} (${w.ratingCount})` : "оцените первым"}
              </span>
            </div>
          </div>
          {w.description ? <p className="mt-3 whitespace-pre-wrap text-ink-300">{w.description}</p> : null}
          {w.isMine ? (
            <div className="mt-4 flex flex-wrap gap-2">
              <PublishControl isPublic={w.isPublic} pending={publish.isPending} onToggle={(p) => publish.mutate(p)} />
              <Link href={`/workouts/${w.id}/edit`} className="btn-ghost">
                Редактировать
              </Link>
              <button onClick={onDelete} className="btn-ghost text-bad">
                Удалить
              </button>
            </div>
          ) : null}
        </div>
      </div>

      {/* ordered exercise list */}
      <div className="card">
        <h2 className="mb-3 font-semibold">Программа</h2>
        <ol className="space-y-3">
          {(w.items ?? []).map((it) => {
            const meta = prescription(it);
            return (
              <li key={it.id} className="flex items-start gap-3 rounded-xl bg-ink-800/40 p-3">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-ink-700 text-sm font-bold">
                  {it.position}
                </div>
                <div className="h-12 w-12 shrink-0 overflow-hidden rounded-lg bg-ink-800">
                  {it.exercise.imageUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={it.exercise.imageUrl} alt="" className="h-full w-full object-cover" />
                  ) : (
                    <div className="flex h-full items-center justify-center opacity-40">🏋️</div>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <Link href={`/exercises/${it.exerciseId}`} className="font-medium hover:text-brand">
                    {it.exercise.name}
                  </Link>
                  <p className="text-xs text-ink-500">
                    {label(CATEGORY_LABELS, it.exercise.category)} · {label(DIFFICULTY_LABELS, it.exercise.difficulty)}
                  </p>
                  {meta ? <p className="mt-1 text-sm text-ink-300">{meta}</p> : null}
                  {it.note ? <p className="mt-0.5 text-sm text-ink-500">{it.note}</p> : null}
                </div>
              </li>
            );
          })}
        </ol>
      </div>

      {/* comments */}
      <div className="card">
        <h2 className="mb-3 font-semibold">Комментарии</h2>
        <div className="mb-4 flex gap-2">
          <input
            className="input"
            placeholder="Оставьте отзыв…"
            value={commentBody}
            onChange={(ev) => setCommentBody(ev.target.value)}
            onKeyDown={(ev) => {
              if (ev.key === "Enter") sendComment();
            }}
          />
          <button onClick={sendComment} className="btn-ghost shrink-0">
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
