"use client";

import type { CheckinStatus, HabitRecentDay } from "@/lib/types";

const CELL: Record<CheckinStatus, string> = {
  success: "bg-good",
  fail: "bg-bad",
  none: "bg-ink-800",
};

// HabitStrip is the compact preview of the tracker: one cell per recent day,
// green = kept, red = missed, muted = unmarked.
export function HabitStrip({ recent }: { recent: HabitRecentDay[] }) {
  return (
    <div className="flex gap-[3px]">
      {recent.map((d) => (
        <span key={d.day} title={`${d.day}: ${d.status === "success" ? "✓" : d.status === "fail" ? "✗" : "—"}`} className={`h-4 flex-1 rounded-sm ${CELL[d.status]}`} />
      ))}
    </div>
  );
}
