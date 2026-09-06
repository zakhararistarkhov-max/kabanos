"use client";

import type { Medication } from "@/lib/types";

function fmtDate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return d && m ? `${d}.${m}${y ? "." + y.slice(2) : ""}` : iso;
}

function courseLine(med: Medication): string {
  if (med.status === "upcoming") return `начнётся ${fmtDate(med.startDate)}`;
  if (med.status === "finished") return "курс завершён";
  return med.courseTotal ? `день ${med.courseDay} из ${med.courseTotal}` : `день ${med.courseDay}`;
}

// PillBar renders one medication's daily schedule as a row of segments that the
// user fills by taking each scheduled dose. Clicking the next empty segment logs
// an intake; clicking the last filled one undoes it.
export function PillBar({
  med,
  onTake,
  onUndo,
  busy,
}: {
  med: Medication;
  onTake: () => void;
  onUndo: () => void;
  busy?: boolean;
}) {
  const filled = Math.min(med.takenToday, med.timesPerDay);
  const done = med.takenToday >= med.timesPerDay;
  const extra = med.takenToday - med.timesPerDay;

  return (
    <div>
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <div className="flex items-center gap-2">
          <span className="font-semibold">{med.name}</span>
          {med.status === "finished" ? (
            <span className="rounded-full bg-ink-800 px-2 py-0.5 text-xs text-ink-400">завершён</span>
          ) : med.status === "upcoming" ? (
            <span className="rounded-full bg-warn/15 px-2 py-0.5 text-xs text-warn">скоро</span>
          ) : done ? (
            <span className="rounded-full bg-good/15 px-2 py-0.5 text-xs text-good">✓ на сегодня</span>
          ) : null}
        </div>
        <span className="text-xs text-ink-500">{courseLine(med)}</span>
      </div>

      <div className="mt-0.5 text-xs text-ink-500">
        {med.dose} {med.unit} × {med.timesPerDay}/день · за неделю выпито {med.weekTaken}
      </div>

      {/* segmented daily bar */}
      <div className="mt-2 flex items-center gap-2">
        <div className="flex flex-1 gap-1.5">
          {Array.from({ length: med.timesPerDay }).map((_, i) => {
            const isFilled = i < filled;
            const isNext = i === filled;
            const isLastFilled = i === filled - 1;
            const clickable = !busy && (isNext || isLastFilled);
            return (
              <button
                key={i}
                type="button"
                disabled={!clickable}
                onClick={() => (isNext ? onTake() : onUndo())}
                title={isFilled ? "Отменить приём" : "Отметить приём"}
                className={`h-3.5 flex-1 rounded-full transition-all ${
                  isFilled ? "bg-brand" : "bg-ink-800"
                } ${clickable ? "cursor-pointer hover:opacity-80" : "cursor-default"} ${
                  isNext ? "ring-1 ring-inset ring-brand/50" : ""
                }`}
              />
            );
          })}
        </div>
        <span className="w-10 shrink-0 text-right text-sm tabular-nums text-ink-400">
          {filled}/{med.timesPerDay}
        </span>
      </div>

      {/* action row */}
      <div className="mt-2 flex items-center gap-2">
        <button
          type="button"
          onClick={onTake}
          disabled={busy}
          className="btn-primary !py-1.5 !px-3 text-sm"
        >
          💊 Принять
        </button>
        {med.takenToday > 0 ? (
          <button type="button" onClick={onUndo} disabled={busy} className="btn-ghost !py-1.5 !px-3 text-sm">
            ↩ Отменить
          </button>
        ) : null}
        {extra > 0 ? <span className="text-xs text-warn">+{extra} сверх нормы</span> : null}
      </div>
    </div>
  );
}
