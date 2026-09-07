"use client";

import { useState } from "react";
import { ChevronLeft, ChevronRight, ImagePlus, Trash2 } from "lucide-react";
import { todayISO } from "@/lib/api";
import {
  useCreateDiaryEntry,
  useDeleteDiaryEntry,
  useDiaryDay,
  uploadDiaryMedia,
  type AttachmentInput,
} from "@/hooks/useDiary";

function addDays(iso: string, n: number): string {
  const d = new Date(iso + "T00:00:00");
  d.setDate(d.getDate() + n);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
function humanDate(iso: string): string {
  return new Date(iso + "T00:00:00").toLocaleDateString("ru-RU", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
function hhmm(iso: string): string {
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

interface Pending {
  key: string;
  kind: string;
  contentType: string;
  preview: string;
}

export default function DiaryPage() {
  const [date, setDate] = useState(todayISO());
  const day = useDiaryDay(date);
  const create = useCreateDiaryEntry();
  const del = useDeleteDiaryEntry();

  const [body, setBody] = useState("");
  const [pending, setPending] = useState<Pending[]>([]);
  const [uploading, setUploading] = useState(false);

  const isToday = date === todayISO();
  const entries = day.data?.items ?? [];

  async function onFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? []);
    e.target.value = "";
    if (files.length === 0) return;
    setUploading(true);
    try {
      for (const f of files) {
        const r = await uploadDiaryMedia(f);
        setPending((p) => [...p, { ...r, preview: URL.createObjectURL(f) }]);
      }
    } catch {
      /* individual upload failures are ignored; user can retry */
    } finally {
      setUploading(false);
    }
  }

  function submit() {
    if (!body.trim() && pending.length === 0) return;
    const attachments: AttachmentInput[] = pending.map((p) => ({ key: p.key, contentType: p.contentType }));
    create.mutate(
      { date, body: body.trim(), attachments },
      {
        onSuccess: () => {
          setBody("");
          setPending([]);
        },
      },
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Дневник</h1>
          <p className="text-sm capitalize text-ink-500">{humanDate(date)}</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setDate(addDays(date, -1))} className="btn-ghost !px-2.5" aria-label="Предыдущий день">
            <ChevronLeft size={18} />
          </button>
          <input
            type="date"
            className="input !w-auto !py-1.5"
            value={date}
            max={todayISO()}
            onChange={(e) => e.target.value && setDate(e.target.value)}
          />
          <button
            onClick={() => setDate(addDays(date, 1))}
            disabled={isToday}
            className="btn-ghost !px-2.5"
            aria-label="Следующий день"
          >
            <ChevronRight size={18} />
          </button>
          {!isToday ? (
            <button onClick={() => setDate(todayISO())} className="btn-ghost">
              Сегодня
            </button>
          ) : null}
        </div>
      </div>

      {/* new entry */}
      <div className="card space-y-3">
        <textarea
          className="input min-h-24"
          placeholder="Что произошло в этот день?…"
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
        {pending.length > 0 ? (
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
            {pending.map((p) => (
              <div key={p.key} className="relative overflow-hidden rounded-lg border border-ink-800 bg-ink-800">
                {p.kind === "video" ? (
                  <video src={p.preview} className="h-24 w-full object-cover" />
                ) : (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={p.preview} className="h-24 w-full object-cover" alt="" />
                )}
                <button
                  onClick={() => setPending((prev) => prev.filter((x) => x.key !== p.key))}
                  className="absolute right-1 top-1 rounded-full bg-ink-950/80 px-1.5 text-xs text-ink-100"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        ) : null}
        <div className="flex items-center gap-2">
          <label className="btn-ghost cursor-pointer">
            <ImagePlus size={16} /> {uploading ? "Загрузка…" : "Фото / видео"}
            <input type="file" accept="image/*,video/*" multiple className="hidden" onChange={onFiles} disabled={uploading} />
          </label>
          <button onClick={submit} disabled={create.isPending || uploading} className="btn-primary ml-auto">
            Сохранить запись
          </button>
        </div>
      </div>

      {/* entries */}
      {day.isLoading ? (
        <div className="card h-32 animate-pulse bg-ink-800/40" />
      ) : entries.length > 0 ? (
        <div className="space-y-4">
          {entries.map((e) => (
            <div key={e.id} className="card">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-sm text-ink-500">{hhmm(e.createdAt)}</span>
                <button onClick={() => del.mutate(e.id)} className="text-ink-500 transition hover:text-bad" aria-label="Удалить">
                  <Trash2 size={16} />
                </button>
              </div>
              {e.body ? <p className="whitespace-pre-wrap text-ink-100">{e.body}</p> : null}
              {e.attachments.length > 0 ? (
                <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
                  {e.attachments.map((a, i) =>
                    !a.url ? null : a.kind === "video" ? (
                      <video key={i} src={a.url} controls className="w-full rounded-lg bg-ink-950" />
                    ) : (
                      <a key={i} href={a.url} target="_blank" rel="noreferrer" className="block">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img src={a.url} className="h-40 w-full rounded-lg object-cover" alt="" />
                      </a>
                    ),
                  )}
                </div>
              ) : null}
            </div>
          ))}
        </div>
      ) : (
        <div className="card text-center text-ink-500">На эту дату записей нет — добавьте первую выше.</div>
      )}
    </div>
  );
}
