"use client";

import { ExerciseForm } from "@/components/training/ExerciseForm";
import { useCreateExercise } from "@/hooks/useTraining";

export default function NewExercisePage() {
  const create = useCreateExercise();
  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Новое упражнение</h1>
        <p className="text-sm text-ink-500">Оно появится в общем каталоге и в разделе «Мои».</p>
      </div>
      <ExerciseForm submitLabel="Создать упражнение" onSubmit={(input) => create.mutateAsync(input)} />
    </div>
  );
}
