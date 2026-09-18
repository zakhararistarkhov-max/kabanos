import type { GtdBucket, GtdEnergy } from "@/lib/types";

// Human labels for buckets, used across the GTD tabs.
export const BUCKET_LABELS: Record<GtdBucket, string> = {
  inbox: "Входящие",
  next: "Следующие действия",
  waiting: "Ожидание",
  calendar: "Календарь",
  someday: "Когда‑нибудь",
  reference: "Справочные материалы",
};

export const BUCKET_ICON: Record<GtdBucket, string> = {
  inbox: "📥",
  next: "⚡",
  waiting: "⏳",
  calendar: "📅",
  someday: "💭",
  reference: "📚",
};

// Suggested contexts (the user can type their own too).
export const CONTEXT_PRESETS = [
  "@звонки",
  "@компьютер",
  "@дом",
  "@офис",
  "@поручения",
  "@встречи",
  "@везде",
];

export const ENERGY_LABELS: Record<GtdEnergy, string> = {
  "": "любая",
  low: "низкая",
  medium: "средняя",
  high: "высокая",
};

// Five priority levels (5 = highest); 0 means "no priority set".
export const PRIORITY_LABELS: Record<number, string> = {
  0: "без приоритета",
  1: "очень низкий",
  2: "низкий",
  3: "средний",
  4: "высокий",
  5: "критичный",
};
export const PRIORITY_LEVELS = [0, 1, 2, 3, 4, 5];

// Colour classes for the priority flag (0 = none).
export const PRIORITY_CLASS: Record<number, string> = {
  0: "text-ink-600",
  1: "text-brand",
  2: "text-warn",
  3: "text-bad",
};

export function fmtDue(dueOn: string | null): string | null {
  if (!dueOn) return null;
  return `${dueOn.slice(8, 10)}.${dueOn.slice(5, 7)}`;
}

export function fmtWhen(iso: string | null, allDay: boolean): string {
  if (!iso) return "";
  const d = new Date(iso);
  const date = d.toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit" });
  if (allDay) return date;
  const time = d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  return `${date} ${time}`;
}

// Converts a datetime-local input value ("YYYY-MM-DDTHH:mm") in the browser's
// zone to an ISO string for the API; "" → null.
export function localToISO(v: string): string | null {
  if (!v) return null;
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

// Converts an ISO timestamp to the value a datetime-local input expects.
export function isoToLocalInput(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
