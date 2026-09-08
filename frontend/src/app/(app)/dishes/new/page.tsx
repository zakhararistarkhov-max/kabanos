"use client";

import { DishForm } from "@/components/nutrition/DishForm";
import { useCreateDish } from "@/hooks/useDishes";

export default function NewDishPage() {
  const create = useCreateDish();

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Новое блюдо</h1>
        <p className="text-sm text-ink-500">
          Укажите КБЖУ вручную (на 100 г) — или соберите блюдо из ингредиентов, и КБЖУ посчитается само.
        </p>
      </div>

      <DishForm submitLabel="Создать блюдо" onSubmit={(input) => create.mutateAsync(input)} />
    </div>
  );
}
