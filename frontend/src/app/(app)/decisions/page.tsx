"use client";

import { useState } from "react";
import { CalendarClock, Check, ChevronDown, ChevronRight, Lightbulb, Plus, ScrollText, Trash2 } from "lucide-react";
import { todayISO } from "@/lib/api";
import { useCreateDecision, useDecisions, useDeleteDecision, useReviewDecision } from "@/hooks/useDecisions";
import type { Decision } from "@/lib/types";

const RATING_LABEL: Record<number, string> = { 1: "ошибочное", 2: "скорее неверное", 3: "спорное", 4: "скорее верное", 5: "верное" };

function shortDate(iso: string): string {
  return new Date(iso + "T00:00:00").toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit", year: "2-digit" });
}

function Scale({ value, onChange }: { value: number | null; onChange: (v: number) => void }) {
  return (
    <div className="flex gap-1">
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          onClick={() => onChange(n)}
          className={`h-8 w-8 rounded-lg text-sm font-medium transition ${value === n ? "bg-brand text-ink-950" : "bg-ink-800 text-ink-400 hover:bg-ink-700"}`}
        >
          {n}
        </button>
      ))}
    </div>
  );
}

export default function DecisionsPage() {
  const list = useDecisions();
  const items = list.data?.items ?? [];
  const due = items.filter((d) => d.status === "open" && d.due);
  const open = items.filter((d) => d.status === "open" && !d.due);
  const reviewed = items.filter((d) => d.status === "reviewed");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Дневник решений</h1>
        <p className="text-sm text-ink-500">Записывайте контекст и решение, а через время возвращайтесь: что вышло, насколько были правы и какие выводы.</p>
      </div>

      <NewDecision />

      {list.isLoading ? (
        <div className="h-24 animate-pulse rounded-2xl bg-ink-800/40" />
      ) : items.length === 0 ? (
        <p className="rounded-2xl border border-dashed border-ink-800 px-3 py-8 text-center text-sm text-ink-500">
          Пока пусто. Запишите первое решение выше.
        </p>
      ) : (
        <div className="space-y-6">
          {due.length > 0 ? (
            <Section title="Пора оценить" accent="text-warn" count={due.length}>
              {due.map((d) => <DecisionCard key={d.id} d={d} />)}
            </Section>
          ) : null}
          {open.length > 0 ? (
            <Section title="В ожидании итога" count={open.length}>
              {open.map((d) => <DecisionCard key={d.id} d={d} />)}
            </Section>
          ) : null}
          {reviewed.length > 0 ? (
            <Section title="Оценённые" count={reviewed.length}>
              {reviewed.map((d) => <DecisionCard key={d.id} d={d} />)}
            </Section>
          ) : null}
        </div>
      )}
    </div>
  );
}

function Section({ title, accent, count, children }: { title: string; accent?: string; count: number; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <h2 className={`text-sm font-semibold ${accent ?? "text-ink-400"}`}>{title} · {count}</h2>
      <div className="space-y-2">{children}</div>
    </div>
  );
}

function NewDecision() {
  const create = useCreateDecision();
  const [openForm, setOpenForm] = useState(false);
  const [title, setTitle] = useState("");
  const [context, setContext] = useState("");
  const [decision, setDecision] = useState("");
  const [expected, setExpected] = useState("");
  const [confidence, setConfidence] = useState<number | null>(null);
  const [reviewAt, setReviewAt] = useState("");

  function reset() {
    setTitle(""); setContext(""); setDecision(""); setExpected(""); setConfidence(null); setReviewAt("");
  }
  function submit() {
    if (!title.trim()) return;
    create.mutate(
      { title: title.trim(), context, decision, expected, confidence, decidedOn: todayISO(), reviewAt: reviewAt || null },
      { onSuccess: () => { reset(); setOpenForm(false); } },
    );
  }

  if (!openForm) {
    return (
      <button onClick={() => setOpenForm(true)} className="btn-primary">
        <Plus size={16} /> Записать решение
      </button>
    );
  }

  return (
    <div className="card space-y-3">
      <input className="input" placeholder="О чём решение? (тема)" value={title} maxLength={200} onChange={(e) => setTitle(e.target.value)} autoFocus />
      <div>
        <label className="label">Контекст — что происходит, какие варианты</label>
        <textarea className="input min-h-20" value={context} maxLength={5000} onChange={(e) => setContext(e.target.value)} />
      </div>
      <div>
        <label className="label">Решение — что я решил сделать</label>
        <textarea className="input min-h-16" value={decision} maxLength={5000} onChange={(e) => setDecision(e.target.value)} />
      </div>
      <div>
        <label className="label">Прогноз — чего жду (необязательно)</label>
        <textarea className="input min-h-16" value={expected} maxLength={5000} onChange={(e) => setExpected(e.target.value)} />
      </div>
      <div className="flex flex-wrap items-end gap-4">
        <div>
          <label className="label">Уверенность</label>
          <Scale value={confidence} onChange={setConfidence} />
        </div>
        <div>
          <label className="label">Вернуться к решению</label>
          <input type="date" className="input !w-auto" min={todayISO()} value={reviewAt} onChange={(e) => setReviewAt(e.target.value)} />
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button onClick={submit} disabled={create.isPending || !title.trim()} className="btn-primary">Сохранить</button>
        <button onClick={() => { reset(); setOpenForm(false); }} className="btn-ghost">Отмена</button>
        {reviewAt ? <span className="text-xs text-ink-500">Напомним {shortDate(reviewAt)}</span> : <span className="text-xs text-ink-500">Без даты — вернётесь вручную</span>}
      </div>
    </div>
  );
}

