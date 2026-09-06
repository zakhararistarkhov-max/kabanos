"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { WorkoutBuilder } from "@/components/training/WorkoutBuilder";
import { useUpdateWorkout, useWorkout } from "@/hooks/useTraining";

export default function EditWorkoutPage() {
  const { id } = useParams<{ id: string }>();
  const wk = useWorkout(id);
  const update = useUpdateWorkout(id);

  if (wk.isLoading) return <div className="card h-64 animate-pulse bg-ink-800/40" />;
  if (wk.isError || !wk.data)
    return (
      <div className="card text-center text-ink-500">
        Тренировка не найдена. <Link href="/workouts" className="text-brand">К каталогу</Link>
      </div>
    );
  if (!wk.data.isMine)
    return (
      <div className="card text-center text-ink-500">
        Редактировать можно только свои тренировки.{" "}
        <Link href={`/workouts/${id}`} className="text-brand">Назад</Link>
      </div>
    );

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <h1 className="text-2xl font-bold">Редактирование тренировки</h1>
      <WorkoutBuilder initial={wk.data} submitLabel="Сохранить" onSubmit={(input) => update.mutateAsync(input)} />
    </div>
  );
}
