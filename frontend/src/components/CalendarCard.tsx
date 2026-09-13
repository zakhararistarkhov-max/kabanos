"use client";

import { useState } from "react";
import { CalendarClock, ExternalLink } from "lucide-react";
import { Field } from "@/components/Field";
import { ApiRequestError } from "@/lib/api";
import {
  useCalendarStatus,
  useCalendars,
  useConnectCalendar,
  useDisconnectCalendar,
  useSelectCalendar,
  useSyncCalendar,
} from "@/hooks/useCalendar";

const APP_PASSWORD_URL = "https://id.yandex.ru/security/app-passwords";

export function CalendarCard() {
  const status = useCalendarStatus();
  const connect = useConnectCalendar();
  const disconnect = useDisconnectCalendar();
  const sync = useSyncCalendar();

  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [syncMsg, setSyncMsg] = useState<string | null>(null);

  const s = status.data;

  async function onConnect(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await connect.mutateAsync({ login: login.trim(), password: password.trim() });
      setPassword("");
      // Kick off an initial sync so events appear right away.
      runSync();
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Не удалось подключиться");
    }
  }

  async function runSync() {
    setSyncMsg(null);
    setError(null);
    try {
      const r = await sync.mutateAsync();
      setSyncMsg(`Синхронизировано: ↑${r.pushed} ↓${r.pulled}${r.deleted ? ` ✕${r.deleted}` : ""}`);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Ошибка синхронизации");
    }
  }

  return (
    <div className="card space-y-3">
      <div className="flex items-center gap-2">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-ink-800 text-brand">
          <CalendarClock size={17} />
        </span>
        <div>
          <h2 className="font-semibold">Яндекс.Календарь</h2>
          <p className="text-sm text-ink-500">Двусторонняя синхронизация раздела «Календарь» (GTD) с Яндексом по CalDAV.</p>
        </div>
      </div>

      {status.isLoading ? (
        <div className="h-16 animate-pulse rounded-lg bg-ink-800/40" />
      ) : s?.connected ? (
        <ConnectedView s={s} onSync={runSync} onDisconnect={() => disconnect.mutate()} syncing={sync.isPending} />
      ) : (
        <form onSubmit={onConnect} className="space-y-3">
          <Field label="Логин Яндекса (email)" name="login" value={login} onChange={(e) => setLogin(e.target.value)} placeholder="you@yandex.ru" />
          <div>
            <label className="label">Пароль приложения</label>
            <input
              type="password"
              className="input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="пароль приложения, не основной пароль"
              autoComplete="off"
            />
            <a href={APP_PASSWORD_URL} target="_blank" rel="noreferrer" className="mt-1 inline-flex items-center gap-1 text-xs text-brand hover:text-brand-soft">
              Создать пароль приложения в Яндекс ID <ExternalLink size={12} />
            </a>
          </div>
          <p className="rounded-lg bg-ink-800/50 px-3 py-2 text-xs text-ink-400">
            В Яндекс ID → «Пароли приложений» создайте пароль для <b>Календаря (CalDAV)</b> и вставьте его сюда. Обычный пароль от аккаунта не подойдёт.
          </p>
          {error ? <p className="field-error">{error}</p> : null}
          <button type="submit" disabled={connect.isPending || !login.trim() || !password.trim()} className="btn-primary">
            {connect.isPending ? "Подключаем…" : "Подключить"}
          </button>
        </form>
      )}

      {syncMsg ? <p className="text-sm text-good">{syncMsg}</p> : null}
      {error && s?.connected ? <p className="field-error">{error}</p> : null}
    </div>
  );
}

function ConnectedView({
  s,
  onSync,
  onDisconnect,
  syncing,
}: {
  s: import("@/lib/types").CalendarStatus;
  onSync: () => void;
  onDisconnect: () => void;
  syncing: boolean;
}) {
  const cals = useCalendars(true);
  const select = useSelectCalendar();

  return (
    <div className="space-y-3">
      <div className="rounded-lg bg-ink-800/40 px-3 py-2 text-sm">
        <div>
          Подключено: <span className="font-medium">{s.login}</span>
        </div>
        <div className="text-ink-500">
          Календарь: {s.calendarName || "—"}
          {s.lastSyncAt ? ` · синхр. ${new Date(s.lastSyncAt).toLocaleString("ru-RU")}` : " · ещё не синхронизировано"}
        </div>
        {s.lastError ? <div className="mt-1 text-bad">Последняя ошибка: {s.lastError}</div> : null}
      </div>

      <div>
        <label className="label">Календарь для синхронизации</label>
        <select
          className="input"
          value={s.calendarUrl}
          onChange={(e) => {
            const opt = (cals.data?.items ?? []).find((c) => c.url === e.target.value);
            if (opt) select.mutate({ url: opt.url, name: opt.name });
          }}
          disabled={cals.isLoading || select.isPending}
        >
          {/* keep the current one selectable even before the list loads */}
          {!(cals.data?.items ?? []).some((c) => c.url === s.calendarUrl) ? (
            <option value={s.calendarUrl}>{s.calendarName || "текущий"}</option>
          ) : null}
          {(cals.data?.items ?? []).map((c) => (
            <option key={c.url} value={c.url}>
              {c.name}
            </option>
          ))}
        </select>
        {cals.isError ? <p className="field-error">Не удалось получить список календарей — проверьте подключение.</p> : null}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <button onClick={onSync} disabled={syncing} className="btn-primary">
          {syncing ? "Синхронизируем…" : "Синхронизировать сейчас"}
        </button>
        <button onClick={onDisconnect} className="btn-ghost text-bad">
          Отключить
        </button>
      </div>
      <p className="text-xs text-ink-500">Автосинхронизация выполняется в фоне примерно раз в 10 минут. Пункты раздела «Календарь» уходят в Яндекс, события Яндекса появляются там же.</p>
    </div>
  );
}
