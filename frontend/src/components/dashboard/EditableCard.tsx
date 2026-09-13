"use client";

import { useRef, useState, type RefObject } from "react";
import { GripVertical, MoveHorizontal, X } from "lucide-react";
import { spanClass } from "@/components/dashboard/widgets";
import type { Span } from "@/hooks/useDashboardLayout";

interface Props {
  id: string;
  span: Span;
  colCount: 1 | 2 | 3;
  gridRef: RefObject<HTMLDivElement | null>;
  isDragging: boolean;
  onReorderStart: (id: string) => void;
  onResize: (id: string, span: Span) => void;
  onHide: (id: string) => void;
  children: React.ReactNode;
}

const GAP = 16; // matches the grid's gap-4

// EditableCard wraps a dashboard widget in edit mode: the grip drags it to
// reorder (pointer-based, so it works with mouse and touch), the corner grip
// resizes it across grid columns (snapping to 1..colCount), and × removes it.
// Widget content is inert so clicks don't navigate while editing.
export function EditableCard({
  id,
  span,
  colCount,
  gridRef,
  isDragging,
  onReorderStart,
  onResize,
  onHide,
  children,
}: Props) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [resizing, setResizing] = useState(false);

  function startResize(e: React.PointerEvent) {
    e.preventDefault();
    e.stopPropagation();
    setResizing(true);
    try {
      (e.currentTarget as Element).setPointerCapture(e.pointerId);
    } catch {
      /* ignore */
    }
  }

  function moveResize(e: React.PointerEvent) {
    if (!resizing || !gridRef.current || !cardRef.current) return;
    const grid = gridRef.current.getBoundingClientRect();
    const card = cardRef.current.getBoundingClientRect();
    const step = (grid.width + GAP) / colCount; // one column incl. gap
    let s = Math.round((e.clientX - card.left + GAP) / step);
    s = Math.max(1, Math.min(colCount, s));
    if (s !== span) onResize(id, s as Span);
  }

  function endResize(e: React.PointerEvent) {
    if (!resizing) return;
    setResizing(false);
    try {
      (e.currentTarget as Element).releasePointerCapture(e.pointerId);
    } catch {
      /* ignore */
    }
  }

  return (
    <div
      ref={cardRef}
      data-wid={id}
      className={`relative ${spanClass(span)} ${isDragging ? "opacity-60 ring-2 ring-brand" : ""} rounded-2xl transition-[opacity]`}
    >
      {/* edit chrome */}
      <div className="pointer-events-none absolute inset-0 z-10 rounded-2xl border-2 border-dashed border-brand/40" />

      {/* drag-to-reorder grip */}
      <div
        onPointerDown={(e) => {
          e.preventDefault();
          onReorderStart(id);
        }}
        className="absolute left-2 top-2 z-20 grid h-7 w-7 cursor-grab touch-none place-items-center rounded-lg bg-ink-950/80 text-ink-300 shadow active:cursor-grabbing"
        title="Перетащите, чтобы переместить"
      >
        <GripVertical size={15} />
      </div>

      <button
        onPointerDown={(e) => e.stopPropagation()}
        onClick={() => onHide(id)}
        className="absolute right-2 top-2 z-20 grid h-7 w-7 place-items-center rounded-lg bg-ink-950/80 text-ink-400 shadow transition hover:text-bad"
        aria-label="Скрыть виджет"
        title="Скрыть"
      >
        <X size={15} />
      </button>

      {/* resize grip (hidden when there's only one column to fill) */}
      {colCount > 1 ? (
        <div
          onPointerDown={startResize}
          onPointerMove={moveResize}
          onPointerUp={endResize}
          onPointerCancel={endResize}
          className="absolute -bottom-1 -right-1 z-20 grid h-7 w-7 cursor-ew-resize touch-none place-items-center rounded-lg bg-brand text-ink-950 shadow"
          title="Потяните, чтобы изменить ширину"
        >
          <MoveHorizontal size={15} />
        </div>
      ) : null}

      {/* inert widget preview */}
      <div className="pointer-events-none select-none">{children}</div>
    </div>
  );
}
