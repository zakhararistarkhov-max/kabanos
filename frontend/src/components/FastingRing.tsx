"use client";

import { useEffect, useState } from "react";
import type { FastingPhase } from "@/lib/types";

const PHASE_COLOR: Record<Exclude<FastingPhase, "idle">, string> = {
  fasting: "#38bdf8", // brand
  eating: "#34d399", // good
};
const OVERRUN_COLOR = "#f59e0b"; // warn
const TRACK = "#1e293b"; // ink-800

const PHASE_LABEL: Record<FastingPhase, string> = { fasting: "Голодание", eating: "Приём пищи", idle: "Готов начать" };

function pad(n: number): string {
  return String(Math.floor(n)).padStart(2, "0");
}
function fmtDur(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  return `${pad(h)}:${pad(m)}:${pad(sec)}`;
}

// FastingRing draws the current phase as a filling ring with a live countdown in
// the middle. It ticks every second from the absolute phase start/end times, so
// it stays smooth without polling the server.
export function FastingRing({
  phase,
  phaseStartAt,
  phaseEndAt,
  size = 240,
  stroke = 16,
}: {
  phase: FastingPhase;
  phaseStartAt: string | null;
  phaseEndAt: string | null;
  size?: number;
  stroke?: number;
}) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, []);

  const start = phaseStartAt ? Date.parse(phaseStartAt) : null;
  const end = phaseEndAt ? Date.parse(phaseEndAt) : null;
  const active = phase !== "idle" && start != null && end != null && end > start;

  const total = active ? end! - start! : 0;
  const elapsed = active ? now - start! : 0;
  const progress = active ? Math.min(1, Math.max(0, elapsed / total)) : 0;
  const remainingMs = active ? end! - now : 0;
  const overrun = active && remainingMs <= 0;
  const pct = Math.round(progress * 100);

  const color = phase === "idle" ? TRACK : overrun ? OVERRUN_COLOR : PHASE_COLOR[phase];
  const r = (size - stroke) / 2;
  const circ = 2 * Math.PI * r;
  const offset = circ * (1 - progress);

  return (
    <div className="relative inline-grid place-items-center" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke={TRACK} strokeWidth={stroke} />
        {active ? (
          <circle
            cx={size / 2}
            cy={size / 2}
            r={r}
            fill="none"
            stroke={color}
            strokeWidth={stroke}
            strokeLinecap="round"
            strokeDasharray={circ}
            strokeDashoffset={offset}
            style={{ transition: "stroke-dashoffset 0.5s linear, stroke 0.3s" }}
          />
        ) : null}
      </svg>
      <div className="absolute inset-0 grid place-items-center text-center">
        <div>
          <div className="text-xs font-semibold uppercase tracking-wide" style={{ color }}>
            {PHASE_LABEL[phase]}
          </div>
          {active ? (
            <>
              <div className="mt-1 font-mono text-3xl font-bold tabular-nums" style={{ fontSize: size / 8 }}>
                {overrun ? `+${fmtDur(-remainingMs)}` : fmtDur(remainingMs)}
              </div>
              <div className="mt-0.5 text-xs text-ink-500">
                {overrun ? "цель достигнута" : `осталось · ${pct}%`}
              </div>
            </>
          ) : (
            <div className="mt-1 text-sm text-ink-500">нажмите «Начать»</div>
          )}
        </div>
      </div>
    </div>
  );
}
