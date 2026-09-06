// Display helpers for the training domain. These mirror the keys defined in the
// backend `meta.go`; unknown keys (user-added equipment tags) fall back to the
// raw value so nothing ever renders blank.

export const CATEGORY_LABELS: Record<string, string> = {
  strength: "Силовое",
  cardio: "Кардио",
  mobility: "Мобильность",
};

export const DIFFICULTY_LABELS: Record<string, string> = {
  easy: "Лёгкий",
  medium: "Средний",
  hard: "Тяжёлый",
};

export const JOINT_LABELS: Record<string, string> = {
  low: "Низкая нагрузка на суставы",
  medium: "Средняя нагрузка на суставы",
  high: "Высокая нагрузка на суставы",
};

export const EQUIPMENT_LABELS: Record<string, string> = {
  none: "Без инвентаря",
  dumbbells: "Гантели",
  barbell: "Штанга",
  kettlebell: "Гиря",
  pullup_bar: "Турник",
  dip_bars: "Брусья",
  bench: "Скамья",
  resistance_band: "Резинка",
  mat: "Коврик",
  machine: "Тренажёр",
  jump_rope: "Скакалка",
  box: "Тумба",
  trx: "Петли TRX",
};

export const MUSCLE_LABELS: Record<string, string> = {
  chest: "Грудь",
  back: "Спина",
  shoulders: "Плечи",
  biceps: "Бицепс",
  triceps: "Трицепс",
  legs: "Ноги",
  glutes: "Ягодицы",
  core: "Пресс / кор",
  full_body: "Всё тело",
};

/** Tailwind color classes per difficulty, for badges. */
export const DIFFICULTY_STYLE: Record<string, string> = {
  easy: "bg-good/15 text-good",
  medium: "bg-warn/15 text-warn",
  hard: "bg-bad/15 text-bad",
};

export function label(map: Record<string, string>, key: string): string {
  return map[key] ?? key;
}

/** Formats a workout item's prescription (sets×reps / duration) for display. */
export function prescription(it: {
  sets: number | null;
  reps: number | null;
  durationSec: number | null;
  weightKg: number | null;
  restSec: number | null;
}): string {
  const parts: string[] = [];
  if (it.sets && it.reps) parts.push(`${it.sets}×${it.reps}`);
  else if (it.sets) parts.push(`${it.sets} подх.`);
  else if (it.reps) parts.push(`${it.reps} повт.`);
  if (it.durationSec) parts.push(formatDuration(it.durationSec));
  if (it.weightKg) parts.push(`${it.weightKg} кг`);
  if (it.restSec) parts.push(`отдых ${formatDuration(it.restSec)}`);
  return parts.join(" · ");
}

export function formatDuration(sec: number): string {
  if (sec < 60) return `${sec} с`;
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return s ? `${m} мин ${s} с` : `${m} мин`;
}
