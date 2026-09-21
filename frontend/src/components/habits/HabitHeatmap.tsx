"use client";

import { useMemo } from "react";
import { useClearCheckin, useHabitCheckins, useSetCheckin } from "@/hooks/useHabits";

const WEEKS = 10;
const WD = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];

function pad(n: number): string {
  return String(n).padStart(2, "0");
}
function ymd(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
function addDays(d: Date, n: number): Date {
  const x = new Date(d);
  x.setDate(x.getDate() + n);
  return x;
}

// HabitHeatmap is the day-by-day tracker: a calendar grid of the last ~10 weeks.
// Tapping a day cycles unmarked → kept (green) → missed (red) → unmarked.
export function HabitHeatmap({ habitId }: { habitId: string }) {
  const { grid, from, to, todayKey } = useMemo(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const dow = (today.getDay() + 6) % 7; // 0 = Monday
    const monday = addDays(today, -dow);
    const start = addDays(monday, -7 * (WEEKS - 1));
    const cells: Date[] = [];
    for (let i = 0; i < WEEKS * 7; i++) cells.push(addDays(start, i));
    return { grid: cells, from: ymd(start), to: ymd(addDays(monday, 6)), todayKey: ymd(today) };
  }, []);

  const checkins = useHabitCheckins(habitId, from, to);
  const set = useSetCheckin(habitId);
  const clear = useClearCheckin(habitId);
  const days = checkins.data?.days ?? {};

  function cycle(key: string) {
    if (!(key in days)) set.mutate({ day: key, success: true });
    else if (days[key]) set.mutate({ day: key, success: false });
    else clear.mutate(key);
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3 text-xs text-ink-500">
        <span className="inline-flex items-center gap-1"><span className="h-3 w-3 rounded-sm bg-good" /> выполнено</span>
        <span className="inline-flex items-center gap-1"><span className="h-3 w-3 rounded-sm bg-bad" /> пропущено</span>
        <span className="inline-flex items-center gap-1"><span className="h-3 w-3 rounded-sm bg-ink-800" /> нет отметки</span>
        <span className="ml-auto">тап по дню меняет статус</span>
      </div>
      <div className="grid grid-cols-7 gap-1">
        {WD.map((d) => (
          <div key={d} className="pb-0.5 text-center text-[11px] text-ink-500">{d}</div>
        ))}
        {grid.map((d) => {
          const key = ymd(d);
          const future = key > todayKey;
          const marked = key in days;
          const success = marked && days[key];
          const cls = future
            ? "opacity-0"
            : !marked
              ? "bg-ink-800 hover:bg-ink-700"
              : success
                ? "bg-good hover:brightness-110"
                : "bg-bad hover:brightness-110";
          return (
            <button
              key={key}
              disabled={future}
              onClick={() => cycle(key)}
              title={key}
              className={`grid aspect-square place-items-center rounded-md text-[11px] font-medium transition ${cls} ${
                key === todayKey ? "ring-2 ring-brand" : ""
              } ${marked ? "text-ink-950" : "text-ink-500"}`}
            >
              {future ? "" : d.getDate()}
            </button>
          );
        })}
      </div>
    </div>
  );
}
