"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { ExerciseForm } from "@/components/training/ExerciseForm";
import { useExercise, useUpdateExercise } from "@/hooks/useTraining";

export default function EditExercisePage() {
  const { id } = useParams<{ id: string }>();
  const ex = useExercise(id);
  const update = useUpdateExercise(id);

  if (ex.isLoading) return <div className="card h-64 animate-pulse bg-ink-800/40" />;
  if (ex.isError || !ex.data)
    return (
      <div className="card text-center text-ink-500">
        Упражнение не найдено. <Link href="/exercises" className="text-brand">К каталогу</Link>
      </div>
    );
  if (!ex.data.isMine)
    return (
      <div className="card text-center text-ink-500">
        Редактировать можно только свои упражнения.{" "}
        <Link href={`/exercises/${id}`} className="text-brand">Назад</Link>
      </div>
    );

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">Редактирование упражнения</h1>
      <ExerciseForm initial={ex.data} submitLabel="Сохранить" onSubmit={(input) => update.mutateAsync(input)} />
    </div>
  );
}
