"use client";

import { useState } from "react";
import { Bell, BellRing, Trash2 } from "lucide-react";
import { FastingRing } from "@/components/FastingRing";
import { browserTZ } from "@/lib/api";
import { isoToLocalInput, localToISO } from "@/lib/gtd";
import { usePush } from "@/hooks/usePush";
import {
  useDeleteFastSession,
  useFasting,
  useFastingHistory,
  useSetActiveStart,
  useSetFastingSchedule,
  useSetFastingSettings,
  useStartFast,
  useStopFast,
} from "@/hooks/useFasting";
import type { FastingSchedule, FastingSession } from "@/lib/types";

const PRESETS: { f: number; e: number; label: string }[] = [
  { f: 14, e: 10, label: "14:10" },
  { f: 16, e: 8, label: "16:8" },
  { f: 18, e: 6, label: "18:6" },
  { f: 20, e: 4, label: "20:4" },
  { f: 23, e: 1, label: "23:1 · OMAD" },
];

function fmtTime(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}
function fmtDay(iso: string | null): string {
  if (!iso) return "";
  return new Date(iso).toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit" });
}
function durationH(startedAt: string, endedAt: string | null): string {
  if (!endedAt) return "—";
  const h = (Date.parse(endedAt) - Date.parse(startedAt)) / 3_600_000;
  return `${h.toFixed(1)} ч`;
}

export default function FastingPage() {
  const state = useFasting();
  const start = useStartFast();
  const stop = useStopFast();
  const setStart = useSetActiveStart();
  const history = useFastingHistory();
  const del = useDeleteFastSession();

  const [editStart, setEditStart] = useState(false);

  const s = state.data;
  // A real, logged fast (has an id) — distinct from a schedule-derived fasting
  // phase shown before any fast is started. Only a real fast can be stopped.
  const hasActiveFast = Boolean(s?.activeId);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Интервальное голодание</h1>
        <p className="text-sm text-ink-500">Задайте окна голодания и еды — таймер покажет, сколько осталось.</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_1.1fr]">
        {/* ring + controls */}
        <div className="card flex flex-col items-center gap-4">
          {s ? (
            <FastingRing phase={s.phase} phaseStartAt={s.phaseStartAt} phaseEndAt={s.phaseEndAt} />
          ) : (
            <div className="h-60 w-60 animate-pulse rounded-full bg-ink-800/40" />
          )}

          {s ? (
            <div className="text-center text-sm text-ink-400">
              {s.phase === "fasting" && hasActiveFast ? (
                <>
                  Голодаю с <b>{fmtTime(s.phaseStartAt)}</b> · цель до <b>{fmtTime(s.phaseEndAt)}</b>{" "}
                  {fmtDay(s.phaseEndAt) !== fmtDay(s.phaseStartAt) ? `(${fmtDay(s.phaseEndAt)})` : ""}
                </>
              ) : s.phase === "fasting" ? (
                <>
                  По расписанию голодание с <b>{fmtTime(s.phaseStartAt)}</b> до <b>{fmtTime(s.phaseEndAt)}</b> · можно отметить кнопкой
                </>
              ) : s.phase === "eating" ? (
                <>
                  Окно еды до <b>{fmtTime(s.phaseEndAt)}</b> · потом голодание
                </>
              ) : (
                <>Пока нет активного голодания. Готовы начать?</>
              )}
            </div>
          ) : null}

          <div className="flex flex-wrap items-center justify-center gap-2">
            {hasActiveFast ? (
              <button onClick={() => stop.mutate(undefined)} disabled={stop.isPending} className="btn-primary">
                Завершить голодание
              </button>
            ) : (
              <button onClick={() => start.mutate(undefined)} disabled={start.isPending} className="btn-primary">
                Начать голодание
              </button>
            )}
            {hasActiveFast ? (
              <button onClick={() => setEditStart((v) => !v)} className="btn-ghost">
                Изменить время начала
              </button>
            ) : null}
          </div>

          {hasActiveFast && editStart ? (
            <div className="flex items-center gap-2">
              <input
                type="datetime-local"
                className="input !w-auto"
                defaultValue={isoToLocalInput(s?.phaseStartAt ?? null)}
                onChange={(e) => {
                  const iso = localToISO(e.target.value);
                  if (iso) setStart.mutate(iso, { onSuccess: () => setEditStart(false) });
                }}
              />
            </div>
          ) : null}
        </div>

        {/* protocol + stats */}
        <div className="space-y-4">
          <ProtocolCard current={s ? { f: s.fastingHours, e: s.eatingHours } : null} />

          {s ? (
            <div className="card grid grid-cols-3 gap-3 text-center">
              <Stat label="Всего постов" value={String(s.stats.totalFasts)} />
              <Stat label="Рекорд" value={`${s.stats.longestHours} ч`} />
              <Stat label="В среднем" value={`${s.stats.avgHours} ч`} />
            </div>
          ) : null}
        </div>
      </div>

      {/* daily schedule + notifications */}
      {s ? <ScheduleCard schedule={s.schedule} fastingHours={s.fastingHours} eatingHours={s.eatingHours} /> : null}

      {/* history */}
      <div className="card">
        <h2 className="mb-3 font-semibold">История</h2>
        {history.data && history.data.items.length > 0 ? (
          <ul className="space-y-2">
            {history.data.items.map((it) => (
              <HistoryRow key={it.id} it={it} onDelete={() => del.mutate(it.id)} />
            ))}
          </ul>
        ) : (
          <p className="text-sm text-ink-500">Пока нет завершённых голоданий.</p>
        )}
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-2xl font-black">{value}</div>
      <div className="text-xs text-ink-500">{label}</div>
    </div>
  );
}

