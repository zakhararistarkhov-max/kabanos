"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { DiaryDateCount, DiaryEntry } from "@/lib/types";

export function useDiaryDay(date: string) {
  return useQuery<{ date: string; items: DiaryEntry[] }>({
    queryKey: ["diary", "day", date],
    queryFn: () => api(`/diary/day?date=${date}&tz=${encodeURIComponent(browserTZ())}`),
    enabled: Boolean(date),
  });
}

export function useDiaryDates(from: string, to: string) {
  return useQuery<{ dates: DiaryDateCount[] }>({
    queryKey: ["diary", "dates", from, to],
    queryFn: () => api(`/diary/dates?from=${from}&to=${to}`),
  });
}

export interface AttachmentInput {
  key: string;
  contentType: string;
}

export function useCreateDiaryEntry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { date: string; body: string; attachments: AttachmentInput[] }) =>
      api<DiaryEntry>("/diary/entries", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["diary"] }),
  });
}

export function useDeleteDiaryEntry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/diary/entries/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["diary"] }),
  });
}

// Uploads one media file (image or video) to object storage via a presigned URL.
export async function uploadDiaryMedia(file: File): Promise<{ key: string; kind: string; contentType: string }> {
  const { uploadUrl, key, kind } = await api<{ uploadUrl: string; key: string; kind: string }>(
    "/diary/media-upload-url",
    { method: "POST", body: JSON.stringify({ contentType: file.type }) },
  );
  const res = await fetch(uploadUrl, { method: "PUT", body: file, headers: { "content-type": file.type } });
  if (!res.ok) throw new Error("upload failed");
  return { key, kind, contentType: file.type };
}
