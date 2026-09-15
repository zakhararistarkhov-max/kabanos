"use client";

import { useEffect, useState } from "react";
import { Plus, X } from "lucide-react";
import {
  uploadGraphImage,
  useAddNode,
  useAddNodeImage,
  useDeleteNode,
  useRemoveNodeImage,
  useUpdateNode,
} from "@/hooks/useGtdGraph";
import type { GtdGraphNode, GtdNodeKind } from "@/lib/types";

const KIND_LABEL: Record<GtdNodeKind, string> = { task: "задача", project: "подпроект", note: "заметка" };

// NodeEditor is an in-app modal for creating and editing a graph node: title,
// deadline, note and multiple photos. For a new node it creates it on save and
// then unlocks the photo section (photos need the node id); editing an existing
// node updates it in place. onChanged refreshes the canvas.
export function NodeEditor({
  board,
  kind,
  node,
  onClose,
  onCreated,
  onChanged,
}: {
  board: string;
  kind: GtdNodeKind;
  node: GtdGraphNode | null; // null = creating
  onClose: () => void;
  onCreated: (id: string) => void;
  onChanged: () => void;
}) {
  const addNode = useAddNode();
  const update = useUpdateNode();
  const addImg = useAddNodeImage();
  const removeImg = useRemoveNodeImage();
  const deleteNode = useDeleteNode();

  const [label, setLabel] = useState(node?.label ?? "");
  const [deadline, setDeadline] = useState(node?.deadline ?? "");
  const [note, setNote] = useState(node?.note ?? "");
  const [uploading, setUploading] = useState(false);
  const [saving, setSaving] = useState(false);

  const editing = Boolean(node);

  // Reseed fields only when switching to a different node (so refreshes that
  // update photos don't clobber in-progress text edits).
  useEffect(() => {
    setLabel(node?.label ?? "");
    setDeadline(node?.deadline ?? "");
    setNote(node?.note ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [node?.id]);

  async function save() {
    if (!label.trim()) return;
    setSaving(true);
    try {
      if (!node) {
        const created = await addNode.mutateAsync({
          project: board,
          kind,
          title: label.trim(),
          deadline: deadline || undefined,
          x: 120 + Math.random() * 240,
          y: 120 + Math.random() * 160,
        });
        if (note.trim()) await update.mutateAsync({ id: created.id, note });
        onChanged();
        onCreated(created.id); // switch modal into edit mode (photos unlock)
      } else {
        await update.mutateAsync({ id: node.id, label: label.trim(), note, deadline });
        onChanged();
        onClose();
      }
    } catch {
      /* ignore */
    } finally {
      setSaving(false);
    }
  }

  async function onFiles(e: React.ChangeEvent<HTMLInputElement>) {
    if (!node) return;
    const files = Array.from(e.target.files ?? []);
    if (files.length === 0) return;
    setUploading(true);
    try {
      for (const f of files) {
        const key = await uploadGraphImage(f);
        await addImg.mutateAsync({ id: node.id, key });
      }
      onChanged();
    } catch {
      /* ignore */
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  }

  async function removeImage(key: string) {
    if (!node) return;
    await removeImg.mutateAsync({ id: node.id, key }).catch(() => undefined);
    onChanged();
  }

  async function remove() {
    if (!node) return;
    await deleteNode.mutateAsync(node.id).catch(() => undefined);
    onChanged();
    onClose();
  }

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4" onClick={onClose}>
      <div className="card w-full max-w-md space-y-4" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <h2 className="font-semibold">{editing ? "Редактирование ноды" : `Новая нода — ${KIND_LABEL[kind]}`}</h2>
          <button onClick={onClose} className="text-ink-500 hover:text-ink-100" aria-label="Закрыть">
            <X size={18} />
          </button>
        </div>

        <div>
          <label className="label">Название</label>
          <input
            className="input"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder={kind === "note" ? "Текст заметки" : "Название"}
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                save();
              }
            }}
          />
        </div>

        <div>
          <label className="label">Дедлайн (необязательно)</label>
          <input type="date" className="input" value={deadline} onChange={(e) => setDeadline(e.target.value)} />
        </div>

        <div>
          <label className="label">Заметка</label>
          <textarea className="input min-h-20" value={note} onChange={(e) => setNote(e.target.value)} placeholder="Текст…" />
        </div>

        <div>
          <label className="label">Фото</label>
          {editing ? (
            <div className="grid grid-cols-4 gap-2">
              {node!.imageUrls.map((url, i) => (
                <div key={url} className="relative overflow-hidden rounded-lg border border-ink-800">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={url} alt="" className="h-16 w-full object-cover" />
                  <button
                    onClick={() => removeImage(node!.imageKeys[i])}
                    className="absolute right-0.5 top-0.5 grid h-5 w-5 place-items-center rounded bg-ink-950/80 text-xs text-ink-300 hover:text-bad"
                    aria-label="Удалить фото"
                  >
                    ✕
                  </button>
                </div>
              ))}
              <label className="grid h-16 cursor-pointer place-items-center rounded-lg border border-dashed border-ink-700 text-ink-500 hover:border-brand/50">
                {uploading ? "…" : <Plus size={16} />}
                <input type="file" accept="image/png,image/jpeg,image/webp" multiple className="hidden" onChange={onFiles} disabled={uploading} />
              </label>
            </div>
          ) : (
            <p className="text-xs text-ink-500">Фото можно добавить после создания ноды.</p>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button onClick={save} disabled={saving || !label.trim()} className="btn-primary">
            {saving ? "Сохраняем…" : editing ? "Сохранить" : "Создать"}
          </button>
          {editing ? (
            <>
              <button onClick={onClose} className="btn-ghost">
                Готово
              </button>
              <button onClick={remove} className="btn-ghost ml-auto text-bad">
                Удалить ноду
              </button>
            </>
          ) : (
            <button onClick={onClose} className="btn-ghost">
              Отмена
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
