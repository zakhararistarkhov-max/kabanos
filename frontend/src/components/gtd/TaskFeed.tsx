"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { Check, ChevronDown, ChevronUp, ImageIcon, Network, PartyPopper } from "lucide-react";
import { useGtdFeed, useSetItemPriority } from "@/hooks/useGtdGraph";
import { useToggleDone } from "@/hooks/useGtd";
import type { GtdColor, GtdFeedTask } from "@/lib/types";
import { PriorityBars, priorityLabel } from "@/components/gtd/PriorityBars";

const ACCENT: Record<GtdColor, string> = { green: "#34d399", yellow: "#f59e0b", red: "#ef4444" };
const COLOR_TEXT: Record<GtdColor, string> = { green: "text-good", yellow: "text-warn", red: "text-bad" };

function deadlineInfo(deadline: string | null, color: GtdColor): string {
  if (!deadline) return "Без срока";
  const today = new Date();
  const d = new Date(deadline + "T00:00:00");
  const days = Math.round((d.getTime() - new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime()) / 86_400_000);
  if (days === 0) return "Дедлайн сегодня";
  if (days < 0) return `Просрочено на ${Math.abs(days)} дн.`;
  if (color === "red") return `Просрочено`;
  return `Осталось ${days} дн. (до ${deadline})`;
}

// TaskFeed is a vertical, full-height, swipeable "reel" of the things to do —
// red (overdue) cards first, then yellow and green, by priority within a colour.
export function TaskFeed() {
  const feed = useGtdFeed();
  const toggleDone = useToggleDone();
  const setPriority = useSetItemPriority();
  const containerRef = useRef<HTMLDivElement>(null);
  const [index, setIndex] = useState(0);

  // Render from a local copy so in-place edits (priority) don't reorder the reel
  // under the user's finger; a changed *set* of cards (e.g. after "Готово") does.
  const [cards, setCards] = useState<GtdFeedTask[]>([]);
  useEffect(() => {
    const next = feed.data?.items ?? [];
    setCards((prev) => {
      const prevIds = prev.map((c) => c.nodeId);
      const nextSet = new Set(next.map((c) => c.nodeId));
      const sameSet = prevIds.length === nextSet.size && prevIds.every((id) => nextSet.has(id));
      if (sameSet && prev.length) {
        const byId = new Map(next.map((c) => [c.nodeId, c]));
        return prev.map((c) => byId.get(c.nodeId) ?? c);
      }
      return next;
    });
  }, [feed.data]);

  const scrollTo = useCallback((i: number) => {
    const el = containerRef.current;
    if (!el) return;
    const clamped = Math.max(0, Math.min(i, el.children.length - 1));
    (el.children[clamped] as HTMLElement | undefined)?.scrollIntoView({ behavior: "smooth" });
  }, []);

  const onScroll = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    setIndex(Math.round(el.scrollTop / el.clientHeight));
  }, []);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "ArrowDown" || e.key === "ArrowRight") {
        e.preventDefault();
        scrollTo(index + 1);
      } else if (e.key === "ArrowUp" || e.key === "ArrowLeft") {
        e.preventDefault();
        scrollTo(index - 1);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [index, scrollTo]);

  const counts = useMemo(() => {
    let red = 0, yellow = 0, green = 0;
    for (const c of cards) {
      if (c.color === "red") red++;
      else if (c.color === "yellow") yellow++;
      else green++;
    }
    return { red, yellow, green };
  }, [cards]);

  if (feed.isLoading) {
    return <div className="h-[calc(100dvh-8rem)] animate-pulse rounded-3xl bg-ink-800/40" />;
  }

  if (cards.length === 0) {
    return (
      <div className="flex h-[calc(100dvh-8rem)] flex-col items-center justify-center gap-3 rounded-3xl border border-ink-800 bg-ink-950/40 text-center">
        <PartyPopper size={48} className="text-brand" />
        <p className="text-lg font-semibold">Все дела разобраны!</p>
        <p className="max-w-xs text-sm text-ink-500">
          В ваших графах нет открытых задач. Добавьте узлы на <Link href="/gtd/graph/root" className="text-brand">карте проектов</Link>.
        </p>
      </div>
    );
  }

  return (
    <div className="relative">
      {/* progress rail */}
      <div className="pointer-events-none absolute right-1 top-1/2 z-10 flex -translate-y-1/2 flex-col gap-1.5">
        {cards.slice(0, 40).map((c, i) => (
          <span
            key={c.nodeId}
            className="h-4 w-1 rounded-full transition"
            style={{ background: i === index ? ACCENT[c.color] : "rgb(var(--ink-700))", opacity: i === index ? 1 : 0.5 }}
          />
        ))}
      </div>

      <div
        ref={containerRef}
        onScroll={onScroll}
        className="hide-scrollbar h-[calc(100dvh-8rem)] snap-y snap-mandatory overflow-y-auto rounded-3xl"
        style={{ scrollbarWidth: "none" }}
      >
        {cards.map((c) => (
          <div key={c.nodeId} className="flex h-full snap-start snap-always items-center justify-center p-1">
            <FeedCard
              task={c}
              onDone={() => c.itemId && toggleDone.mutate({ id: c.itemId, done: true })}
              onPriority={(p) => c.itemId && setPriority.mutate({ id: c.itemId, priority: p })}
              doneBusy={toggleDone.isPending}
            />
          </div>
        ))}
      </div>

      {/* nav + counts */}
      <div className="mt-3 flex items-center justify-between">
        <div className="flex items-center gap-3 text-xs text-ink-500">
          <span>{index + 1} / {cards.length}</span>
          {counts.red > 0 ? <span className="text-bad">● {counts.red}</span> : null}
          {counts.yellow > 0 ? <span className="text-warn">● {counts.yellow}</span> : null}
          <span className="text-good">● {counts.green}</span>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => scrollTo(index - 1)} disabled={index === 0} className="btn-ghost !px-2.5 !py-2 disabled:opacity-40" aria-label="Предыдущее">
            <ChevronUp size={18} />
          </button>
          <button onClick={() => scrollTo(index + 1)} disabled={index >= cards.length - 1} className="btn-ghost !px-2.5 !py-2 disabled:opacity-40" aria-label="Следующее">
            <ChevronDown size={18} />
          </button>
        </div>
      </div>
    </div>
  );
}

