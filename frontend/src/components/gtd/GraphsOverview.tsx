"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { SlidersHorizontal } from "lucide-react";
import { useBoards, useGraphSettings, useSaveGraphSettings } from "@/hooks/useGtdGraph";
import type { GtdColor } from "@/lib/types";

const DOT: Record<GtdColor, string> = { green: "bg-good", yellow: "bg-warn", red: "bg-bad" };

// GraphsOverview lists every graph (root + projects) highlighted by its worst
// node's colour, with per-graph counts and projected completion, plus the
// deadline-colour thresholds.
export function GraphsOverview() {
  const boards = useBoards();
  const items = boards.data?.items ?? [];

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold">Мои графы</h2>
        <ThresholdSettings />
      </div>
      {boards.isLoading ? (
        <div className="card h-16 animate-pulse bg-ink-800/40" />
      ) : (
        <div className="space-y-2">
          {items.map((b) => {
            const href = b.projectId ? `/gtd/graph/${b.projectId}` : "/gtd/graph/root";
            return (
              <Link
                key={b.projectId ?? "root"}
                href={href}
                className="flex items-center gap-3 rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2.5 transition hover:border-brand/50"
              >
                <span className={`h-3 w-3 shrink-0 rounded-full ${DOT[b.summary.color]}`} title={b.summary.color} />
                <span className="min-w-0 flex-1 truncate font-medium">{b.title}</span>
                <span className="flex shrink-0 items-center gap-2 text-xs text-ink-500">
                  {b.summary.red > 0 ? <span className="text-bad">● {b.summary.red}</span> : null}
                  {b.summary.yellow > 0 ? <span className="text-warn">● {b.summary.yellow}</span> : null}
                  <span className="text-good">● {b.summary.green}</span>
                  {b.summary.projectedCompletion ? <span className="ml-1">до {b.summary.projectedCompletion}</span> : null}
                </span>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

function ThresholdSettings() {
  const settings = useGraphSettings();
  const save = useSaveGraphSettings();
  const [open, setOpen] = useState(false);
  const [soon, setSoon] = useState("3");
  const [grace, setGrace] = useState("1");

  useEffect(() => {
    if (settings.data) {
      setSoon(String(settings.data.soonDays));
      setGrace(String(settings.data.graceDays));
    }
  }, [settings.data]);

  return (
    <div className="relative">
      <button onClick={() => setOpen((o) => !o)} className="btn-ghost !py-1.5 text-sm">
        <SlidersHorizontal size={14} /> Пороги цветов
      </button>
      {open ? (
        <div className="absolute right-0 z-10 mt-1 w-64 space-y-3 rounded-xl border border-ink-800 bg-ink-950 p-3 shadow-xl">
          <div>
            <label className="label">Жёлтый за сколько дней до дедлайна</label>
            <input className="input" inputMode="numeric" value={soon} onChange={(e) => setSoon(e.target.value)} />
          </div>
          <div>
            <label className="label">Жёлтый ещё сколько дней после (потом красный)</label>
            <input className="input" inputMode="numeric" value={grace} onChange={(e) => setGrace(e.target.value)} />
          </div>
          <button
            onClick={async () => {
              await save.mutateAsync({ soonDays: Math.max(0, parseInt(soon) || 0), graceDays: Math.max(0, parseInt(grace) || 0) });
              setOpen(false);
            }}
            disabled={save.isPending}
            className="btn-primary w-full !py-1.5"
          >
            Сохранить
          </button>
        </div>
      ) : null}
    </div>
  );
}
