// Blood-pressure category presentation (labels/colors), mirroring the backend
// ACC/AHA keys from internal/pressure.

export const PRESSURE_CATEGORY: Record<
  string,
  { label: string; text: string; badge: string }
> = {
  normal: { label: "Нормальное", text: "text-good", badge: "bg-good/15 text-good" },
  elevated: { label: "Повышенное", text: "text-warn", badge: "bg-warn/15 text-warn" },
  high1: { label: "Гипертония 1 ст.", text: "text-warn", badge: "bg-warn/15 text-warn" },
  high2: { label: "Гипертония 2 ст.", text: "text-bad", badge: "bg-bad/15 text-bad" },
  crisis: { label: "Гипертонический криз", text: "text-bad", badge: "bg-bad/15 text-bad" },
};

export function pressureCat(key: string) {
  return PRESSURE_CATEGORY[key];
}
