"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Field } from "@/components/Field";
import { isoDaysAgo, todayISO } from "@/lib/api";
import {
  useActivityTypes,
  useAddActivity,
  useAddManualEntry,
  useDeleteActivity,
  useDeleteDietEntry,
  useNutritionDay,
  useNutritionHistory,
  useSetNutritionGoal,
} from "@/hooks/useNutrition";
import type { Meal } from "@/lib/types";

const MEALS: { value: Meal; label: string }[] = [
  { value: "breakfast", label: "Завтрак" },
  { value: "lunch", label: "Обед" },
  { value: "dinner", label: "Ужин" },
  { value: "snack", label: "Перекус" },
];

function num(v: string): number {
  const n = parseFloat(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

function hhmm(iso: string): string {
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

interface MacroSlice {
  name: string;
  grams: number;
  kcal: number;
  pct: number;
  color: string;
}

// energySplit converts grams of protein/fat/carbs into their share of calories
// (protein & carbs 4 kcal/g, fat 9 kcal/g) — the basis for the balance donut.
function energySplit(protein: number, fat: number, carbs: number): { total: number; slices: MacroSlice[] } {
  const p = protein * 4;
  const f = fat * 9;
  const c = carbs * 4;
  const total = p + f + c;
  const pct = (x: number) => (total > 0 ? Math.round((x / total) * 100) : 0);
  return {
    total,
    slices: [
      { name: "Белки", grams: protein, kcal: Math.round(p), pct: pct(p), color: "#34d399" },
      { name: "Жиры", grams: fat, kcal: Math.round(f), pct: pct(f), color: "#fbbf24" },
      { name: "Углеводы", grams: carbs, kcal: Math.round(c), pct: pct(c), color: "#38bdf8" },
    ],
  };
}

export default function NutritionPage() {
  const day = useNutritionDay();
  const history = useNutritionHistory(isoDaysAgo(13), todayISO());
  const activityTypes = useActivityTypes();
  const addManual = useAddManualEntry();
  const deleteEntry = useDeleteDietEntry();
  const addActivity = useAddActivity();
  const deleteActivity = useDeleteActivity();
  const setGoal = useSetNutritionGoal();

  const d = day.data;

  // cumulative macro timeline
  const timeline = useMemo(() => {
    if (!d) return [];
    const pts = [...d.entries].sort((a, b) => a.consumedAt.localeCompare(b.consumedAt));
    let kcal = 0,
      protein = 0,
      fat = 0,
      carbs = 0;
    const rows = pts.map((e) => {
      kcal += e.macros.kcal;
      protein += e.macros.protein;
      fat += e.macros.fat;
      carbs += e.macros.carbs;
      return { t: hhmm(e.consumedAt), kcal: Math.round(kcal), protein: +protein.toFixed(1), fat: +fat.toFixed(1), carbs: +carbs.toFixed(1) };
    });
    return rows;
  }, [d]);

  const historyData = useMemo(
    () =>
      (history.data?.series ?? []).map((r) => ({
        day: r.date.slice(5),
        eaten: r.kcal,
        burned: r.burnedKcal,
        net: Math.round(r.kcal - r.burnedKcal),
      })),
    [history.data],
  );

  // Today's vs goal macro split by ENERGY (kcal): protein/carbs = 4, fat = 9.
  const macroSplit = useMemo(() => (d ? energySplit(d.consumed.protein, d.consumed.fat, d.consumed.carbs) : null), [d]);
  const goalSplit = useMemo(() => (d ? energySplit(d.goal.protein, d.goal.fat, d.goal.carbs) : null), [d]);

  // 14-day macro history + averages over days that actually have data.
  const macroHistory = useMemo(
    () => (history.data?.series ?? []).map((r) => ({ day: r.date.slice(5), protein: r.protein, fat: r.fat, carbs: r.carbs })),
    [history.data],
  );
  const macroAvg = useMemo(() => {
    const rows = (history.data?.series ?? []).filter((r) => r.kcal > 0 || r.protein > 0 || r.fat > 0 || r.carbs > 0);
    if (rows.length === 0) return { protein: 0, fat: 0, carbs: 0 };
    const sum = rows.reduce((a, r) => ({ protein: a.protein + r.protein, fat: a.fat + r.fat, carbs: a.carbs + r.carbs }), { protein: 0, fat: 0, carbs: 0 });
    return { protein: +(sum.protein / rows.length).toFixed(1), fat: +(sum.fat / rows.length).toFixed(1), carbs: +(sum.carbs / rows.length).toFixed(1) };
  }, [history.data]);

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-bold">Калории и рацион</h1>
          <p className="text-sm text-ink-500">Сегодня, {d?.date ?? "…"}</p>
        </div>
        <Link href="/dishes" className="btn-ghost">
          🍽️ Выбрать блюдо
        </Link>
      </div>

      {d ? (
        <>
          {/* balance */}
          <div className="grid gap-4 sm:grid-cols-4">
            <MacroCard label="Калории" value={d.consumed.kcal} goal={d.goal.kcal} unit="ккал" accent="brand" />
            <MacroCard label="Белки" value={d.consumed.protein} goal={d.goal.protein} unit="г" accent="good" />
            <MacroCard label="Жиры" value={d.consumed.fat} goal={d.goal.fat} unit="г" accent="warn" />
            <MacroCard label="Углеводы" value={d.consumed.carbs} goal={d.goal.carbs} unit="г" accent="brand" />
          </div>

          <div className="card flex flex-wrap items-center justify-between gap-4">
            <Balance label="Съедено" value={`${Math.round(d.consumed.kcal)} ккал`} />
            <Balance label="Сожжено" value={`− ${Math.round(d.burnedKcal)} ккал`} muted />
            <Balance label="Итого (нетто)" value={`${Math.round(d.netKcal)} ккал`} />
            <div className="text-right">
              <div className="text-sm text-ink-500">{d.remainingKcal >= 0 ? "Осталось до цели" : "Профицит"}</div>
              <div className={`text-2xl font-black ${d.remainingKcal >= 0 ? "text-good" : "text-bad"}`}>
                {d.remainingKcal >= 0 ? "" : "+"}
                {Math.abs(Math.round(d.remainingKcal))} ккал
              </div>
            </div>
          </div>

          {/* charts */}
          <div className="grid gap-6 lg:grid-cols-2">
            <div className="card">
              <h2 className="mb-3 font-semibold">Калории по времени</h2>
              {timeline.length > 0 ? (
                <div className="h-56">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={timeline} margin={{ top: 5, right: 12, left: 4, bottom: 0 }}>
                      <defs>
                        <linearGradient id="kcalFill" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="#38bdf8" stopOpacity={0.5} />
                          <stop offset="100%" stopColor="#38bdf8" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                      <XAxis dataKey="t" tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                      <YAxis width={44} tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                      <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => [`${v} ккал`, "Накоплено"]} />
                      <ReferenceLine y={d.goal.kcal} stroke="#34d399" strokeDasharray="4 4" />
                      <Area type="monotone" dataKey="kcal" stroke="#38bdf8" strokeWidth={2.5} fill="url(#kcalFill)" />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <p className="text-sm text-ink-500">Пока ничего не добавлено сегодня.</p>
              )}
            </div>

            <div className="card">
              <h2 className="mb-3 font-semibold">БЖУ по времени</h2>
              {timeline.length > 0 ? (
                <div className="h-56">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={timeline} margin={{ top: 5, right: 12, left: 4, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                      <XAxis dataKey="t" tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                      <YAxis width={44} unit=" г" tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                      <Tooltip contentStyle={tooltipStyle} />
                      <Line type="monotone" dataKey="protein" name="Белки" stroke="#34d399" strokeWidth={2} dot={false} />
                      <Line type="monotone" dataKey="fat" name="Жиры" stroke="#fbbf24" strokeWidth={2} dot={false} />
                      <Line type="monotone" dataKey="carbs" name="Углеводы" stroke="#38bdf8" strokeWidth={2} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <p className="text-sm text-ink-500">Добавьте приёмы пищи, чтобы увидеть динамику.</p>
              )}
            </div>
          </div>

          {/* macro balance infographic */}
          <div className="card">
            <div className="mb-1 flex flex-wrap items-center justify-between gap-2">
              <h2 className="font-semibold">Баланс БЖУ</h2>
              <span className="text-sm text-ink-500">доля калорий из белков / жиров / углеводов</span>
            </div>
            <div className="grid gap-6 sm:grid-cols-2">
              {macroSplit && macroSplit.total > 0 ? (
                <MacroDonut title="Сегодня" split={macroSplit} centerLabel={`${Math.round(macroSplit.total)}`} centerHint="ккал из БЖУ" />
              ) : (
                <div className="flex h-56 items-center justify-center text-sm text-ink-500">
                  Добавьте приёмы пищи, чтобы увидеть баланс.
                </div>
              )}
              {goalSplit && goalSplit.total > 0 ? (
                <MacroDonut title="Ваша цель" split={goalSplit} centerLabel={`${Math.round(goalSplit.total)}`} centerHint="ккал из БЖУ" muted />
              ) : (
                <div className="flex h-56 items-center justify-center text-sm text-ink-500">
                  Задайте цели по БЖУ ниже.
                </div>
              )}
            </div>
          </div>

          {/* inputs */}
          <div className="grid gap-6 lg:grid-cols-2">
            <ManualFoodForm onAdd={(v) => addManual.mutate(v)} pending={addManual.isPending} />
            <ActivityForm
              types={activityTypes.data?.types ?? []}
              onAdd={(v) => addActivity.mutate(v)}
              pending={addActivity.isPending}
            />
          </div>

          {/* lists */}
          <div className="grid gap-6 lg:grid-cols-2">
            <div className="card">
              <h2 className="mb-3 font-semibold">Приёмы пищи</h2>
              {d.entries.length > 0 ? (
                <ul className="space-y-2">
                  {[...d.entries].reverse().map((e) => (
                    <li key={e.id} className="flex items-center justify-between rounded-xl bg-ink-800/50 px-3 py-2 text-sm">
                      <div>
                        <div className="font-medium">
                          {e.name} {e.grams ? <span className="text-ink-500">· {e.grams} г</span> : null}
                        </div>
                        <div className="text-ink-500">
                          {Math.round(e.macros.kcal)} ккал · Б {e.macros.protein} · Ж {e.macros.fat} · У {e.macros.carbs} ·{" "}
                          {hhmm(e.consumedAt)}
                        </div>
                      </div>
                      <button onClick={() => deleteEntry.mutate(e.id)} className="text-ink-500 hover:text-bad">
                        ✕
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-ink-500">Ещё ничего не добавлено.</p>
              )}
            </div>

            <div className="card">
              <h2 className="mb-3 font-semibold">Активности</h2>
              {d.activities.length > 0 ? (
                <ul className="space-y-2">
                  {[...d.activities].reverse().map((a) => (
                    <li key={a.id} className="flex items-center justify-between rounded-xl bg-ink-800/50 px-3 py-2 text-sm">
                      <div>
                        <div className="font-medium">
                          {a.type} {a.durationMin ? <span className="text-ink-500">· {a.durationMin} мин</span> : null}
                        </div>
                        <div className="text-ink-500">
                          − {Math.round(a.kcal)} ккал · {hhmm(a.performedAt)} {a.source === "met" ? "· MET" : ""}
                        </div>
                      </div>
                      <button onClick={() => deleteActivity.mutate(a.id)} className="text-ink-500 hover:text-bad">
                        ✕
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-ink-500">Добавьте тренировку или прогулку, чтобы «сжечь» калории.</p>
              )}
            </div>
          </div>

          {/* goal + history */}
          <GoalEditor
            goal={d.goal}
            onSave={(g) => setGoal.mutate(g)}
            pending={setGoal.isPending}
          />

          <div className="card">
            <h2 className="mb-4 font-semibold">Баланс за 14 дней</h2>
            <div className="h-56">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={historyData} margin={{ top: 5, right: 12, left: 4, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="day" tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                  <YAxis width={44} tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                  <Tooltip contentStyle={tooltipStyle} />
                  <ReferenceLine y={d.goal.kcal} stroke="#34d399" strokeDasharray="4 4" />
                  <Line type="monotone" dataKey="eaten" name="Съедено" stroke="#38bdf8" strokeWidth={2} dot={false} />
                  <Line type="monotone" dataKey="net" name="Нетто" stroke="#fbbf24" strokeWidth={2} dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>

          {/* macro history */}
          <div className="card">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
              <h2 className="font-semibold">БЖУ за 14 дней</h2>
              <span className="text-sm text-ink-500">
                среднее в день: <span className="text-good">Б {macroAvg.protein} г</span> ·{" "}
                <span className="text-warn">Ж {macroAvg.fat} г</span> ·{" "}
                <span className="text-brand">У {macroAvg.carbs} г</span>
              </span>
            </div>
            <div className="grid gap-4 sm:grid-cols-3">
              <MacroTrend data={macroHistory} dataKey="protein" label="Белки" color="#34d399" goal={d.goal.protein} avg={macroAvg.protein} />
              <MacroTrend data={macroHistory} dataKey="fat" label="Жиры" color="#fbbf24" goal={d.goal.fat} avg={macroAvg.fat} />
              <MacroTrend data={macroHistory} dataKey="carbs" label="Углеводы" color="#38bdf8" goal={d.goal.carbs} avg={macroAvg.carbs} />
            </div>
          </div>
        </>
      ) : (
        <div className="card h-40 animate-pulse bg-ink-800/40" />
      )}
    </div>
  );
}

const tooltipStyle = { background: "#0f172a", border: "1px solid #334155", borderRadius: 12, color: "#e2e8f0" };

function MacroCard({ label, value, goal, unit, accent }: { label: string; value: number; goal: number; unit: string; accent: string }) {
  const pct = goal > 0 ? Math.min(100, (value / goal) * 100) : 0;
  const bar = accent === "good" ? "bg-good" : accent === "warn" ? "bg-warn" : "bg-brand";
  return (
    <div className="card">
      <div className="text-sm text-ink-500">{label}</div>
      <div className="mt-1 text-2xl font-black">
        {Math.round(value)}
        <span className="text-base font-medium text-ink-500">
          {" "}
          / {Math.round(goal)} {unit}
        </span>
      </div>
      <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-ink-800">
        <div className={`h-full rounded-full ${bar} transition-all`} style={{ width: `${pct}%` }} />
      </div>
      <div className="mt-1 text-xs text-ink-500">{goal > 0 ? `${Math.round(pct)}% цели` : "цель не задана"}</div>
    </div>
  );
}

function MacroDonut({
  title,
  split,
  centerLabel,
  centerHint,
  muted,
}: {
  title: string;
  split: { total: number; slices: MacroSlice[] };
  centerLabel: string;
  centerHint: string;
  muted?: boolean;
}) {
  return (
    <div>
      <div className="mb-1 text-center text-sm font-medium text-ink-300">{title}</div>
      <div className="relative h-44">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={split.slices}
              dataKey="kcal"
              nameKey="name"
              innerRadius={52}
              outerRadius={72}
              paddingAngle={2}
              stroke="none"
              opacity={muted ? 0.75 : 1}
            >
              {split.slices.map((s) => (
                <Cell key={s.name} fill={s.color} />
              ))}
            </Pie>
            <Tooltip contentStyle={tooltipStyle} formatter={(v: number, n: string) => [`${v} ккал`, n]} />
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <div className="text-xl font-black">{centerLabel}</div>
          <div className="text-[11px] text-ink-500">{centerHint}</div>
        </div>
      </div>
      <div className="mt-2 space-y-1">
        {split.slices.map((s) => (
          <div key={s.name} className="flex items-center justify-between text-sm">
            <span className="flex items-center gap-2">
              <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ background: s.color }} />
              {s.name}
            </span>
            <span className="text-ink-400">
              {s.grams} г · {s.pct}%
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function MacroTrend({
  data,
  dataKey,
  label,
  color,
  goal,
  avg,
}: {
  data: { day: string; protein: number; fat: number; carbs: number }[];
  dataKey: "protein" | "fat" | "carbs";
  label: string;
  color: string;
  goal: number;
  avg: number;
}) {
  return (
    <div className="rounded-xl border border-ink-800 bg-ink-950/40 p-3">
      <div className="mb-1 flex items-baseline justify-between">
        <span className="text-sm font-medium">{label}</span>
        <span className="text-xs text-ink-500">{goal > 0 ? `цель ${Math.round(goal)} г` : "цель —"}</span>
      </div>
      <div className="h-28">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, left: -18, bottom: 0 }}>
            <defs>
              <linearGradient id={`grad-${dataKey}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.45} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
            <XAxis dataKey="day" tick={{ fill: "#64748b", fontSize: 10 }} tickLine={false} axisLine={false} interval="preserveStartEnd" />
            <YAxis width={26} tick={{ fill: "#64748b", fontSize: 10 }} tickLine={false} axisLine={false} />
            <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => [`${v} г`, label]} />
            {goal > 0 ? <ReferenceLine y={goal} stroke="#64748b" strokeDasharray="4 4" /> : null}
            <Area type="monotone" dataKey={dataKey} stroke={color} strokeWidth={2} fill={`url(#grad-${dataKey})`} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <div className="mt-1 text-center text-xs text-ink-500">среднее {avg} г/день</div>
    </div>
  );
}

function Balance({ label, value, muted }: { label: string; value: string; muted?: boolean }) {
  return (
    <div>
      <div className="text-sm text-ink-500">{label}</div>
      <div className={`text-xl font-bold ${muted ? "text-ink-300" : ""}`}>{value}</div>
    </div>
  );
}

function ManualFoodForm({ onAdd, pending }: { onAdd: (v: { name: string; kcal: number; protein: number; fat: number; carbs: number; meal?: Meal }) => void; pending: boolean }) {
  const [f, setF] = useState({ name: "", kcal: "", protein: "", fat: "", carbs: "" });
  const [meal, setMeal] = useState<Meal | "">("");

  function submit() {
    if (!f.name.trim() || num(f.kcal) <= 0) return;
    onAdd({ name: f.name.trim(), kcal: num(f.kcal), protein: num(f.protein), fat: num(f.fat), carbs: num(f.carbs), meal: meal || undefined });
    setF({ name: "", kcal: "", protein: "", fat: "", carbs: "" });
  }

  return (
    <div className="card space-y-3">
      <h2 className="font-semibold">Добавить вручную</h2>
      <Field label="Название" name="name" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} />
      <div className="grid grid-cols-4 gap-2">
        <Field label="Ккал" name="kcal" inputMode="decimal" value={f.kcal} onChange={(e) => setF({ ...f, kcal: e.target.value })} />
        <Field label="Б" name="protein" inputMode="decimal" value={f.protein} onChange={(e) => setF({ ...f, protein: e.target.value })} />
        <Field label="Ж" name="fat" inputMode="decimal" value={f.fat} onChange={(e) => setF({ ...f, fat: e.target.value })} />
        <Field label="У" name="carbs" inputMode="decimal" value={f.carbs} onChange={(e) => setF({ ...f, carbs: e.target.value })} />
      </div>
      <div className="flex items-end gap-2">
        <div className="flex-1">
          <label className="label">Приём пищи</label>
          <select className="input" value={meal} onChange={(e) => setMeal(e.target.value as Meal | "")}>
            <option value="">—</option>
            {MEALS.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </div>
        <button onClick={submit} disabled={pending} className="btn-primary">
          Добавить
        </button>
      </div>
    </div>
  );
}

function ActivityForm({
  types,
  onAdd,
  pending,
}: {
  types: { key: string; label: string; met: number }[];
  onAdd: (v: { type?: string; met?: number; durationMin?: number; kcal?: number }) => void;
  pending: boolean;
}) {
  const [mode, setMode] = useState<"met" | "manual">("met");
  const [type, setType] = useState("");
  const [duration, setDuration] = useState("");
  const [kcal, setKcal] = useState("");

  function submit() {
    if (mode === "met") {
      const t = types.find((x) => x.key === type);
      if (!t || num(duration) <= 0) return;
      onAdd({ type: t.label, met: t.met, durationMin: Math.round(num(duration)) });
      setDuration("");
    } else {
      if (num(kcal) <= 0) return;
      onAdd({ type: type ? types.find((x) => x.key === type)?.label ?? "Активность" : "Активность", kcal: num(kcal), durationMin: duration ? Math.round(num(duration)) : undefined });
      setKcal("");
      setDuration("");
    }
  }

  return (
    <div className="card space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold">Сжечь калории</h2>
        <div className="flex gap-1 text-xs">
          <button onClick={() => setMode("met")} className={`rounded-lg px-2 py-1 ${mode === "met" ? "bg-ink-700" : "bg-ink-800/50"}`}>
            По времени (MET)
          </button>
          <button onClick={() => setMode("manual")} className={`rounded-lg px-2 py-1 ${mode === "manual" ? "bg-ink-700" : "bg-ink-800/50"}`}>
            Вручную (ккал)
          </button>
        </div>
      </div>

      <div>
        <label className="label">Активность</label>
        <select className="input" value={type} onChange={(e) => setType(e.target.value)}>
          <option value="">Выберите…</option>
          {types.map((t) => (
            <option key={t.key} value={t.key}>
              {t.label} (MET {t.met})
            </option>
          ))}
        </select>
      </div>

      {mode === "met" ? (
        <Field label="Длительность, мин" name="duration" inputMode="numeric" value={duration} onChange={(e) => setDuration(e.target.value)} hint="Ккал считаются из MET и вашего веса" />
      ) : (
        <div className="grid grid-cols-2 gap-2">
          <Field label="Сожжено, ккал" name="kcal" inputMode="numeric" value={kcal} onChange={(e) => setKcal(e.target.value)} />
          <Field label="Мин (необяз.)" name="dur" inputMode="numeric" value={duration} onChange={(e) => setDuration(e.target.value)} />
        </div>
      )}
      <button onClick={submit} disabled={pending} className="btn-ghost w-full">
        Добавить активность
      </button>
    </div>
  );
}

function GoalEditor({ goal, onSave, pending }: { goal: { kcal: number; protein: number; fat: number; carbs: number }; onSave: (g: { kcal: number; protein: number; fat: number; carbs: number }) => void; pending: boolean }) {
  const [open, setOpen] = useState(false);
  const [g, setG] = useState({ kcal: String(goal.kcal), protein: String(goal.protein), fat: String(goal.fat), carbs: String(goal.carbs) });

  return (
    <div className="card">
      <button onClick={() => setOpen((o) => !o)} className="flex w-full items-center justify-between font-semibold">
        <span>🎯 Цели по КБЖУ</span>
        <span className="text-sm text-ink-500">
          {goal.kcal} ккал · Б{goal.protein} Ж{goal.fat} У{goal.carbs} {open ? "▲" : "▼"}
        </span>
      </button>
      {open ? (
        <div className="mt-4 grid grid-cols-2 items-end gap-3 sm:grid-cols-5">
          <Field label="Ккал" name="gkcal" inputMode="numeric" value={g.kcal} onChange={(e) => setG({ ...g, kcal: e.target.value })} />
          <Field label="Белки" name="gp" inputMode="decimal" value={g.protein} onChange={(e) => setG({ ...g, protein: e.target.value })} />
          <Field label="Жиры" name="gf" inputMode="decimal" value={g.fat} onChange={(e) => setG({ ...g, fat: e.target.value })} />
          <Field label="Углеводы" name="gc" inputMode="decimal" value={g.carbs} onChange={(e) => setG({ ...g, carbs: e.target.value })} />
          <button
            onClick={() => onSave({ kcal: Math.round(num(g.kcal)), protein: num(g.protein), fat: num(g.fat), carbs: num(g.carbs) })}
            disabled={pending}
            className="btn-primary"
          >
            Сохранить
          </button>
        </div>
      ) : null}
    </div>
  );
}
