"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Plus, SlidersHorizontal } from "lucide-react";
import { useDashboardLayout, type Span } from "@/hooks/useDashboardLayout";
import { useGridColumns } from "@/hooks/useGridColumns";
import { WIDGET_IDS, spanClass, widgetById } from "@/components/dashboard/widgets";
import { EditableCard } from "@/components/dashboard/EditableCard";
import { KabanosMark } from "@/components/Logo";

export default function DashboardPage() {
  const { mounted, enabled, hidden, spans, toggle, reorder, setSpan, reset } = useDashboardLayout(WIDGET_IDS);
  const [customizing, setCustomizing] = useState(false);
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const cols = useGridColumns();

  // Effective width for a widget: the user's override, else the widget default.
  const effSpan = (id: string): Span => spans[id] ?? (widgetById(id)?.span ?? 1);

  // Keep the latest order/reorder reachable from the imperative drag listeners.
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;
  const reorderRef = useRef(reorder);
  reorderRef.current = reorder;
  const cleanupRef = useRef<() => void>(() => {});

  useEffect(() => () => cleanupRef.current(), []);

  // startDrag attaches the move/up listeners synchronously on pointer-down (not
  // via an effect) so a fast drag doesn't lose its first moves. It reorders live
  // as the pointer passes over other cards (nearest [data-wid] under the point).
  function startDrag(id: string) {
    setDraggingId(id);
    const onMove = (e: PointerEvent) => {
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const overId = el?.closest("[data-wid]")?.getAttribute("data-wid");
      if (!overId || overId === id) return;
      const cur = [...enabledRef.current];
      const from = cur.indexOf(id);
      const to = cur.indexOf(overId);
      if (from < 0 || to < 0) return;
      cur.splice(from, 1);
      cur.splice(to, 0, id);
      reorderRef.current(cur);
    };
    const onUp = () => {
      setDraggingId(null);
      cleanupRef.current();
    };
    cleanupRef.current = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onUp);
      cleanupRef.current = () => {};
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onUp);
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <span className="animate-float">
          <KabanosMark size={46} />
        </span>
        <button onClick={() => setCustomizing((s) => !s)} className={customizing ? "btn-primary" : "btn-ghost"}>
          {customizing ? (
            <>
              <Check size={16} /> Готово
            </>
          ) : (
            <>
              <SlidersHorizontal size={16} /> Настроить
            </>
          )}
        </button>
      </div>

      {customizing && (
        <div className="card rise-in space-y-3">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-ink-400">
              Перетаскивайте кубики, чтобы поменять порядок; тяните уголок&nbsp;
              <span className="inline-grid h-4 w-4 -translate-y-px place-items-center rounded bg-brand align-middle text-[10px] text-ink-950">↔</span>
              &nbsp;— чтобы изменить ширину; × — скрыть.
            </p>
            <button onClick={reset} className="shrink-0 text-sm text-ink-500 hover:text-ink-100">
              Сбросить
            </button>
          </div>

          {hidden.length > 0 && (
            <div>
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-500">Скрытые — нажмите, чтобы добавить</p>
              <div className="flex flex-wrap gap-2">
                {hidden.map((id) => {
                  const w = widgetById(id);
                  if (!w) return null;
                  return (
                    <button key={id} onClick={() => toggle(id)} className="chip inline-flex items-center gap-1.5 transition hover:border-brand/50 hover:text-ink-100">
                      <Plus size={14} strokeWidth={2.5} />
                      <w.icon size={14} strokeWidth={2} />
                      {w.title}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}

      {!mounted ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="card h-40 animate-pulse bg-ink-800/40" />
          ))}
        </div>
      ) : enabled.length === 0 ? (
        <div className="card text-center text-ink-500">
          Все кубики скрыты. Нажмите «Настроить» и добавьте нужные.
        </div>
      ) : (
        <div ref={gridRef} className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {enabled.map((id, i) => {
            const w = widgetById(id);
            if (!w) return null;
            const W = w.Component;
            const span = effSpan(id);

            if (customizing) {
              return (
                <EditableCard
                  key={id}
                  id={id}
                  span={span}
                  colCount={cols}
                  gridRef={gridRef}
                  isDragging={draggingId === id}
                  onReorderStart={startDrag}
                  onResize={(wid, s) => setSpan(wid, s)}
                  onHide={toggle}
                >
                  <W />
                </EditableCard>
              );
            }

            return (
              <div key={id} className={`rise-in ${spanClass(span)}`} style={{ animationDelay: `${Math.min(i, 8) * 60}ms` }}>
                <W />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