function DecisionCard({ d }: { d: Decision }) {
  const del = useDeleteDecision();
  const [expanded, setExpanded] = useState(d.due);
  const [reviewing, setReviewing] = useState(false);
  const reviewed = d.status === "reviewed";

  return (
    <div className={`rounded-2xl border bg-ink-950/40 ${d.due ? "border-warn/50" : reviewed ? "border-ink-800" : "border-ink-800"}`}>
      <div className="flex items-start gap-2 p-3">
        <button onClick={() => setExpanded((v) => !v)} className="mt-0.5 text-ink-500 hover:text-ink-100">
          {expanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
        </button>
        <button onClick={() => setExpanded((v) => !v)} className="min-w-0 flex-1 text-left">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">{d.title}</span>
            {reviewed && d.rating ? <span className="rounded-full bg-ink-800 px-2 py-0.5 text-xs">итог: {d.rating}/5 · {RATING_LABEL[d.rating]}</span> : null}
            {d.due ? <span className="rounded-full bg-warn/20 px-2 py-0.5 text-xs text-warn">пора оценить</span> : null}
          </div>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs text-ink-500">
            <span>решено {shortDate(d.decidedOn)}</span>
            {!reviewed && d.reviewAt ? <span className="inline-flex items-center gap-1"><CalendarClock size={12} /> вернуться {shortDate(d.reviewAt)}</span> : null}
            {reviewed && d.reviewedAt ? <span>оценено {new Date(d.reviewedAt).toLocaleDateString("ru-RU")}</span> : null}
            {d.confidence ? <span>уверенность {d.confidence}/5</span> : null}
          </div>
        </button>
        <button
          onClick={() => { if (confirm("Удалить запись?")) del.mutate(d.id); }}
          className="text-ink-600 transition hover:text-bad"
          aria-label="Удалить"
        >
          <Trash2 size={15} />
        </button>
      </div>

      {expanded ? (
        <div className="space-y-3 border-t border-ink-800 p-3 text-sm">
          {d.context ? <Field label="Контекст" value={d.context} /> : null}
          {d.decision ? <Field label="Решение" value={d.decision} /> : null}
          {d.expected ? <Field label="Прогноз" value={d.expected} /> : null}

          {reviewed ? (
            <div className="space-y-3 rounded-xl border border-good/30 bg-good/5 p-3">
              {d.result ? <Field label="Что вышло" value={d.result} /> : null}
              {d.rating ? <p><span className="text-ink-500">Правильность: </span>{d.rating}/5 — {RATING_LABEL[d.rating]}</p> : null}
              {d.lessons ? <Field label="Выводы на будущее" value={d.lessons} icon /> : null}
            </div>
          ) : reviewing ? (
            <ReviewForm d={d} onDone={() => setReviewing(false)} />
          ) : (
            <button onClick={() => setReviewing(true)} className="btn-primary !py-1.5">
              <Check size={15} /> Оценить решение
            </button>
          )}
        </div>
      ) : null}
    </div>
  );
}

function Field({ label, value, icon }: { label: string; value: string; icon?: boolean }) {
  return (
    <div>
      <p className="flex items-center gap-1 text-xs font-medium text-ink-500">{icon ? <Lightbulb size={12} /> : null} {label}</p>
      <p className="whitespace-pre-wrap text-ink-200">{value}</p>
    </div>
  );
}

function ReviewForm({ d, onDone }: { d: Decision; onDone: () => void }) {
  const review = useReviewDecision();
  const [result, setResult] = useState("");
  const [rating, setRating] = useState<number | null>(null);
  const [lessons, setLessons] = useState("");

  function submit() {
    review.mutate(
      { id: d.id, result, rating, lessons },
      { onSuccess: onDone },
    );
  }

  return (
    <div className="space-y-3 rounded-xl border border-ink-800 bg-ink-950/40 p-3">
      <h3 className="flex items-center gap-1.5 font-semibold"><ScrollText size={15} /> Итог</h3>
      <div>
        <label className="label">Что в итоге вышло</label>
        <textarea className="input min-h-16" value={result} maxLength={5000} onChange={(e) => setResult(e.target.value)} autoFocus />
      </div>
      <div>
        <label className="label">Насколько решение оказалось правильным</label>
        <Scale value={rating} onChange={setRating} />
        {rating ? <span className="ml-1 text-xs text-ink-500">{RATING_LABEL[rating]}</span> : null}
      </div>
      <div>
        <label className="label">Выводы на будущее</label>
        <textarea className="input min-h-16" value={lessons} maxLength={5000} onChange={(e) => setLessons(e.target.value)} placeholder="Что учесть в следующий раз" />
      </div>
      <div className="flex items-center gap-2">
        <button onClick={submit} disabled={review.isPending} className="btn-primary !py-1.5">Сохранить оценку</button>
        <button onClick={onDone} className="btn-ghost !py-1.5">Отмена</button>
      </div>
    </div>
  );
}
