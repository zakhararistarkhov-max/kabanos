"use client";

import { useState } from "react";
import Link from "next/link";
import { ChevronDown, ChevronRight, Network, Pencil, Plus, Trash2 } from "lucide-react";
import {
  useCreateProject,
  useDeleteProject,
  useGtdItems,
  useGtdProjects,
  useUpdateProject,
  type GtdProjectInput,
} from "@/hooks/useGtd";
import { ItemEditor } from "@/components/gtd/ItemEditor";
import { ItemRow } from "@/components/gtd/ItemRow";
import { BUCKET_ICON, BUCKET_LABELS } from "@/lib/gtd";
import type { GtdBucket, GtdItem, GtdProject, GtdProjectStatus } from "@/lib/types";

const SECTIONS: GtdBucket[] = ["next", "calendar", "waiting", "someday", "reference"];

// Step 3 — Organize: everything sorted into projects and bucket lists.
export function OrganizeTab() {
  const [creating, setCreating] = useState(false);
  const [editItem, setEditItem] = useState<GtdItem | null>(null);

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap gap-2">
        <button
          onClick={() => {
            setEditItem(null);
            setCreating((c) => !c);
          }}
          className={creating ? "btn-ghost" : "btn-primary"}
        >
          {creating ? "Закрыть" : (<><Plus size={16} /> Пункт</>)}
        </button>
      </div>

      {creating ? (
        <ItemEditor onDone={() => setCreating(false)} onCancel={() => setCreating(false)} />
      ) : null}
      {editItem ? (
        <ItemEditor initial={editItem} onDone={() => setEditItem(null)} onCancel={() => setEditItem(null)} />
      ) : null}

      <ProjectManager />

      {SECTIONS.map((b) => (
        <BucketSection key={b} bucket={b} onEdit={setEditItem} />
      ))}
    </div>
  );
}

function BucketSection({ bucket, onEdit }: { bucket: GtdBucket; onEdit: (it: GtdItem) => void }) {
  const [open, setOpen] = useState(true);
  const q = useGtdItems({ bucket, done: false });
  const items = q.data?.items ?? [];

  return (
    <div className="space-y-2">
      <button onClick={() => setOpen((o) => !o)} className="flex w-full items-center gap-2 text-left">
        {open ? <ChevronDown size={16} className="text-ink-500" /> : <ChevronRight size={16} className="text-ink-500" />}
        <span className="font-semibold">
          {BUCKET_ICON[bucket]} {BUCKET_LABELS[bucket]}
        </span>
        <span className="text-sm text-ink-500">{items.length}</span>
      </button>
      {open ? (
        items.length > 0 ? (
          <div className="space-y-2">
            {items.map((it) => (
              <ItemRow key={it.id} item={it} onEdit={onEdit} />
            ))}
          </div>
        ) : (
          <p className="px-2 text-sm text-ink-600">Пусто.</p>
        )
      ) : null}
    </div>
  );
}

const STATUS_LABELS: Record<GtdProjectStatus, string> = {
  active: "активный",
  someday: "когда‑нибудь",
  done: "завершён",
  dropped: "отменён",
};

