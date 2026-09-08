"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { DishForm } from "@/components/nutrition/DishForm";
import { useDish, useUpdateDish } from "@/hooks/useDishes";

export default function EditDishPage() {
  const { id } = useParams<{ id: string }>();
  const dish = useDish(id);
  const update = useUpdateDish(id);

  if (dish.isLoading) return <div className="card h-64 animate-pulse bg-ink-800/40" />;
  if (dish.isError || !dish.data)
    return (
      <div className="card text-center text-ink-500">
        Блюдо не найдено. <Link href="/dishes" className="text-brand">К каталогу</Link>
      </div>
    );
  if (!dish.data.isMine)
    return (
      <div className="card text-center text-ink-500">
        Редактировать можно только свои блюда.{" "}
        <Link href={`/dishes/${id}`} className="text-brand">Назад</Link>
      </div>
    );

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">Редактирование блюда</h1>
      <DishForm initial={dish.data} submitLabel="Сохранить" onSubmit={(input) => update.mutateAsync(input)} />
    </div>
  );
}
