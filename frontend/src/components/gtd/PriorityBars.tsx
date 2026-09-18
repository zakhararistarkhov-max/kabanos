"use client";

import { PRIORITY_LABELS } from "@/lib/gtd";

// Priority is shown as five rising bars filled up to the level (5 = highest).
// It's deliberately colour-neutral (brand fill) so it never clashes with the
// deadline colours (green/yellow/red). Labels are the canonical GTD ones.
export function priorityLabel(p: number): string {
  return PRIORITY_LABELS[p] ?? PRIORITY_LABELS[3];
}

export function PriorityBars({
  value,
  onChange,
  allowZero,
  size = "md",
}: {
  value: number;
  onChange?: (v: number) => void;
  allowZero?: boolean; // clicking the current top bar clears to 0
  size?: "sm" | "md";
}) {
  const editable = !!onChange;
  const w = size === "sm" ? 4 : 6;
  const base = size === "sm" ? 5 : 6;
  const step = size === "sm" ? 2 : 3;
  return (
    <div className="flex items-end gap-[3px]" title={`Приоритет: ${priorityLabel(value)}`} role={editable ? "group" : undefined}>
      {[1, 2, 3, 4, 5].map((n) => {
        const on = n <= value;
        const cls = `rounded-sm transition ${on ? "bg-brand" : "bg-ink-700"} ${editable ? "cursor-pointer hover:opacity-75" : ""}`;
        const style = { width: w, height: base + n * step };
        if (editable) {
          return (
            <button
              key={n}
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                onChange(allowZero && value === n ? 0 : n);
              }}
              className={cls}
              style={style}
              aria-label={`Приоритет ${n}`}
            />
          );
        }
        return <span key={n} className={cls} style={style} />;
      })}
    </div>
  );
}