function ProjectManager() {
  const projects = useGtdProjects();
  const create = useCreateProject();
  const update = useUpdateProject();
  const del = useDeleteProject();
  const [open, setOpen] = useState(true);
  const [form, setForm] = useState<{ id: string | null; input: GtdProjectInput } | null>(null);

  const items = projects.data?.items ?? [];

  function startNew() {
    setForm({ id: null, input: { title: "", outcome: "", status: "active" } });
  }

  async function save() {
    if (!form || !form.input.title.trim()) return;
    const input = { ...form.input, title: form.input.title.trim() };
    if (form.id) await update.mutateAsync({ id: form.id, input });
    else await create.mutateAsync(input);
    setForm(null);
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <button onClick={() => setOpen((o) => !o)} className="flex items-center gap-2 text-left">
          {open ? <ChevronDown size={16} className="text-ink-500" /> : <ChevronRight size={16} className="text-ink-500" />}
          <span className="font-semibold">🎯 Проекты</span>
          <span className="text-sm text-ink-500">{items.length}</span>
        </button>
        <button onClick={startNew} className="flex items-center gap-1 text-sm text-brand hover:text-brand-soft">
          <Plus size={15} /> Проект
        </button>
      </div>

      {form ? (
        <div className="card space-y-3">
          <div>
            <label className="label">Название проекта</label>
            <input
              className="input"
              value={form.input.title}
              onChange={(e) => setForm({ ...form, input: { ...form.input, title: e.target.value } })}
              placeholder="Напр.: Отпуск в июле"
              autoFocus
            />
          </div>
          <div>
            <label className="label">Желаемый результат (необязательно)</label>
            <input
              className="input"
              value={form.input.outcome ?? ""}
              onChange={(e) => setForm({ ...form, input: { ...form.input, outcome: e.target.value } })}
              placeholder="Как выглядит успешное завершение?"
            />
          </div>
          <div>
            <label className="label">Статус</label>
            <select
              className="input"
              value={form.input.status ?? "active"}
              onChange={(e) => setForm({ ...form, input: { ...form.input, status: e.target.value as GtdProjectStatus } })}
            >
              {(Object.keys(STATUS_LABELS) as GtdProjectStatus[]).map((s) => (
                <option key={s} value={s}>
                  {STATUS_LABELS[s]}
                </option>
              ))}
            </select>
          </div>
          <div className="flex gap-2">
            <button onClick={save} disabled={create.isPending || update.isPending} className="btn-primary">
              {form.id ? "Сохранить" : "Создать"}
            </button>
            <button onClick={() => setForm(null)} className="btn-ghost">
              Отмена
            </button>
          </div>
        </div>
      ) : null}

      {open ? (
        items.length > 0 ? (
          <div className="space-y-2">
            {items.map((p) => (
              <ProjectRow key={p.id} p={p} onEdit={() => setForm({ id: p.id, input: { title: p.title, outcome: p.outcome, notes: p.notes, status: p.status } })} onDelete={() => { if (confirm("Удалить проект? Действия останутся без проекта.")) del.mutate(p.id); }} />
            ))}
          </div>
        ) : (
          <p className="px-2 text-sm text-ink-600">Проектов пока нет. Проект — это всё, что требует больше одного шага.</p>
        )
      ) : null}
    </div>
  );
}

function ProjectRow({ p, onEdit, onDelete }: { p: GtdProject; onEdit: () => void; onDelete: () => void }) {
  const noNext = p.status === "active" && p.nextActions === 0;
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-ink-800 bg-ink-950/40 px-3 py-2.5">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className={`text-sm font-medium ${p.status === "done" || p.status === "dropped" ? "text-ink-600 line-through" : "text-ink-100"}`}>{p.title}</span>
          <span className="rounded-full bg-ink-800 px-2 py-0.5 text-[11px] text-ink-400">{STATUS_LABELS[p.status]}</span>
        </div>
        <div className="mt-0.5 text-xs text-ink-500">
          действий: {p.openActions}
          {noNext ? <span className="text-warn"> · нет следующего действия</span> : null}
          {p.outcome ? <span> · {p.outcome}</span> : null}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2.5">
        <Link href={`/gtd/graph/${p.id}`} className="flex items-center gap-1 text-xs text-brand hover:text-brand-soft" title="Открыть граф проекта">
          <Network size={14} /> Граф
        </Link>
        <button onClick={onEdit} className="text-ink-500 hover:text-ink-100" aria-label="Изменить">
          <Pencil size={15} />
        </button>
        <button onClick={onDelete} className="text-ink-500 hover:text-bad" aria-label="Удалить">
          <Trash2 size={15} />
        </button>
      </div>
    </div>
  );
}
