"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { Stars } from "@/components/Stars";
import {
  useAddExerciseComment,
  useDeleteExercise,
  useDeleteExerciseComment,
  useExercise,
  useExerciseComments,
  useRateExercise,
  useToggleExerciseFavorite,
} from "@/hooks/useTraining";
import {
  CATEGORY_LABELS,
  DIFFICULTY_LABELS,
  DIFFICULTY_STYLE,
  EQUIPMENT_LABELS,
  JOINT_LABELS,
  MUSCLE_LABELS,
  label,
} from "@/lib/training";

export default function ExerciseDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();

  const ex = useExercise(id);
  const comments = useExerciseComments(id);
  const rate = useRateExercise(id);
  const favorite = useToggleExerciseFavorite();
  const addComment = useAddExerciseComment(id);
  const deleteComment = useDeleteExerciseComment(id);
  const remove = useDeleteExercise();

  const [commentBody, setCommentBody] = useState("");

  const e = ex.data;
  if (ex.isLoading) return <div className="card h-64 animate-pulse bg-ink-800/40" />;
  if (ex.isError || !e)
    return (
      <div className="card text-center text-ink-500">
        Упражнение не найдено. <Link href="/exercises" className="text-brand">К каталогу</Link>
      </div>
    );

  async function onDelete() {
    if (!confirm("Удалить это упражнение? Оно исчезнет из тренировок, где используется.")) return;
    await remove.mutateAsync(id);
    router.push("/exercises");
  }

  function sendComment() {
    if (commentBody.trim()) {
      addComment.mutate(commentBody.trim());
      setCommentBody("");
    }
  }

  return (
    <div className="space-y-6">
      <Link href="/exercises" className="text-sm text-ink-500 hover:text-ink-100">
        ← Все упражнения
      </Link>

      <div className="grid gap-6 md:grid-cols-[1.2fr_1fr]">
        <div className="space-y-4">
          <div className="card !p-0 overflow-hidden">
            <div className="h-56 w-full bg-ink-800">
              {e.imageUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={e.imageUrl} alt={e.name} className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-6xl opacity-40">🏋️</div>
              )}
            </div>
            <div className="p-5">
              <div className="flex items-start justify-between gap-3">
                <h1 className="text-2xl font-bold">{e.name}</h1>
                <button onClick={() => favorite.mutate({ id: e.id, favorite: !e.isFavorite })} className="text-2xl">
                  <span className={e.isFavorite ? "text-bad" : "text-ink-400"}>{e.isFavorite ? "♥" : "♡"}</span>
                </button>
              </div>
              <p className="mt-1 text-sm text-ink-500">автор: {e.authorName || "аноним"}</p>

              <div className="mt-3 flex flex-wrap gap-2">
                <span className="rounded-md bg-ink-800 px-2 py-1 text-xs text-ink-300">{label(CATEGORY_LABELS, e.category)}</span>
                <span className={`rounded-md px-2 py-1 text-xs ${DIFFICULTY_STYLE[e.difficulty] ?? "bg-ink-800 text-ink-300"}`}>
                  {label(DIFFICULTY_LABELS, e.difficulty)}
                </span>
                <span className="rounded-md bg-ink-800 px-2 py-1 text-xs text-ink-300">{label(JOINT_LABELS, e.jointImpact)}</span>
              </div>

              {e.equipment.length > 0 ? (
                <p className="mt-3 text-sm text-ink-300">
                  <span className="text-ink-500">Инвентарь:</span> {e.equipment.map((k) => label(EQUIPMENT_LABELS, k)).join(", ")}
                </p>
              ) : null}
              {e.muscles.length > 0 ? (
                <p className="mt-1 text-sm text-ink-300">
                  <span className="text-ink-500">Мышцы:</span> {e.muscles.map((k) => label(MUSCLE_LABELS, k)).join(", ")}
                </p>
              ) : null}
              {e.description ? <p className="mt-3 whitespace-pre-wrap text-ink-300">{e.description}</p> : null}
              {e.videoUrl ? (
                <a href={e.videoUrl} target="_blank" rel="noopener noreferrer" className="mt-3 inline-block text-brand hover:text-brand-soft">
                  ▶ Смотреть видео
                </a>
              ) : null}

              {e.isMine ? (
                <div className="mt-4 flex gap-2">
                  <Link href={`/exercises/${e.id}/edit`} className="btn-ghost">
                    Редактировать
                  </Link>
                  <button onClick={onDelete} className="btn-ghost text-bad">
                    Удалить
                  </button>
                </div>
              ) : null}
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="card">
            <h2 className="mb-2 font-semibold">Оценка</h2>
            <div className="flex items-center gap-3">
              <Stars value={e.myRating ?? 0} onRate={(r) => rate.mutate(r)} size="text-2xl" />
              <div className="text-sm text-ink-500">
                {e.ratingCount > 0 ? (
                  <>
                    средняя <span className="text-ink-100">{e.ratingAvg.toFixed(1)}</span> ({e.ratingCount})
                  </>
                ) : (
                  "оцените первым"
                )}
              </div>
            </div>
            {e.myRating ? <p className="mt-1 text-xs text-ink-500">ваша оценка: {e.myRating} ★</p> : null}
          </div>

          <div className="card">
            <h2 className="mb-2 font-semibold">Использовать</h2>
            <p className="text-sm text-ink-500">
              Добавьте это упражнение в тренировку в конструкторе.
            </p>
            <Link href="/workouts/new" className="btn-ghost mt-3 inline-block">
              + Составить тренировку
            </Link>
          </div>
        </div>
      </div>

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
