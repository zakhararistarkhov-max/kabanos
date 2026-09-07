"use client";

import { useMemo, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { useAddPressure, useDeletePressure, usePressureSummary } from "@/hooks/useBloodPressure";
import { pressureCat } from "@/lib/pressure";

function num(v: string): number {
  const n = parseInt(v.replace(/[^0-9]/g, ""), 10);
  return Number.isFinite(n) ? n : 0;
}

function shortTime(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getDate()).padStart(2, "0")}.${String(d.getMonth() + 1).padStart(2, "0")} ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

export default function PressurePage() {
  const summary = usePressureSummary();
  const add = useAddPressure();
  const del = useDeletePressure();

  const [sys, setSys] = useState("");
  const [dia, setDia] = useState("");
  const [pulse, setPulse] = useState("");
  const [note, setNote] = useState("");

  const data = summary.data;
  const cat = data?.category ? pressureCat(data.category) : undefined;

  const chartData = useMemo(
    () =>
      (data?.series ?? []).map((e) => ({
        t: shortTime(e.measuredAt),
        sys: e.systolic,
        dia: e.diastolic,
        pulse: e.pulse ?? null,
      })),
    [data],
  );

  function submit() {
    const s = num(sys);
    const d = num(dia);
    if (s <= 0 || d <= 0) return;
    add.mutate(
      { systolic: s, diastolic: d, pulse: pulse ? num(pulse) : null, note: note.trim() || undefined },
      {
        onSuccess: () => {
          setSys("");
          setDia("");
          setPulse("");
          setNote("");
        },
      },
    );
  }

  const avg = data?.averages;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Давление</h1>
        <p className="text-sm text-ink-500">Дневник артериального давления и пульса с графиком и средними.</p>
      </div>

      {/* stat cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="card">
          <div className="text-sm text-ink-500">Последнее измерение</div>
          <div className="mt-1 text-3xl font-black">
            {data?.latest ? (
              <>
                {data.latest.systolic}<span className="text-ink-500">/</span>{data.latest.diastolic}
                <span className="ml-1 text-base font-medium text-ink-500">мм рт.ст.</span>
              </>
            ) : (
              "—"
            )}
          </div>
          {cat ? (
            <span className={`mt-2 inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${cat.badge}`}>{cat.label}</span>
          ) : (
            <div className="mt-2 text-sm text-ink-500">нет данных</div>
          )}
        </div>
        <div className="card">
          <div className="text-sm text-ink-500">Среднее{data?.count ? ` (${data.count} изм.)` : ""}</div>
          <div className="mt-1 text-3xl font-black">
            {avg?.systolic != null && avg?.diastolic != null ? (
              <>
                {avg.systolic}<span className="text-ink-500">/</span>{avg.diastolic}
              </>
            ) : (
              "—"
            )}
          </div>
          <div className="mt-1 text-sm text-ink-500">систолическое / диастолическое</div>
        </div>
        <div className="card">
          <div className="text-sm text-ink-500">Пульс</div>
          <div className="mt-1 text-3xl font-black">
            {data?.latest?.pulse != null ? (
              <>
                {data.latest.pulse}<span className="ml-1 text-base font-medium text-ink-500">уд/мин</span>
              </>
            ) : (
              "—"
            )}
          </div>
          {avg?.pulse != null ? <div className="mt-1 text-sm text-ink-500">среднее {avg.pulse}</div> : null}
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-[1fr_1.4fr]">
        {/* input */}
        <div className="card space-y-3">
          <h2 className="font-semibold">Записать измерение</h2>
          <div className="grid grid-cols-3 gap-2">
            <div>
              <label className="label">Систол.</label>
              <input className="input" inputMode="numeric" placeholder="120" value={sys} onChange={(e) => setSys(e.target.value)} />
            </div>
            <div>
              <label className="label">Диастол.</label>
              <input className="input" inputMode="numeric" placeholder="80" value={dia} onChange={(e) => setDia(e.target.value)} />
            </div>
            <div>
              <label className="label">Пульс</label>
              <input className="input" inputMode="numeric" placeholder="70" value={pulse} onChange={(e) => setPulse(e.target.value)} />
            </div>
          </div>
          <input className="input" placeholder="Заметка (напр. после кофе)" value={note} onChange={(e) => setNote(e.target.value)} />
          <button onClick={submit} disabled={add.isPending} className="btn-primary w-full">
            Сохранить
          </button>
          <p className="text-xs text-ink-500">Можно записывать несколько раз в день (утро/вечер).</p>
        </div>

        {/* chart */}
        <div className="card">
          <h2 className="mb-4 font-semibold">Динамика</h2>
          {chartData.length > 0 ? (
            <div className="h-72 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData} margin={{ top: 5, right: 12, left: 4, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                  <XAxis dataKey="t" tick={{ fill: "#64748b", fontSize: 11 }} tickLine={false} axisLine={false} minTickGap={24} />
                  <YAxis width={40} domain={["dataMin - 10", "dataMax + 10"]} tick={{ fill: "#64748b", fontSize: 12 }} tickLine={false} axisLine={false} />
                  <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #334155", borderRadius: 12, color: "#e2e8f0" }} />
                  <ReferenceLine y={120} stroke="#334155" strokeDasharray="3 3" />
                  <ReferenceLine y={80} stroke="#334155" strokeDasharray="3 3" />
                  <Line type="monotone" dataKey="sys" name="Систол." stroke="#f87171" strokeWidth={2.5} dot={{ r: 2 }} activeDot={{ r: 5 }} />
                  <Line type="monotone" dataKey="dia" name="Диастол." stroke="#38bdf8" strokeWidth={2.5} dot={{ r: 2 }} activeDot={{ r: 5 }} />
                  <Line type="monotone" dataKey="pulse" name="Пульс" stroke="#a78bfa" strokeWidth={2} dot={false} connectNulls />
                </LineChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <p className="text-sm text-ink-500">Пока нет измерений. Запишите первое слева.</p>
          )}

          {data && data.series.length > 0 ? (
            <ul className="mt-4 max-h-44 space-y-1 overflow-auto pr-1 text-sm">
              {[...data.series].reverse().map((e) => (
                <li key={e.id} className="flex items-center justify-between rounded-lg px-2 py-1 hover:bg-ink-800/50">
                  <span>
                    <span className="font-medium">
                      {e.systolic}/{e.diastolic}
                    </span>
                    {e.pulse != null ? <span className="text-ink-500"> · ♥ {e.pulse}</span> : null}
                    <span className="ml-2 text-ink-500">{shortTime(e.measuredAt)}</span>
                    {e.note ? <span className="ml-2 text-ink-500">· {e.note}</span> : null}
                  </span>
                  <button onClick={() => del.mutate(e.id)} className="text-ink-500 hover:text-bad">
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