function FeedCard({
  task,
  onDone,
  onPriority,
  doneBusy,
}: {
  task: GtdFeedTask;
  onDone: () => void;
  onPriority: (p: number) => void;
  doneBusy: boolean;
}) {
  const accent = ACCENT[task.color];
  const cover = task.imageUrls[0];
  return (
    <div
      className="flex h-full w-full max-w-md flex-col overflow-hidden rounded-3xl border bg-ink-900/60 shadow-xl"
      style={{ borderColor: accent }}
    >
      {cover ? (
        <div className="relative h-2/5 w-full shrink-0">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={cover} alt="" className="h-full w-full object-cover" />
          {task.imageUrls.length > 1 ? (
            <span className="absolute bottom-2 right-2 flex items-center gap-1 rounded-full bg-black/60 px-2 py-0.5 text-xs text-white">
              <ImageIcon size={12} /> {task.imageUrls.length}
            </span>
          ) : null}
          <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-ink-900/90 to-transparent" />
        </div>
      ) : (
        <div className="h-2 w-full shrink-0" style={{ background: accent }} />
      )}

      <div className="flex min-h-0 flex-1 flex-col gap-3 p-5">
        <div className="flex items-center gap-2">
          <span className="rounded-full px-2.5 py-1 text-xs font-semibold" style={{ background: `${accent}22`, color: accent }}>
            {deadlineInfo(task.deadline, task.color)}
          </span>
        </div>

        <h2 className="text-2xl font-bold leading-snug">{task.label || "Без названия"}</h2>

        {task.note ? <p className="min-h-0 flex-1 overflow-y-auto whitespace-pre-wrap text-sm text-ink-300">{task.note}</p> : <div className="flex-1" />}

        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-xs text-ink-500">
            <Network size={13} />
            {task.boardId ? (
              <Link href={`/gtd/graph/${task.boardId}`} className="truncate hover:text-brand">{task.boardTitle}</Link>
            ) : (
              <Link href="/gtd/graph/root" className="truncate hover:text-brand">{task.boardTitle}</Link>
            )}
          </div>
          <div className="flex flex-col items-end gap-0.5">
            <PriorityBars value={task.priority} onChange={onPriority} allowZero size="sm" />
            <span className="text-[10px] text-ink-500">{priorityLabel(task.priority)}</span>
          </div>
        </div>

        <button
          onClick={onDone}
          disabled={doneBusy || !task.itemId}
          className="flex items-center justify-center gap-2 rounded-2xl bg-brand py-3 font-semibold text-ink-950 transition hover:bg-brand-strong disabled:opacity-60"
        >
          <Check size={18} /> Готово
        </button>
        <p className={`text-center text-[11px] ${COLOR_TEXT[task.color]}`}>Свайп вверх — следующее дело</p>
      </div>
    </div>
  );
}
