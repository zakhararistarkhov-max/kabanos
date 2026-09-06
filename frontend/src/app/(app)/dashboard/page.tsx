"use client";

import { useState } from "react";
import { useSession } from "@/hooks/useSession";
import { useDashboardLayout } from "@/hooks/useDashboardLayout";
import { WIDGET_IDS, spanClass, widgetById } from "@/components/dashboard/widgets";
import { KabanosMark } from "@/components/Logo";

export default function DashboardPage() {
  const { data: session } = useSession();
  const { mounted, enabled, hidden, toggle, move, reset } = useDashboardLayout(WIDGET_IDS);
  const [customizing, setCustomizing] = useState(false);

  const name = session?.user?.displayName || "друг";

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span className="animate-float">
            <KabanosMark size={46} />
          </span>
          <div>
            <h1 className="text-2xl font-bold">
              Привет, <span className="text-gradient">{name}</span> 👋
            </h1>
            <p className="text-sm text-ink-500">Коротко о том, как проходит день.</p>
          </div>
        </div>
        <button onClick={() => setCustomizing((s) => !s)} className={customizing ? "btn-primary" : "btn-ghost"}>
          {customizing ? "Готово" : "🧩 Настроить"}
        </button>
      </div>

      {customizing && (
        <div className="card rise-in space-y-5">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="font-semibold">Кубики дашборда</h2>
              <p className="text-sm text-ink-500">Выберите, что показывать, и задайте порядок.</p>
            </div>
            <button onClick={reset} className="text-sm text-ink-500 hover:text-ink-100">
              Сбросить
            </button>
          </div>

          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-500">Показаны</p>
            {enabled.length > 0 ? (
              <ul className="space-y-2">
                {enabled.map((id, i) => {
                  const w = widgetById(id);
                  if (!w) return null;
                  return (
                    <li key={id} className="flex items-center gap-2 rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2">
                      <span className="grid h-7 w-7 place-items-center rounded-lg bg-ink-800">{w.icon}</span>
                      <span className="flex-1 text-sm font-medium">{w.title}</span>
                      <button onClick={() => move(id, -1)} disabled={i === 0} className="rounded px-1.5 text-ink-500 hover:text-ink-100 disabled:opacity-30">
                        ↑
                      </button>
                      <button onClick={() => move(id, 1)} disabled={i === enabled.length - 1} className="rounded px-1.5 text-ink-500 hover:text-ink-100 disabled:opacity-30">
                        ↓
                      </button>
                      <button onClick={() => toggle(id)} className="ml-1 text-xs text-ink-500 hover:text-bad">
                        Скрыть
                      </button>
                    </li>
                  );
                })}
              </ul>
            ) : (
              <p className="text-sm text-ink-500">Ничего не выбрано.</p>
            )}
          </div>

          {hidden.length > 0 && (
            <div>
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-500">Скрытые</p>
              <div className="flex flex-wrap gap-2">
                {hidden.map((id) => {
                  const w = widgetById(id);
                  if (!w) return null;
                  return (
                    <button key={id} onClick={() => toggle(id)} className="chip transition hover:border-brand/50 hover:text-ink-100">
                      + {w.icon} {w.title}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}

      {!mounted ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="card h-40 animate-pulse bg-ink-800/40" />
          ))}
        </div>
      ) : enabled.length === 0 ? (
        <div className="card text-center text-ink-500">
          Все кубики скрыты. Нажмите «Настроить» и добавьте нужные.
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {enabled.map((id, i) => {
            const w = widgetById(id);
            if (!w) return null;
            const W = w.Component;
            return (
              <div key={id} className={`rise-in ${spanClass(w.span)}`} style={{ animationDelay: `${Math.min(i, 8) * 60}ms` }}>
                <W />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
