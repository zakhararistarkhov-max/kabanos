"use client";

import Link from "next/link";
import { useSession } from "@/hooks/useSession";
import { useWaterDay } from "@/hooks/useWater";
import { useWeightSummary } from "@/hooks/useWeight";
import { useNutritionDay } from "@/hooks/useNutrition";

export default function DashboardPage() {
  const { data: session } = useSession();
  const water = useWaterDay();
  const weight = useWeightSummary(2);
  const nutrition = useNutritionDay();

  const name = session?.user?.displayName || "друг";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Привет, {name} 👋</h1>
        <p className="text-sm text-ink-500">Коротко о том, как проходит день.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <Link href="/water" className="card transition hover:border-brand/50">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">💧 Вода</h2>
            <span className="text-sm text-ink-500">сегодня →</span>
          </div>
          {water.data ? (
            <>
              <div className="mt-3 text-3xl font-black">{Math.round(water.data.percent)}%</div>
              <div className="mt-1 text-sm text-ink-500">
                {(water.data.consumedMl / 1000).toFixed(2)} из {(water.data.goalMl / 1000).toFixed(2)} л
              </div>
              <div className="mt-3 h-2 w-full overflow-hidden rounded-full bg-ink-800">
                <div className="h-full rounded-full bg-brand transition-all" style={{ width: `${Math.min(100, water.data.percent)}%` }} />
              </div>
            </>
          ) : (
            <div className="mt-3 h-16 animate-pulse rounded-xl bg-ink-800/50" />
          )}
        </Link>

        <Link href="/nutrition" className="card transition hover:border-brand/50">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">🍎 Калории</h2>
            <span className="text-sm text-ink-500">рацион →</span>
          </div>
          {nutrition.data ? (
            <>
              <div className="mt-3 text-3xl font-black">
                {Math.round(nutrition.data.consumed.kcal)}
                <span className="text-base font-medium text-ink-500"> / {nutrition.data.goal.kcal} ккал</span>
              </div>
              <div className="mt-1 text-sm text-ink-500">
                {nutrition.data.remainingKcal >= 0
                  ? `осталось ${Math.round(nutrition.data.remainingKcal)} ккал`
                  : `профицит ${Math.abs(Math.round(nutrition.data.remainingKcal))} ккал`}
                {nutrition.data.burnedKcal > 0 ? ` · сожжено ${Math.round(nutrition.data.burnedKcal)}` : ""}
              </div>
              <div className="mt-3 h-2 w-full overflow-hidden rounded-full bg-ink-800">
                <div
                  className="h-full rounded-full bg-good transition-all"
                  style={{ width: `${Math.min(100, nutrition.data.goal.kcal > 0 ? (nutrition.data.consumed.kcal / nutrition.data.goal.kcal) * 100 : 0)}%` }}
                />
              </div>
            </>
          ) : (
            <div className="mt-3 h-16 animate-pulse rounded-xl bg-ink-800/50" />
          )}
        </Link>

        <Link href="/weight" className="card transition hover:border-brand/50">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">⚖️ Вес</h2>
            <span className="text-sm text-ink-500">подробнее →</span>
          </div>
          {weight.data ? (
            <>
              <div className="mt-3 text-3xl font-black">
                {weight.data.latestKg != null ? `${weight.data.latestKg} кг` : "—"}
              </div>
              <div className="mt-1 text-sm text-ink-500">
                {weight.data.bmi != null ? `ИМТ ${weight.data.bmi}` : "укажите рост для расчёта ИМТ"}
                {weight.data.targetKg != null ? ` · цель ${weight.data.targetKg} кг` : ""}
              </div>
            </>
          ) : (
            <div className="mt-3 h-16 animate-pulse rounded-xl bg-ink-800/50" />
          )}
        </Link>
      </div>

      <div className="card">
        <h2 className="font-semibold">Скоро здесь появятся</h2>
        <p className="mt-1 text-sm text-ink-500">
          Калории и блюда, таблетки/витамины, тренировки и упражнения, планировщик дня и Telegram-дайджест. Ядро уже
          готово — эти разделы подключаются поверх той же архитектуры.
        </p>
        <div className="mt-3 flex flex-wrap gap-2 text-xs text-ink-500">
          {["💊 Таблетки", "🏋️ Тренировки", "📅 Планировщик", "🤖 Telegram"].map((t) => (
            <span key={t} className="rounded-full border border-ink-800 px-3 py-1">
              {t}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
