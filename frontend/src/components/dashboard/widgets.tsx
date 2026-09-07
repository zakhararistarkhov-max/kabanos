"use client";

import Link from "next/link";
import type { ComponentType } from "react";
import {
  ClipboardList,
  Droplet,
  Dumbbell,
  Flame,
  HeartPulse,
  Pill,
  Salad,
  Scale,
  Utensils,
  Zap,
  type LucideIcon,
} from "lucide-react";
import { PillBar } from "@/components/PillBar";
import { useWaterDay } from "@/hooks/useWater";
import { useWeightSummary } from "@/hooks/useWeight";
import { useNutritionDay } from "@/hooks/useNutrition";
import { useWorkouts } from "@/hooks/useTraining";
import { useMeds, useTakeIntake, useUndoIntake } from "@/hooks/useMeds";
import { usePressureSummary } from "@/hooks/useBloodPressure";
import { DIFFICULTY_LABELS, DIFFICULTY_STYLE, label } from "@/lib/training";
import { pressureCat } from "@/lib/pressure";

export interface WidgetMeta {
  id: string;
  title: string;
  icon: LucideIcon;
  span: 1 | 2 | 3;
  Component: ComponentType;
}

// IconBadge renders a widget's icon in a rounded square, matching the header
// treatment across widgets.
function IconBadge({ Icon }: { Icon: LucideIcon }) {
  return (
    <span className="grid h-8 w-8 place-items-center rounded-lg bg-ink-800 text-brand">
      <Icon size={17} strokeWidth={2} />
    </span>
  );
}

// ---- shared bits ----

function Skeleton({ h = "h-16" }: { h?: string }) {
  return <div className={`mt-3 ${h} animate-pulse rounded-xl bg-ink-800/50`} />;
}

function Bar({ pct, color = "bg-brand" }: { pct: number; color?: string }) {
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-ink-800">
      <div className={`h-full rounded-full ${color} transition-all duration-500`} style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
    </div>
  );
}

function CardLink({ href, icon, title, hint, children }: { href: string; icon: LucideIcon; title: string; hint: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="card card-interactive block h-full">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 font-semibold">
          <IconBadge Icon={icon} />
          {title}
        </h2>
        <span className="text-sm text-ink-500">{hint} →</span>
      </div>
      {children}
    </Link>
  );
}

// ---- stat widgets ----

function WaterWidget() {
  const water = useWaterDay();
  const d = water.data;
  return (
    <CardLink href="/water" icon={Droplet} title="Вода" hint="сегодня">
      {d ? (
        <>
          <div className="mt-3 stat-value">{Math.round(d.percent)}%</div>
          <div className="mt-1 text-sm text-ink-500">
            {(d.consumedMl / 1000).toFixed(2)} из {(d.goalMl / 1000).toFixed(2)} л
          </div>
          <div className="mt-3">
            <Bar pct={d.percent} />
          </div>
        </>
      ) : (
        <Skeleton />
      )}
    </CardLink>
  );
}

function NutritionWidget() {
  const n = useNutritionDay();
  const d = n.data;
  const pct = d && d.goal.kcal > 0 ? (d.consumed.kcal / d.goal.kcal) * 100 : 0;
  return (
    <CardLink href="/nutrition" icon={Flame} title="Калории" hint="рацион">
      {d ? (
        <>
          <div className="mt-3 stat-value">
            {Math.round(d.consumed.kcal)}
            <span className="text-base font-medium text-ink-500"> / {d.goal.kcal} ккал</span>
          </div>
          <div className="mt-1 text-sm text-ink-500">
            {d.remainingKcal >= 0 ? `осталось ${Math.round(d.remainingKcal)}` : `профицит ${Math.abs(Math.round(d.remainingKcal))}`} ккал
          </div>
          <div className="mt-3">
            <Bar pct={pct} color="bg-good" />
          </div>
        </>
      ) : (
        <Skeleton />
      )}
    </CardLink>
  );
}

function WeightWidget() {
  const w = useWeightSummary(2);
  const d = w.data;
  return (
    <CardLink href="/weight" icon={Scale} title="Вес" hint="подробнее">
      {d ? (
        <>
          <div className="mt-3 stat-value">{d.latestKg != null ? `${d.latestKg} кг` : "—"}</div>
          <div className="mt-1 text-sm text-ink-500">
            {d.bmi != null ? `ИМТ ${d.bmi}` : "укажите рост для ИМТ"}
            {d.targetKg != null ? ` · цель ${d.targetKg} кг` : ""}
          </div>
        </>
      ) : (
        <Skeleton />
      )}
    </CardLink>
  );
}

