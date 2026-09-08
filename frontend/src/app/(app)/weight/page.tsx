"use client";

import { useMemo, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceArea,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  useAddWeightEntry,
  useDeleteWeightEntry,
  useSetWeightGoal,
  useWeightSummary,
} from "@/hooks/useWeight";

const BMI_LABELS: Record<string, { label: string; color: string }> = {
  underweight: { label: "Недостаток веса", color: "text-warn" },
  normal: { label: "Норма", color: "text-good" },
  overweight: { label: "Избыточный вес", color: "text-warn" },
  obese: { label: "Ожирение", color: "text-bad" },
};

export default function WeightPage() {
  const summary = useWeightSummary();
  const addEntry = useAddWeightEntry();
  const deleteEntry = useDeleteWeightEntry();
  const setGoal = useSetWeightGoal();

  const [weight, setWeight] = useState("");
  const [note, setNote] = useState("");
  const [goalInput, setGoalInput] = useState("");

  const data = summary.data;

  const chartData = useMemo(
    () =>
      (data?.series ?? []).map((e) => ({
        day: e.measuredOn.slice(5),
        kg: e.weightKg,
      })),
    [data],
  );

  function submitEntry() {
    const kg = parseFloat(weight.replace(",", "."));
    if (!Number.isFinite(kg) || kg <= 0) return;
    addEntry.mutate({ weightKg: kg, note: note.trim() || undefined });
    setWeight("");
    setNote("");
  }

  function saveGoal() {
    const kg = parseFloat(goalInput.replace(",", "."));
    if (!Number.isFinite(kg) || kg <= 0) return;
    setGoal.mutate(kg);
    setGoalInput("");
  }

  const bmiInfo = data?.bmiCategory ? BMI_LABELS[data.bmiCategory] : undefined;
  const latestKg = data?.latestKg ?? null;
  const targetKg = data?.targetKg ?? null;
  const toGoal = latestKg != null && targetKg != null ? +(latestKg - targetKg).toFixed(1) : null;

  // Short human description of the remaining distance to the goal weight.
  const goalText =
    toGoal == null
      ? null
      : toGoal === 0
        ? "цель достигнута 🎯"
        : toGoal > 0
          ? `осталось сбросить ${toGoal} кг`
          : `ниже цели на ${Math.abs(toGoal)} кг`;

  // Y-axis range that always includes the goal line, so the gap to it is
  // visible on the chart even when the goal sits outside the measured range.
  const yDomain = useMemo<[number, number]>(() => {
    const ys = chartData.map((d) => d.kg);
    if (targetKg != null) ys.push(targetKg);
    if (ys.length === 0) return [0, 100];
    return [Math.floor(Math.min(...ys) - 2), Math.ceil(Math.max(...ys) + 2)];
  }, [chartData, targetKg]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Вес</h1>
        <p className="text-sm text-ink-500">Отслеживайте динамику и индекс массы тела.</p>
      </div>

      {/* stat cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="card">
          <div className="text-sm text-ink-500">Текущий вес</div>
          <div className="mt-1 text-3xl font-black">{data?.latestKg != null ? `${data.latestKg} кг` : "—"}</div>
          {toGoal != null ? (
            <div className="mt-1 text-sm text-ink-500">
              {toGoal > 0 ? `до цели ${toGoal} кг` : toGoal < 0 ? `ниже цели на ${Math.abs(toGoal)} кг` : "цель достигнута 🎯"}
            </div>
          ) : null}
        </div>
        <div className="card">
          <div className="text-sm text-ink-500">Целевой вес</div>
          <div className="mt-1 text-3xl font-black">{data?.targetKg != null ? `${data.targetKg} кг` : "—"}</div>
        </div>
        <div className="card">
          <div className="text-sm text-ink-500">ИМТ</div>
          <div className="mt-1 text-3xl font-black">{data?.bmi != null ? data.bmi : "—"}</div>
          {bmiInfo ? (
            <div className={`mt-1 text-sm font-medium ${bmiInfo.color}`}>{bmiInfo.label}</div>
          ) : (
            <div className="mt-1 text-sm text-ink-500">укажите рост в настройках</div>
          )}
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-[1fr_1.4fr]">
        {/* inputs */}
        <div className="space-y-4">
          <div className="card space-y-3">
            <h2 className="font-semibold">Записать вес</h2>
            <input
              className="input"
              inputMode="decimal"
              placeholder="Вес, кг (напр. 74.5)"
              value={weight}
              onChange={(e) => setWeight(e.target.value)}
            />
            <input
              className="input"
              placeholder="Заметка (необязательно)"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <button onClick={submitEntry} disabled={addEntry.isPending} className="btn-primary w-full">
              Сохранить за сегодня
            </button>
            <p className="text-xs text-ink-500">Повторная запись за сегодня перезапишет предыдущую.</p>
          </div>

          <div className="card space-y-3">
            <h2 className="font-semibold">Целевой вес</h2>
            <div className="flex gap-2">
              <input
                className="input"
                inputMode="decimal"
                placeholder="кг"
                value={goalInput}
                onChange={(e) => setGoalInput(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && saveGoal()}
              />
              <button onClick={saveGoal} className="btn-ghost shrink-0">
                Задать
              </button>
            </div>
          </div>
        </div>

        {/* chart */}
        <div className="card">
          <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
            <h2 className="font-semibold">Динамика</h2>
            {goalText ? (
              <span
                className={`rounded-full px-2.5 py-1 text-xs font-medium ${
                  toGoal === 0 ? "bg-good/15 text-good" : "bg-brand/10 text-brand"
                }`}
              >
                {goalText}
              </span>
            ) : null}
          </div>
          {chartData.length > 0 ? (
            <div className="h-72 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData} margin={{ top: 5, right: 12, left: 4, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="day" tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                  <YAxis
                    width={52}
                    domain={yDomain}
                    tick={{ fill: "#64748b", fontSize: 12 }}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v: number) => `${v}`}
                    unit=" кг"
                  />
                  <Tooltip
                    contentStyle={{ background: "#0f172a", border: "1px solid #334155", borderRadius: 12, color: "#e2e8f0" }}
                    formatter={(v: number) => [`${v} кг`, "Вес"]}
                  />
                  {/* Shaded gap between the latest weight and the goal, so the
                      distance still to go reads at a glance. */}
                  {targetKg != null && latestKg != null && toGoal !== 0 ? (
                    <ReferenceArea
                      y1={Math.min(latestKg, targetKg)}
                      y2={Math.max(latestKg, targetKg)}
                      fill={toGoal! > 0 ? "#38bdf8" : "#f59e0b"}
                      fillOpacity={0.08}
                      stroke="none"
                    />
                  ) : null}
                  {targetKg != null ? (
                    <ReferenceLine
                      y={targetKg}
                      stroke="#34d399"
                      strokeDasharray="4 4"
                      label={{
                        value:
                          toGoal != null && toGoal > 0
                            ? `цель ${targetKg} кг · ${goalText}`
                            : `цель ${targetKg} кг`,
                        position: "insideBottomRight",
                        fill: "#34d399",
                        fontSize: 11,
                      }}
                    />
                  ) : null}
                  <Line type="monotone" dataKey="kg" stroke="#38bdf8" strokeWidth={2.5} dot={{ r: 3, fill: "#38bdf8" }} activeDot={{ r: 5 }} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <p className="text-sm text-ink-500">Пока нет измерений. Запишите первый вес слева.</p>
          )}

          {data && data.series.length > 0 ? (
            <ul className="mt-4 max-h-40 space-y-1 overflow-auto pr-1 text-sm">
              {[...data.series].reverse().map((e) => (
                <li key={e.id} className="flex items-center justify-between rounded-lg px-2 py-1 hover:bg-ink-800/50">
                  <span>
                    <span className="text-ink-500">{e.measuredOn}</span> — {e.weightKg} кг
                    {e.note ? <span className="ml-2 text-ink-500">· {e.note}</span> : null}
                  </span>
                  <button onClick={() => deleteEntry.mutate(e.id)} className="text-ink-500 hover:text-bad">
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </div>
    </div>
  );
}
