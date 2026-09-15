"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { GraphCanvas } from "@/components/gtd/GraphCanvas";
import { GraphsOverview } from "@/components/gtd/GraphsOverview";
import { useGtdProjects } from "@/hooks/useGtd";

export default function GtdGraphPage() {
  const { board } = useParams<{ board: string }>();
  const isRoot = board === "root";
  const projects = useGtdProjects();

  const title = isRoot
    ? "Карта проектов"
    : projects.data?.items.find((p) => p.id === board)?.title ?? "Граф проекта";

  return (
    <div className="space-y-4">
      <Link href="/gtd" className="text-sm text-ink-500 hover:text-ink-100">
        ← К GTD
      </Link>
      {isRoot ? (
        <div className="card">
          <GraphsOverview />
        </div>
      ) : null}
      <GraphCanvas board={isRoot ? "" : board} title={title} />
    </div>
  );
}
