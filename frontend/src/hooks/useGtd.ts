"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { GtdBucket, GtdEnergy, GtdItem, GtdProject, GtdProjectStatus, GtdReview } from "@/lib/types";

export interface GtdItemInput {
  projectId?: string | null;
  title: string;
  notes?: string;
  bucket?: GtdBucket;
  context?: string;
  waitingFor?: string;
  scheduledAt?: string | null;
  endAt?: string | null;
  allDay?: boolean;
  dueOn?: string | null;
  energy?: GtdEnergy;
  timeMinutes?: number | null;
  priority?: number;
}

export interface GtdProjectInput {
  title: string;
  outcome?: string;
  notes?: string;
  status?: GtdProjectStatus;
}

export interface ItemsFilter {
  bucket?: GtdBucket;
  context?: string;
  projectId?: string;
  done?: boolean;
  maxTime?: number;
  energy?: GtdEnergy;
}

function qs(f: ItemsFilter): string {
  const p = new URLSearchParams();
  if (f.bucket) p.set("bucket", f.bucket);
  if (f.context) p.set("context", f.context);
  if (f.projectId) p.set("projectId", f.projectId);
  if (f.done !== undefined) p.set("done", String(f.done));
  if (f.maxTime) p.set("maxTime", String(f.maxTime));
  if (f.energy) p.set("energy", f.energy);
  const s = p.toString();
  return s ? `?${s}` : "";
}

export function useGtdItems(filter: ItemsFilter = {}) {
  return useQuery<{ items: GtdItem[] }>({
    queryKey: ["gtd", "items", filter],
    queryFn: () => api(`/gtd/items${qs(filter)}`),
  });
}

export function useGtdProjects() {
  return useQuery<{ items: GtdProject[] }>({
    queryKey: ["gtd", "projects"],
    queryFn: () => api("/gtd/projects"),
  });
}

export function useGtdReview() {
  return useQuery<GtdReview>({
    queryKey: ["gtd", "review"],
    queryFn: () => api("/gtd/review"),
  });
}

export function useGtdContexts() {
  return useQuery<{ items: string[] }>({
    queryKey: ["gtd", "contexts"],
    queryFn: () => api("/gtd/contexts"),
  });
}

// Any GTD mutation can shift counts across every view, so we invalidate the
// whole "gtd" tree rather than tracking fine-grained keys.
function useGtdInvalidate() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ["gtd"] });
}

export function useCaptureItem() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: (input: GtdItemInput) => api<GtdItem>("/gtd/items", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useUpdateItem() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: GtdItemInput }) =>
      api<GtdItem>(`/gtd/items/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useDeleteItem() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: (id: string) => api(`/gtd/items/${id}`, { method: "DELETE" }),
    onSuccess: invalidate,
  });
}

export function useToggleDone() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: ({ id, done }: { id: string; done: boolean }) =>
      api<GtdItem>(`/gtd/items/${id}/done`, { method: done ? "POST" : "DELETE" }),
    onSuccess: invalidate,
  });
}

export function useCreateProject() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: (input: GtdProjectInput) => api<GtdProject>("/gtd/projects", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useUpdateProject() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: GtdProjectInput }) =>
      api<GtdProject>(`/gtd/projects/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useDeleteProject() {
  const invalidate = useGtdInvalidate();
  return useMutation({
    mutationFn: (id: string) => api(`/gtd/projects/${id}`, { method: "DELETE" }),
    onSuccess: invalidate,
  });
}

// itemToInput turns an existing item into an editable input (for edit forms).
export function itemToInput(it: GtdItem): GtdItemInput {
  return {
    projectId: it.projectId,
    title: it.title,
    notes: it.notes,
    bucket: it.bucket,
    context: it.context,
    waitingFor: it.waitingFor,
    scheduledAt: it.scheduledAt,
    endAt: it.endAt,
    allDay: it.allDay,
    dueOn: it.dueOn,
    energy: it.energy,
    timeMinutes: it.timeMinutes,
    priority: it.priority,
  };
}