function MacrosWidget() {
  const n = useNutritionDay();
  const d = n.data;
  const rows = d
    ? [
        { label: "Белки", v: d.consumed.protein, g: d.goal.protein, color: "bg-good" },
        { label: "Жиры", v: d.consumed.fat, g: d.goal.fat, color: "bg-warn" },
        { label: "Углеводы", v: d.consumed.carbs, g: d.goal.carbs, color: "bg-brand" },
      ]
    : [];
  return (
    <CardLink href="/nutrition" icon={Salad} title="Баланс БЖУ" hint="сегодня">
      {d ? (
        <div className="mt-3 space-y-2.5">
          {rows.map((r) => (
            <div key={r.label}>
              <div className="mb-1 flex justify-between text-xs">
                <span className="text-ink-400">{r.label}</span>
                <span className="text-ink-500">
                  {Math.round(r.v)} / {Math.round(r.g)} г
                </span>
              </div>
              <Bar pct={r.g > 0 ? (r.v / r.g) * 100 : 0} color={r.color} />
            </div>
          ))}
        </div>
      ) : (
        <Skeleton h="h-24" />
      )}
    </CardLink>
  );
}

function PressureWidget() {
  const p = usePressureSummary(30);
  const d = p.data;
  const cat = d?.category ? pressureCat(d.category) : undefined;
  return (
    <CardLink href="/pressure" icon={HeartPulse} title="Давление" hint="дневник">
      {d ? (
        <>
          <div className="mt-3 stat-value">
            {d.latest ? (
              <>
                {d.latest.systolic}
                <span className="text-ink-500">/</span>
                {d.latest.diastolic}
              </>
            ) : (
              "—"
            )}
          </div>
          {cat ? (
            <span className={`mt-2 inline-block rounded-full px-2 py-0.5 text-xs ${cat.badge}`}>{cat.label}</span>
          ) : (
            <div className="mt-1 text-sm text-ink-500">нет измерений</div>
          )}
          {d.averages.systolic != null && d.averages.diastolic != null ? (
            <div className="mt-2 text-sm text-ink-500">
              среднее {d.averages.systolic}/{d.averages.diastolic}
              {d.averages.pulse != null ? ` · ♥ ${d.averages.pulse}` : ""}
            </div>
          ) : null}
        </>
      ) : (
        <Skeleton />
      )}
    </CardLink>
  );
}

// ---- rich widgets ----

