"use client";

import { WorkoutBuilder } from "@/components/training/WorkoutBuilder";
import { useCreateWorkout } from "@/hooks/useTraining";

export default function NewWorkoutPage() {
  const create = useCreateWorkout();
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Новая тренировка</h1>
        <p className="text-sm text-ink-500">Соберите программу из упражнений и задайте подходы/повторы.</p>
      </div>
      <WorkoutBuilder submitLabel="Создать тренировку" onSubmit={(input) => create.mutateAsync(input)} />
    </div>
  );
}
