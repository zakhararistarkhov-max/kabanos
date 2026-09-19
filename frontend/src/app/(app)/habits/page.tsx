"use client";

import { useState } from "react";
import { Bell, Leaf, Plus, ShieldAlert } from "lucide-react";
import { usePush } from "@/hooks/usePush";
import { useCreateHabit, useHabits } from "@/hooks/useHabits";
import { HabitCard } from "@/components/habits/HabitCard";
import type { Habit, HabitKind } from "@/lib/types";

export default function HabitsPage() {
  const habits = useHabits();
  const push = usePush();
  const items = habits.data?.items ?? [];
  const good = items.filter((h) => h.kind === "good");
  const bad = items.filter((h) => h.kind === "bad");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Привычки</h1>
        <p className="text-sm text-ink-500">Формируйте полезные привычки и избавляйтесь от вредных: напоминания и мини‑дневник по каждой.</p>
      </div>

      {!push.loading && push.supported && push.secure && push.configured && !push.subscribed ? (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-warn/40 bg-warn/10 px-3 py-2 text-sm">
          <span className="text-warn">Чтобы напоминания приходили, включите уведомления в этом браузере.</span>
          <button onClick={push.enable} disabled={push.loading} className="btn-primary !py-1.5">
            <Bell size={15} /> Включить уведомления
          </button>
        </div>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-2">
        <HabitColumn
          kind="good"
          title="Полезные привычки"
          hint="Хочу развить"
          Icon={Leaf}
          accent="text-good"
          list={good}
          loading={habits.isLoading}
        />
        <HabitColumn
          kind="bad"
          title="Вредные привычки"
          hint="Хочу бросить"
          Icon={ShieldAlert}
          accent="text-bad"
          list={bad}
          loading={habits.isLoading}
        />
      </div>
    </div>
  );
}

function HabitColumn({
  kind,
  title,
  hint,
  Icon,
  accent,
  list,
  loading,
}: {
  kind: HabitKind;
  title: string;
  hint: string;
  Icon: typeof Leaf;
  accent: string;
  list: Habit[];
  loading: boolean;
}) {
  const create = useCreateHabit();
  const [name, setName] = useState("");

  function add() {
    const t = name.trim();
    if (!t) return;
    create.mutate({ name: t, kind }, { onSuccess: () => setName("") });
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Icon size={18} className={accent} />
        <h2 className="font-semibold">{title}</h2>
        <span className="text-xs text-ink-500">{hint}</span>
      </div>

      <div className="flex gap-2">
        <input
          className="input"
          placeholder={kind === "good" ? "Напр. «Читать 20 минут»" : "Напр. «Скроллить ленту в кровати»"}
          value={name}
          maxLength={140}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              add();
            }
          }}
        />
        <button onClick={add} disabled={create.isPending || !name.trim()} className="btn-primary shrink-0">
          <Plus size={16} /> Добавить
        </button>
      </div>

      {loading ? (
        <div className="h-16 animate-pulse rounded-2xl bg-ink-800/40" />
      ) : list.length > 0 ? (
        <div className="space-y-2">
          {list.map((h) => (
            <HabitCard key={h.id} habit={h} />
          ))}
        </div>
      ) : (
        <p className="rounded-2xl border border-dashed border-ink-800 px-3 py-6 text-center text-sm text-ink-500">
          {kind === "good" ? "Добавьте первую полезную привычку." : "Добавьте привычку, от которой хотите избавиться."}
        </p>
      )}
    </div>
  );
}
