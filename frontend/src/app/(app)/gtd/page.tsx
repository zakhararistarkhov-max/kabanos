"use client";

import { useState } from "react";
import Link from "next/link";
import { Network } from "lucide-react";
import { CaptureTab } from "@/components/gtd/CaptureTab";
import { ClarifyTab } from "@/components/gtd/ClarifyTab";
import { OrganizeTab } from "@/components/gtd/OrganizeTab";
import { ReviewTab } from "@/components/gtd/ReviewTab";
import { EngageTab } from "@/components/gtd/EngageTab";
import { useGtdReview } from "@/hooks/useGtd";

type Tab = "capture" | "clarify" | "organize" | "review" | "engage";

const TABS: { id: Tab; step: number; label: string; hint: string }[] = [
  { id: "capture", step: 1, label: "Сбор", hint: "Записать всё" },
  { id: "clarify", step: 2, label: "Обработка", hint: "Что это и куда" },
  { id: "organize", step: 3, label: "Организация", hint: "Списки и проекты" },
  { id: "review", step: 4, label: "Обзор", hint: "Еженедельный обзор" },
  { id: "engage", step: 5, label: "Выполнение", hint: "Делать" },
];

export default function GtdPage() {
  const [tab, setTab] = useState<Tab>("capture");
  const review = useGtdReview();
  const inboxCount = review.data?.inboxCount ?? 0;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">GTD — привести дела в порядок</h1>
          <p className="text-sm text-ink-500">Пять шагов: собрать всё, обработать, организовать, регулярно пересматривать и спокойно делать.</p>
        </div>
        <Link href="/gtd/graph/root" className="btn-ghost shrink-0">
          <Network size={16} /> Карта проектов
        </Link>
      </div>

      {/* step tabs */}
      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {TABS.map((t) => {
          const active = tab === t.id;
          return (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`flex shrink-0 items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium transition ${
                active ? "bg-brand text-ink-950" : "bg-ink-800/60 text-ink-300 hover:bg-ink-800"
              }`}
              title={t.hint}
            >
              <span className={`grid h-5 w-5 place-items-center rounded-full text-xs ${active ? "bg-ink-950/20" : "bg-ink-900/60"}`}>
                {t.step}
              </span>
              {t.label}
              {t.id === "capture" && inboxCount > 0 ? (
                <span className={`rounded-full px-1.5 text-xs ${active ? "bg-ink-950/20" : "bg-brand/20 text-brand"}`}>{inboxCount}</span>
              ) : null}
            </button>
          );
        })}
      </div>

      {tab === "capture" ? <CaptureTab onGoClarify={() => setTab("clarify")} /> : null}
      {tab === "clarify" ? <ClarifyTab /> : null}
      {tab === "organize" ? <OrganizeTab /> : null}
      {tab === "review" ? <ReviewTab onJump={(t) => setTab(t as Tab)} /> : null}
      {tab === "engage" ? <EngageTab /> : null}
    </div>
  );
}
