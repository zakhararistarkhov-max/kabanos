"use client";

// A visual water container that fills from the bottom to the consumed/goal
// ratio, with an animated surface. Purely presentational.
export function WaterContainer({
  consumedMl,
  goalMl,
  percent,
}: {
  consumedMl: number;
  goalMl: number;
  percent: number;
}) {
  const fill = Math.min(100, Math.max(0, percent));
  const reached = consumedMl >= goalMl && goalMl > 0;

  return (
    <div className="relative mx-auto h-80 w-56 select-none">
      {/* glass body */}
      <div className="absolute inset-0 overflow-hidden rounded-b-[2.5rem] rounded-t-2xl border-2 border-ink-700 bg-ink-950/40 shadow-inner">
        {/* water */}
        <div
          className="absolute inset-x-0 bottom-0 transition-[height] duration-700 ease-out"
          style={{ height: `${fill}%` }}
        >
          {/* animated surface waves */}
          <div className="absolute -top-3 left-0 h-6 w-[200%] animate-wave">
            <svg viewBox="0 0 400 20" preserveAspectRatio="none" className="h-full w-full">
              <path d="M0 10 Q 50 0 100 10 T 200 10 T 300 10 T 400 10 V20 H0 Z" fill="#0ea5e9" opacity="0.9" />
            </svg>
          </div>
          <div className="absolute -top-2 left-0 h-6 w-[200%] animate-wave-slow">
            <svg viewBox="0 0 400 20" preserveAspectRatio="none" className="h-full w-full">
              <path d="M0 10 Q 50 20 100 10 T 200 10 T 300 10 T 400 10 V20 H0 Z" fill="#38bdf8" opacity="0.55" />
            </svg>
          </div>
          <div className="h-full w-full bg-gradient-to-b from-brand-strong to-brand-strong/70" />
        </div>

        {/* level marks */}
        {[25, 50, 75].map((m) => (
          <div key={m} className="absolute left-0 w-3 border-t border-ink-700/70" style={{ bottom: `${m}%` }} />
        ))}
      </div>

      {/* readout */}
      <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
        <div className="text-4xl font-black text-white drop-shadow">{Math.round(percent)}%</div>
        <div className="mt-1 text-sm font-medium text-ink-100 drop-shadow">
          {(consumedMl / 1000).toFixed(2)} / {(goalMl / 1000).toFixed(2)} л
        </div>
        {reached ? <div className="mt-1 text-xs font-semibold text-good drop-shadow">Цель достигнута 🎉</div> : null}
      </div>
    </div>
  );
}
