"use client";

import Link from "next/link";
import { CalendarClock } from "lucide-react";
import { useBoards, useSetProjectPriority } from "@/hooks/useGtdGraph";
import type { GtdColor } from "@/lib/types";
import { PriorityBars, priorityLabel } from "@/components/gtd/PriorityBars";

const DOT: Record<GtdColor, string> = { green: "bg-good", yellow: "bg-warn", red: "bg-bad" };
const RING: Record<GtdColor, string> = { green: "border-good/40", yellow: "border-warn/50", red: "border-bad/60" };
const COLOR_LABEL: Record<GtdColor, string> = { green: "в порядке", yellow: "скоро дедлайн", red: "просрочено" };

// GraphsPanel is the prioritised board of graphs: red graphs float to the top,
// then yellow, then green, and within each colour band graphs are ordered by
// priority (the backend already sorts; here you can also re-prioritise inline).
export function GraphsPanel() {
  const boards = useBoards();
  const setPriority = useSetProjectPriority();
  const items = boards.data?.items ?? [];
  const projects = items.filter((b) => b.projectId);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-lg font-semibold">Мои графы по приоритету</h2>
          <p className="text-sm text-ink-500">Сверху — то, что горит: сначала красные, потом жёлтые и зелёные; внутри цвета — по приоритету.</p>
        </div>
        <div className="flex items-center gap-3 text-xs text-ink-500">
          <span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-full bg-bad" /> красный</span>
          <span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-full bg-warn" /> жёлтый</span>
          <span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-full bg-good" /> зелёный</span>
        </div>
      </div>

      {boards.isLoading ? (
        <div className="space-y-2">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded-2xl bg-ink-800/40" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <div className="card text-sm text-ink-500">
          Пока нет графов с задачами. Создайте проект и добавьте узлы на <Link href="/gtd/graph/root" className="text-brand">карте проектов</Link>.
        </div>
      ) : (
        <div className="space-y-2.5">
          {projects.map((b) => {
            const c = b.summary.color;
            return (
              <div key={b.projectId} className={`rounded-2xl border ${RING[c]} bg-ink-950/40 p-3.5`}>
                <div className="flex items-start gap-3">
                  <span className={`mt-1 h-3.5 w-3.5 shrink-0 rounded-full ${DOT[c]}`} title={COLOR_LABEL[c]} />
                  <div className="min-w-0 flex-1">
                    <Link href={`/gtd/graph/${b.projectId}`} className="block truncate font-semibold hover:text-brand">
                      {b.title}
                    </Link>
                    <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-ink-500">
                      {b.summary.red > 0 ? <span className="text-bad">● {b.summary.red} просрочено</span> : null}
                      {b.summary.yellow > 0 ? <span className="text-warn">● {b.summary.yellow} скоро</span> : null}
                      <span className="text-good">● {b.summary.green} в норме</span>
                      {b.summary.projectedCompletion ? (
                        <span className="flex items-center gap-1 text-ink-400">
                          <CalendarClock size={12} /> до {b.summary.projectedCompletion}
                        </span>
                      ) : null}
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-1">
                    <PriorityBars value={b.priority} onChange={(p) => setPriority.mutate({ id: b.projectId!, priority: p })} />
                    <span className="text-[11px] text-ink-500">{priorityLabel(b.priority)}</span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
