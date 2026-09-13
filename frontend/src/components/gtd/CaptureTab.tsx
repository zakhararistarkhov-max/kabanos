"use client";

import { useState } from "react";
import { ArrowRight } from "lucide-react";
import { useCaptureItem, useGtdItems } from "@/hooks/useGtd";
import { ItemRow } from "@/components/gtd/ItemRow";

// Step 1 — Capture: dump anything on your mind into the inbox, fast.
export function CaptureTab({ onGoClarify }: { onGoClarify: () => void }) {
  const [title, setTitle] = useState("");
  const capture = useCaptureItem();
  const inbox = useGtdItems({ bucket: "inbox", done: false });
  const items = inbox.data?.items ?? [];

  function add() {
    const t = title.trim();
    if (!t) return;
    capture.mutate({ title: t, bucket: "inbox" });
    setTitle("");
  }

  return (
    <div className="space-y-5">
      <div className="card space-y-3">
        <div>
          <h2 className="font-semibold">Собрать всё в одном месте</h2>
          <p className="text-sm text-ink-500">Записывайте любую мысль, задачу или идею — не держите в голове. Разберёте позже.</p>
        </div>
        <div className="flex gap-2">
          <input
            className="input"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                add();
              }
            }}
            placeholder="Что у вас на уме?"
            autoFocus
          />
          <button onClick={add} disabled={capture.isPending || !title.trim()} className="btn-primary shrink-0">
            Добавить
          </button>
        </div>
        <p className="text-xs text-ink-500">Enter — добавить и продолжить. Пишите коротко, детали добавите на шаге «Обработка».</p>
      </div>

      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-ink-400">
          Во входящих: {items.length}
        </h3>
        {items.length > 0 ? (
          <button onClick={onGoClarify} className="flex items-center gap-1 text-sm text-brand hover:text-brand-soft">
            Обработать <ArrowRight size={15} />
          </button>
        ) : null}
      </div>

      {inbox.isLoading ? (
        <div className="card h-20 animate-pulse bg-ink-800/40" />
      ) : items.length > 0 ? (
        <div className="space-y-2">
          {items.map((it) => (
            <ItemRow key={it.id} item={it} />
          ))}
        </div>
      ) : (
        <div className="card text-center text-sm text-ink-500">Входящие пусты — отличная работа! 🎉</div>
      )}
    </div>
  );
}