function HistoryRow({ it, onDelete }: { it: FastingSession; onDelete: () => void }) {
  const ongoing = !it.endedAt;
  return (
    <li className="flex items-center justify-between rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2 text-sm">
      <div>
        <span className="text-ink-300">{fmtDay(it.startedAt)} </span>
        {fmtTime(it.startedAt)} → {ongoing ? <span className="text-brand">идёт…</span> : fmtTime(it.endedAt)}
      </div>
      <div className="flex items-center gap-3">
        <span className={`text-ink-400 ${!ongoing && Date.parse(it.endedAt!) - Date.parse(it.startedAt) >= it.goalHours * 3600_000 ? "text-good" : ""}`}>
          {durationH(it.startedAt, it.endedAt)} <span className="text-ink-600">/ цель {it.goalHours} ч</span>
        </span>
        <button onClick={onDelete} className="text-ink-500 hover:text-bad" aria-label="Удалить">
          <Trash2 size={15} />
        </button>
      </div>
    </li>
  );
}

function ProtocolCard({ current }: { current: { f: number; e: number } | null }) {
  const save = useSetFastingSettings();
  const [custom, setCustom] = useState(false);
  const [f, setF] = useState("16");
  const [e, setE] = useState("8");

  const isActive = (p: { f: number; e: number }) => current && current.f === p.f && current.e === p.e;

  return (
    <div className="card space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold">Протокол</h2>
        {current ? <span className="text-sm text-ink-500">сейчас {current.f}:{current.e}</span> : null}
      </div>
      <div className="flex flex-wrap gap-2">
        {PRESETS.map((p) => (
          <button
            key={p.label}
            onClick={() => save.mutate({ fastingHours: p.f, eatingHours: p.e })}
            disabled={save.isPending}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
              isActive(p) ? "bg-brand text-ink-950" : "bg-ink-800 text-ink-300 hover:bg-ink-700"
            }`}
          >
            {p.label}
          </button>
        ))}
        <button onClick={() => setCustom((v) => !v)} className="rounded-lg bg-ink-800 px-3 py-1.5 text-sm text-ink-300 hover:bg-ink-700">
          Своё
        </button>
      </div>
      {custom ? (
        <div className="flex flex-wrap items-end gap-2">
          <div>
            <label className="label">Голодание, ч</label>
            <input className="input !w-24" inputMode="decimal" value={f} onChange={(ev) => setF(ev.target.value)} />
          </div>
          <div>
            <label className="label">Еда, ч</label>
            <input className="input !w-24" inputMode="decimal" value={e} onChange={(ev) => setE(ev.target.value)} />
          </div>
          <button
            onClick={() => {
              const fh = parseFloat(f.replace(",", "."));
              const eh = parseFloat(e.replace(",", "."));
              if (fh > 0 && eh > 0) save.mutate({ fastingHours: fh, eatingHours: eh }, { onSuccess: () => setCustom(false) });
            }}
            className="btn-primary"
          >
            Сохранить
          </button>
        </div>
      ) : null}
      <p className="text-xs text-ink-500">Сумма окон обычно равна 24 ч (16:8, 18:6, 20:4). OMAD — один приём пищи в день.</p>
    </div>
  );
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

function Toggle({ label, checked, onChange, disabled }: { label: string; checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <label className={`flex items-center justify-between gap-3 rounded-lg border border-ink-800 bg-ink-950/40 px-3 py-2 ${disabled ? "opacity-50" : "cursor-pointer"}`}>
      <span className="text-sm">{label}</span>
      <input type="checkbox" className="h-4 w-4 accent-brand" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
    </label>
  );
}

function addMinutes(hhmm: string, mins: number): string | null {
  const [h, m] = hhmm.split(":").map((x) => parseInt(x, 10));
  if (Number.isNaN(h) || Number.isNaN(m)) return null;
  const total = ((h * 60 + m + Math.round(mins)) % (24 * 60) + 24 * 60) % (24 * 60);
  return `${pad2(Math.floor(total / 60))}:${pad2(total % 60)}`;
}

function ScheduleCard({ schedule, fastingHours, eatingHours }: { schedule: FastingSchedule; fastingHours: number; eatingHours: number }) {
  const save = useSetFastingSchedule();
  const push = usePush();

  const [enabled, setEnabled] = useState(schedule.enabled);
  const [time, setTime] = useState(`${pad2(schedule.eatStartHour)}:${pad2(schedule.eatStartMinute)}`);
  const [autoStart, setAutoStart] = useState(schedule.autoStart);
  const [notifyStart, setNotifyStart] = useState(schedule.notifyStart);
  const [notifyHourly, setNotifyHourly] = useState(schedule.notifyHourly);

  // The fast begins when the eating window closes (eat start + eating hours);
  // it ends one eating-window-start later (eat start + 24h → same clock time).
  const fastStart = addMinutes(time, eatingHours * 60);

  function onSave() {
    const [h, m] = time.split(":").map((x) => parseInt(x, 10));
    save.mutate({
      enabled,
      eatStartHour: Number.isNaN(h) ? 12 : h,
      eatStartMinute: Number.isNaN(m) ? 0 : m,
      timezone: browserTZ(),
      autoStart,
      notifyStart,
      notifyHourly,
    });
  }

  const notificationsReady = push.subscribed;

  return (
    <div className="card space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="font-semibold">Расписание и уведомления</h2>
          <p className="text-sm text-ink-500">Укажите, когда начинаете есть каждый день, — остальное посчитаем и напомним.</p>
        </div>
        <BellRing size={20} className="shrink-0 text-brand" />
      </div>

      <label className="flex items-center justify-between gap-3">
        <span className="font-medium">Ежедневное расписание</span>
        <input type="checkbox" className="h-5 w-5 accent-brand" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
      </label>

      {enabled ? (
        <div className="space-y-3">
          <div className="flex flex-wrap items-end gap-3">
            <div>
              <label className="label">Начало периода еды</label>
              <input type="time" className="input !w-auto" value={time} onChange={(e) => setTime(e.target.value)} />
            </div>
            <p className="pb-2 text-sm text-ink-500">
              Окно еды {time}–{fastStart ?? "—"} ({eatingHours} ч)
              {fastStart ? <> · голодание {fastingHours} ч начнётся в <b>{fastStart}</b></> : null}
            </p>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <Toggle label="Автостарт голодания" checked={autoStart} onChange={setAutoStart} />
            <Toggle label="Уведомление о старте" checked={notifyStart} onChange={setNotifyStart} />
            <Toggle label="Уведомления каждый час" checked={notifyHourly} onChange={setNotifyHourly} />
          </div>

          {/* push permission status */}
          {!push.loading && !notificationsReady ? (
            <div className="rounded-lg border border-warn/40 bg-warn/10 px-3 py-2 text-sm">
              {!push.supported ? (
                <span className="text-warn">Браузер не поддерживает push-уведомления.</span>
              ) : !push.secure ? (
                <span className="text-warn">Уведомления работают только по HTTPS.</span>
              ) : !push.configured ? (
                <span className="text-warn">Push-уведомления не настроены на сервере.</span>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="text-warn">Чтобы получать напоминания, включите уведомления в этом браузере.</span>
                  <button onClick={push.enable} disabled={push.loading} className="btn-primary !py-1.5">
                    <Bell size={15} /> Включить уведомления
                  </button>
                </div>
              )}
              {push.error ? <p className="mt-1 text-xs text-bad">{push.error}</p> : null}
            </div>
          ) : notificationsReady ? (
            <p className="flex items-center gap-1.5 text-sm text-good">
              <Bell size={14} /> Уведомления включены в этом браузере.
            </p>
          ) : null}
        </div>
      ) : null}

      <div className="flex items-center gap-3">
        <button onClick={onSave} disabled={save.isPending} className="btn-primary">
          {save.isPending ? "Сохранение…" : "Сохранить"}
        </button>
        {save.isSuccess && !save.isPending ? <span className="text-sm text-good">Сохранено</span> : null}
      </div>
    </div>
  );
}
