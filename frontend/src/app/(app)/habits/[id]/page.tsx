"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Check, Flame, Leaf, Pencil, ShieldAlert, Trash2, X } from "lucide-react";
import { todayISO } from "@/lib/api";
import { useDeleteHabit, useHabits, useSetCheckin, useUpdateHabit } from "@/hooks/useHabits";
import { HabitHeatmap } from "@/components/habits/HabitHeatmap";
import { HabitReminders } from "@/components/habits/HabitReminders";
import { HabitDiary } from "@/components/habits/HabitDiary";
import type { HabitColor } from "@/lib/types";

const DOT: Record<HabitColor, string> = { green: "bg-good", yellow: "bg-warn", red: "bg-bad" };

export default function HabitTrackerPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const router = useRouter();
  const habits = useHabits();
  const set = useSetCheckin(id);
  const update = useUpdateHabit();
  const del = useDeleteHabit();

  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [desc, setDesc] = useState("");

  const habit = habits.data?.items.find((h) => h.id === id);

  if (!habits.isLoading && !habit) {
    return (
      <div className="space-y-4">
        <Link href="/habits" className="btn-ghost !py-1.5 text-sm"><ArrowLeft size={15} /> К привычкам</Link>
        <p className="card text-center text-ink-500">Привычка не найдена.</p>
      </div>
    );
  }
  if (!habit) {
    return <div className="card h-40 animate-pulse bg-ink-800/40" />;
  }

  const good = habit.kind === "good";
  const Icon = good ? Leaf : ShieldAlert;
  const today = todayISO();

  function startEdit() {
    setName(habit!.name);
    setDesc(habit!.description);
    setEditing(true);
  }
  function saveEdit() {
    update.mutate({ id, name: name.trim() || habit!.name, description: desc }, { onSuccess: () => setEditing(false) });
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-2">
        <Link href="/habits" className="btn-ghost !py-1.5 text-sm"><ArrowLeft size={15} /> К привычкам</Link>
        <div className="flex items-center gap-2">
          <button onClick={startEdit} className="btn-ghost !py-1.5 text-sm"><Pencil size={14} /> Изменить</button>
          <button
            onClick={() => {
              if (confirm(`Удалить привычку «${habit.name}»?`)) del.mutate(id, { onSuccess: () => router.push("/habits") });
            }}
            className="btn-ghost !py-1.5 text-sm text-bad"
          >
            <Trash2 size={14} /> Удалить
          </button>
        </div>
      </div>

      {editing ? (
        <div className="card space-y-2">
          <input className="input" value={name} maxLength={140} onChange={(e) => setName(e.target.value)} placeholder="Название" />
          <textarea className="input min-h-16" value={desc} maxLength={2000} onChange={(e) => setDesc(e.target.value)} placeholder="Описание (необязательно)" />
          <div className="flex items-center gap-2">
            <button onClick={saveEdit} disabled={update.isPending} className="btn-primary">Сохранить</button>
            <button onClick={() => setEditing(false)} className="btn-ghost">Отмена</button>
          </div>
        </div>
      ) : (
        <div>
          <div className="flex items-center gap-2">
            <span className={`h-3 w-3 rounded-full ${DOT[habit.color]}`} title={habit.color} />
            <Icon size={20} className={good ? "text-good" : "text-bad"} />
            <h1 className="text-2xl font-bold">{habit.name}</h1>
            {habit.streak > 0 ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-brand/15 px-2 py-0.5 text-sm text-brand"><Flame size={13} /> {habit.streak} дней</span>
            ) : null}
          </div>
          <p className="text-sm text-ink-500">
            {good ? "Полезная привычка" : "Вредная привычка"}
            {habit.description ? ` · ${habit.description}` : ""}
          </p>
        </div>
      )}

      {/* today quick-mark */}
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-ink-400">Сегодня:</span>
        <button onClick={() => set.mutate({ day: today, success: true })} className="inline-flex items-center gap-1 rounded-lg bg-good/15 px-3 py-1.5 text-sm text-good transition hover:bg-good/25">
          <Check size={15} /> {good ? "Выполнил" : "Удержался"}
        </button>
        <button onClick={() => set.mutate({ day: today, success: false })} className="inline-flex items-center gap-1 rounded-lg bg-bad/15 px-3 py-1.5 text-sm text-bad transition hover:bg-bad/25">
          <X size={15} /> {good ? "Пропустил" : "Сорвался"}
        </button>
      </div>

      {/* tracker */}
      <div className="card space-y-3">
        <h2 className="font-semibold">Трекер</h2>
        <HabitHeatmap habitId={id} />
      </div>

      <div className="card"><HabitReminders habitId={id} /></div>
      <div className="card"><HabitDiary habitId={id} kind={habit.kind} /></div>
    </div>
  );
}