function TrainingWidget() {
  const my = useWorkouts({ scope: "mine", q: "", sort: "new", page: 1 });
  const items = my.data?.items ?? [];
  return (
    <div className="card h-full">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 font-semibold">
          <IconBadge Icon={ClipboardList} />
          Тренировки
        </h2>
        <Link href="/workouts" className="text-sm text-ink-500 hover:text-ink-100">
          все →
        </Link>
      </div>
      {my.data ? (
        items.length > 0 ? (
          <ul className="mt-3 space-y-2">
            {items.slice(0, 3).map((wk) => (
              <li key={wk.id}>
                <Link href={`/workouts/${wk.id}`} className="flex items-center gap-3 rounded-xl border border-ink-800 bg-ink-950/40 p-3 transition hover:border-brand/50">
                  <div className="h-10 w-10 shrink-0 overflow-hidden rounded-lg bg-ink-800">
                    {wk.imageUrl ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={wk.imageUrl} alt="" className="h-full w-full object-cover" />
                    ) : (
                      <div className="flex h-full items-center justify-center opacity-40">📋</div>
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">{wk.name}</p>
                    <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs">
                      <span className={`rounded px-1.5 py-0.5 ${DIFFICULTY_STYLE[wk.difficulty] ?? "bg-ink-800 text-ink-300"}`}>{label(DIFFICULTY_LABELS, wk.difficulty)}</span>
                      <span className="text-ink-500">{wk.exerciseCount} упр.</span>
                      {wk.ratingCount > 0 ? <span className="text-warn">★ {wk.ratingAvg.toFixed(1)}</span> : null}
                    </div>
                  </div>
                  <span className="shrink-0 text-ink-500">→</span>
                </Link>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-3 text-sm text-ink-500">Нет своих тренировок. Составьте первую программу.</p>
        )
      ) : (
        <Skeleton h="h-24" />
      )}
      <div className="mt-4 flex flex-wrap gap-2">
        <Link href="/workouts/new" className="btn-primary !py-2">
          + Составить тренировку
        </Link>
        <Link href="/exercises" className="btn-ghost !py-2">
          Каталог упражнений
        </Link>
      </div>
    </div>
  );
}

function MedsWidget() {
  const meds = useMeds();
  const take = useTakeIntake();
  const undo = useUndoIntake();
  const active = (meds.data?.items ?? []).filter((m) => m.status !== "finished");
  return (
    <div className="card h-full">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 font-semibold">
          <IconBadge Icon={Pill} />
          Таблетки и витамины
        </h2>
        <Link href="/meds" className="text-sm text-ink-500 hover:text-ink-100">
          все →
        </Link>
      </div>
      {meds.data ? (
        active.length > 0 ? (
          <div className="mt-3 space-y-4">
            {active.slice(0, 3).map((m) => (
              <div key={m.id} className="rounded-xl border border-ink-800 bg-ink-950/40 p-3">
                <PillBar med={m} onTake={() => take.mutate(m.id)} onUndo={() => undo.mutate(m.id)} busy={take.isPending || undo.isPending} />
              </div>
            ))}
          </div>
        ) : (
          <p className="mt-3 text-sm text-ink-500">Нет активных курсов. Добавьте таблетки/витамины.</p>
        )
      ) : (
        <Skeleton h="h-24" />
      )}
      <div className="mt-4">
        <Link href="/meds" className="btn-ghost !py-2">
          Управление курсами
        </Link>
      </div>
    </div>
  );
}

function QuickWidget() {
  const actions: { href: string; Icon: LucideIcon; label: string }[] = [
    { href: "/water", Icon: Droplet, label: "Добавить воду" },
    { href: "/nutrition", Icon: Flame, label: "Записать еду" },
    { href: "/meds", Icon: Pill, label: "Отметить приём" },
    { href: "/weight", Icon: Scale, label: "Записать вес" },
    { href: "/workouts/new", Icon: Dumbbell, label: "Новая тренировка" },
    { href: "/dishes/new", Icon: Utensils, label: "Новое блюдо" },
  ];
  return (
    <div className="card h-full">
      <h2 className="flex items-center gap-2 font-semibold">
        <IconBadge Icon={Zap} />
        Быстрые действия
      </h2>
      <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
        {actions.map((a) => (
          <Link
            key={a.href + a.label}
            href={a.href}
            className="flex flex-col items-center gap-1.5 rounded-xl border border-ink-800 bg-ink-950/40 p-3 text-center text-xs text-ink-300 transition hover:-translate-y-0.5 hover:border-brand/50 hover:text-ink-100"
          >
            <a.Icon size={20} className="text-brand" />
            {a.label}
          </Link>
        ))}
      </div>
    </div>
  );
}

// ---- registry ----

export const WIDGETS: WidgetMeta[] = [
  { id: "water", title: "Вода", icon: Droplet, span: 1, Component: WaterWidget },
  { id: "nutrition", title: "Калории", icon: Flame, span: 1, Component: NutritionWidget },
  { id: "weight", title: "Вес", icon: Scale, span: 1, Component: WeightWidget },
  { id: "pressure", title: "Давление", icon: HeartPulse, span: 1, Component: PressureWidget },
  { id: "macros", title: "Баланс БЖУ", icon: Salad, span: 1, Component: MacrosWidget },
  { id: "quick", title: "Быстрые действия", icon: Zap, span: 3, Component: QuickWidget },
  { id: "training", title: "Тренировки", icon: ClipboardList, span: 2, Component: TrainingWidget },
  { id: "meds", title: "Таблетки", icon: Pill, span: 2, Component: MedsWidget },
];

export const WIDGET_IDS = WIDGETS.map((w) => w.id);

export function widgetById(id: string): WidgetMeta | undefined {
  return WIDGETS.find((w) => w.id === id);
}

// spanClass maps a widget's column span to static Tailwind classes (kept literal
// so the JIT compiler includes them).
export function spanClass(span: 1 | 2 | 3): string {
  if (span === 3) return "sm:col-span-2 lg:col-span-3";
  if (span === 2) return "sm:col-span-2 lg:col-span-2";
  return "";
}
