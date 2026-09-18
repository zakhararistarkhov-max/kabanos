"use client";

import Link from "next/link";
import { ArrowLeft, Film } from "lucide-react";
import { TaskFeed } from "@/components/gtd/TaskFeed";

export default function GtdFeedPage() {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Film size={20} className="text-brand" />
          <h1 className="text-xl font-bold">Лента дел</h1>
        </div>
        <Link href="/gtd" className="btn-ghost !py-1.5 text-sm">
          <ArrowLeft size={15} /> К GTD
        </Link>
      </div>
      <TaskFeed />
    </div>
  );
}
