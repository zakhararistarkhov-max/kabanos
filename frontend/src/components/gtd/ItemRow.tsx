"use client";

import { Pencil, Trash2 } from "lucide-react";
import { useDeleteItem, useToggleDone } from "@/hooks/useGtd";
import { BUCKET_ICON, PRIORITY_CLASS, fmtDue, fmtWhen } from "@/lib/gtd";
import type { GtdItem } from "@/lib/types";

// One item as a row with a done checkbox, its metadata, and edit/delete actions.
export function ItemRow({
  item,
  onEdit,
  showBucket = false,
}: {
  item: GtdItem;
  onEdit?: (it: GtdItem) => void;
  showBucket?: boolean;
}) {
  const toggle = useToggleDone();
  const del = useDeleteItem();
  const due = fmtDue(item.dueOn);

  return (
    <div className="flex items-start gap-3 rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2.5">
      <input
        type="checkbox"
        checked={item.done}
        onChange={(e) => toggle.mutate({ id: item.id, done: e.target.checked })}
        className="mt-1 h-4 w-4 shrink-0 accent-brand"
        aria-label="Выполнено"
      />
      <div className="min-w-0 flex-1">
        <div className={`text-sm ${item.done ? "text-ink-600 line-through" : "text-ink-100"}`}>
          {item.priority > 0 ? <span className={PRIORITY_CLASS[item.priority]}>⚑ </span> : null}
          {item.title}
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-ink-500">
          {showBucket ? <span title={item.bucket}>{BUCKET_ICON[item.bucket]}</span> : null}
          {item.context ? <span className="text-brand">{item.context}</span> : null}
          {item.projectTitle ? <span>· {item.projectTitle}</span> : null}
          {item.waitingFor ? <span>· ждём: {item.waitingFor}</span> : null}
          {item.bucket === "calendar" && item.scheduledAt ? <span>· {fmtWhen(item.scheduledAt, item.allDay)}</span> : null}
          {due ? <span>· до {due}</span> : null}
          {item.timeMinutes ? <span>· {item.timeMinutes} мин</span> : null}
          {item.energy ? <span>· энергия: {item.energy === "low" ? "низкая" : item.energy === "medium" ? "средняя" : "высокая"}</span> : null}
          {item.notes ? <span className="max-w-[16rem] truncate italic">· {item.notes}</span> : null}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {onEdit ? (
          <button onClick={() => onEdit(item)} className="text-ink-500 transition hover:text-ink-100" aria-label="Изменить">
            <Pencil size={15} />
          </button>
        ) : null}
        <button
          onClick={() => {
            if (confirm("Удалить пункт?")) del.mutate(item.id);
          }}
          className="text-ink-500 transition hover:text-bad"
          aria-label="Удалить"
        >
          <Trash2 size={15} />
        </button>
      </div>
    </div>
  );
}
