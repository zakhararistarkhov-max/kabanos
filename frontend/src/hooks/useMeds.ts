"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { MedHistory, Medication } from "@/lib/types";

export function useMeds() {
  return useQuery<{ items: Medication[]; date: string }>({
    queryKey: ["meds"],
    queryFn: () => api(`/meds?tz=${encodeURIComponent(browserTZ())}`),
  });
}

// useMedHistory fetches a course's full intake log. Pass enabled=false to defer
// the request until the user opens the history panel.
export function useMedHistory(id: string, enabled: boolean) {
  return useQuery<MedHistory>({
    queryKey: ["meds", "history", id],
    queryFn: () => api<MedHistory>(`/meds/${id}/history`),
    enabled,
  });
}

export interface MedInput {
  name: string;
  unit: string;
  dose: number;
  timesPerDay: number;
  startDate: string;
  durationDays: number | null;
  notes: string;
  active?: boolean;
}

function invalidate(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ["meds"] });
}

export function useCreateMed() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: MedInput) => api<Medication>("/meds", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => invalidate(qc),
  });
}

export function useUpdateMed(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: MedInput) => api<Medication>(`/meds/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: () => invalidate(qc),
  });
}

export function useDeleteMed() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/meds/${id}`, { method: "DELETE" }),
    onSuccess: () => invalidate(qc),
  });
}

export function useTakeIntake() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/meds/${id}/intake?tz=${encodeURIComponent(browserTZ())}`, { method: "POST" }),
    onSuccess: () => invalidate(qc),
  });
}

export function useUndoIntake() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/meds/${id}/intake?tz=${encodeURIComponent(browserTZ())}`, { method: "DELETE" }),
    onSuccess: () => invalidate(qc),
  });
}
