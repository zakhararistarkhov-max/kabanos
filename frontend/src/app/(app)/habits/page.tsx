"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Bell, Check, ChevronRight, Flame, Leaf, Plus, ShieldAlert, X } from "lucide-react";
import { todayISO } from "@/lib/api";
import { usePush } from "@/hooks/usePush";
import { useCreateHabit, useHabits, useSetCheckin } from "@/hooks/useHabits";
import { HabitStrip } from "@/components/habits/HabitStrip";
import type { Habit, HabitColor, HabitKind } from "@/lib/types";

const BORDER: Record<HabitColor, string> = {
  green: "border-l-good",
  yellow: "border-l-warn",
  red: "border-l-bad",
};
const HEALTH_LABEL: Record<HabitColor, string> = {
  green: "всё хорошо",
  yellow: "стоит подтянуть",
  red: "срывается",
};

export default function HabitsPage() {
  const habits = useHabits();
  const push = usePush();
  const items = habits.data?.items ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Привычки</h1>
        <p className="text-sm text-ink-500">Дашборд привычек: то, что срывается, — красным вверху; отмечайте дни и открывайте трекер.</p>
      </div>

      {!push.loading && push.supported && push.secure && push.configured && !push.subscribed ? (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-warn/40 bg-warn/10 px-3 py-2 text-sm">
          <span className="text-warn">Чтобы напоминания приходили, включите уведомления в этом браузере.</span>
          <button onClick={push.enable} disabled={push.loading} className="btn-primary !py-1.5">
            <Bell size={15} /> Включить уведомления
          </button>
        </div>
      ) : null}

      <AddHabit />

      {habits.isLoading ? (
        <div className="space-y-2">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded-2xl bg-ink-800/40" />
          ))}
        </div>
      ) : items.length === 0 ? (
        <p className="rounded-2xl border border-dashed border-ink-800 px-3 py-8 text-center text-sm text-ink-500">
          Пока нет привычек. Добавьте первую выше.
        </p>
      ) : (
        <div className="space-y-2.5">
          {items.map((h) => (
            <DashCard key={h.id} habit={h} />
          ))}
        </div>
      )}
    </div>
  );
}

function AddHabit() {
  const create = useCreateHabit();
  const [name, setName] = useState("");
  const [kind, setKind] = useState<HabitKind>("good");

  function add() {
    const t = name.trim();
    if (!t) return;
    create.mutate({ name: t, kind }, { onSuccess: () => setName("") });
  }

  return (
    <div className="card space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex overflow-hidden rounded-lg border border-ink-800">
          <button
            onClick={() => setKind("good")}
            className={`flex items-center gap-1 px-2.5 py-1.5 text-sm transition ${kind === "good" ? "bg-good/20 text-good" : "text-ink-400 hover:bg-ink-800"}`}
          >
            <Leaf size={14} /> Полезная
          </button>
          <button
            onClick={() => setKind("bad")}
            className={`flex items-center gap-1 px-2.5 py-1.5 text-sm transition ${kind === "bad" ? "bg-bad/20 text-bad" : "text-ink-400 hover:bg-ink-800"}`}
          >
            <ShieldAlert size={14} /> Вредная
          </button>
        </div>
        <input
          className="input min-w-40 flex-1"
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
    </div>
  );
}

function DashCard({ habit }: { habit: Habit }) {
  const router = useRouter();
  const set = useSetCheckin(habit.id);
  const good = habit.kind === "good";
  const Icon = good ? Leaf : ShieldAlert;
  const today = todayISO();

  const mark = (success: boolean) => set.mutate({ day: today, success });

  return (
    <div
      onClick={() => router.push(`/habits/${habit.id}`)}
      className={`cursor-pointer rounded-2xl border border-ink-800 border-l-4 bg-ink-950/40 p-3.5 transition hover:border-brand/40 ${BORDER[habit.color]}`}
    >
      <div className="flex items-center gap-2">
        <Icon size={16} className={good ? "text-good/80" : "text-bad/80"} />
        <span className="min-w-0 flex-1 truncate font-semibold">{habit.name}</span>
        {habit.streak > 0 ? (
          <span className="inline-flex items-center gap-1 rounded-full bg-brand/15 px-2 py-0.5 text-xs text-brand">
            <Flame size={12} /> {habit.streak}
          </span>
        ) : null}
        {/* today quick-mark */}
        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
          <button onClick={() => mark(true)} className="grid h-7 w-7 place-items-center rounded-lg bg-good/15 text-good transition hover:bg-good/25" title="Отметить сегодня выполненным">
            <Check size={15} />
          </button>
          <button onClick={() => mark(false)} className="grid h-7 w-7 place-items-center rounded-lg bg-bad/15 text-bad transition hover:bg-bad/25" title="Отметить сегодня пропущенным">
            <X size={15} />
          </button>
        </div>
        <ChevronRight size={16} className="text-ink-600" />
      </div>

      <div className="mt-2.5">
        <HabitStrip recent={habit.recent} />
      </div>

      <div className="mt-1.5 flex items-center justify-between text-xs text-ink-500">
        <span>за 2 недели: {habit.successes} ✓ · {habit.fails} ✗</span>
        <span className={habit.color === "red" ? "text-bad" : habit.color === "yellow" ? "text-warn" : "text-good"}>{HEALTH_LABEL[habit.color]}</span>
      </div>
    </div>
  );
}
