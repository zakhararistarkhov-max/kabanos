"use client";

import { useMemo, useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { WaterContainer } from "@/components/WaterContainer";
import { isoDaysAgo, todayISO } from "@/lib/api";
import {
  useAddIntake,
  useDeleteIntake,
  useSetWaterGoal,
  useWaterDay,
  useWaterHistory,
} from "@/hooks/useWater";
import type { WaterSource } from "@/lib/types";

const PRESETS: { source: WaterSource; label: string; amountMl: number; emoji: string }[] = [
  { source: "glass", label: "Стакан", amountMl: 250, emoji: "🥛" },
  { source: "bottle_small", label: "Бутылка", amountMl: 500, emoji: "🧴" },
  { source: "bottle_large", label: "Бутылка", amountMl: 1000, emoji: "🍶" },
];

export default function WaterPage() {
  const day = useWaterDay();
  const history = useWaterHistory(isoDaysAgo(13), todayISO());
  const addIntake = useAddIntake();
  const deleteIntake = useDeleteIntake();
  const setGoal = useSetWaterGoal();

  const [custom, setCustom] = useState("");
  const [goalInput, setGoalInput] = useState("");

  const chartData = useMemo(
    () =>
      (history.data?.series ?? []).map((d) => ({
        day: d.date.slice(5), // MM-DD
        liters: +(d.totalMl / 1000).toFixed(2),
      })),
    [history.data],
  );
  const goalLiters = history.data ? history.data.goalMl / 1000 : 2;

  function addCustom() {
    const ml = Math.round(parseFloat(custom.replace(",", ".")) * 1000);
    if (!Number.isFinite(ml) || ml <= 0) return;
    addIntake.mutate({ amountMl: ml, source: "custom" });
    setCustom("");
  }

  function saveGoal() {
    const ml = Math.round(parseFloat(goalInput.replace(",", ".")) * 1000);
    if (!Number.isFinite(ml) || ml <= 0) return;
    setGoal.mutate(ml);
    setGoalInput("");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-bold">Вода</h1>
          <p className="text-sm text-ink-500">Наливайте воду и следите за дневной нормой.</p>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* container + actions */}
        <div className="card">
          {day.isLoading ? (
            <div className="h-80 animate-pulse rounded-2xl bg-ink-800/50" />
          ) : day.data ? (
            <WaterContainer consumedMl={day.data.consumedMl} goalMl={day.data.goalMl} percent={day.data.percent} />
          ) : (
            <p className="text-bad">Не удалось загрузить данные.</p>
          )}

          <div className="mt-6 grid grid-cols-3 gap-2">
            {PRESETS.map((p) => (
              <button
                key={p.source + p.amountMl}
                onClick={() => addIntake.mutate({ amountMl: p.amountMl, source: p.source })}
                disabled={addIntake.isPending}
                className="btn-ghost flex-col !py-3 text-center"
              >
                <span className="text-2xl">{p.emoji}</span>
                <span className="text-xs">
                  {p.label} {p.amountMl / 1000} л
                </span>
              </button>
            ))}
          </div>

          <div className="mt-3 flex gap-2">
            <input
              className="input"
              inputMode="decimal"
              placeholder="Своё, л (напр. 0.33)"
              value={custom}
              onChange={(e) => setCustom(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && addCustom()}
            />
            <button onClick={addCustom} className="btn-primary shrink-0">
              Добавить
            </button>
          </div>

          <div className="mt-3 flex items-center gap-2 text-sm text-ink-500">
            <span>Дневная цель:</span>
            <input
              className="input !py-1.5 max-w-[8rem]"
              inputMode="decimal"
              placeholder={`${(day.data?.goalMl ?? 2000) / 1000} л`}
              value={goalInput}
              onChange={(e) => setGoalInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && saveGoal()}
            />
            <button onClick={saveGoal} className="text-brand hover:text-brand-soft">
              Сохранить
            </button>
          </div>
        </div>

        {/* today's log */}
        <div className="card">
          <h2 className="mb-3 font-semibold">Сегодня</h2>
          {day.data && day.data.intakes.length > 0 ? (
            <ul className="max-h-80 space-y-2 overflow-auto pr-1">
              {[...day.data.intakes].reverse().map((it) => (
                <li key={it.id} className="flex items-center justify-between rounded-xl bg-ink-800/50 px-3 py-2">
                  <span className="text-sm">
                    {(it.amountMl / 1000).toFixed(2)} л
                    <span className="ml-2 text-ink-500">
                      {new Date(it.consumedAt).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" })}
                    </span>
                  </span>
                  <button onClick={() => deleteIntake.mutate(it.id)} className="text-ink-500 hover:text-bad" title="Удалить">
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-ink-500">Пока ничего не выпито. Начните со стакана воды 🥛</p>
          )}
        </div>
      </div>

      {/* history */}
      <div className="card">
        <h2 className="mb-4 font-semibold">Последние 14 дней</h2>
        <div className="h-56 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} margin={{ top: 5, right: 12, left: 4, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
              <XAxis dataKey="day" tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
              <YAxis width={42} tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} unit=" л" />
              <Tooltip
                contentStyle={{ background: "#0f172a", border: "1px solid #334155", borderRadius: 12, color: "#e2e8f0" }}
                formatter={(v: number) => [`${v} л`, "Выпито"]}
              />
              <ReferenceLine y={goalLiters} stroke="#34d399" strokeDasharray="4 4" />
              <Bar dataKey="liters" fill="#38bdf8" radius={[6, 6, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
