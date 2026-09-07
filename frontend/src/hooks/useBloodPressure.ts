"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { PressureEntry, PressureSummary } from "@/lib/types";

export function usePressureSummary(limit = 60) {
  return useQuery<PressureSummary>({
    queryKey: ["pressure", "summary", limit],
    queryFn: () => api<PressureSummary>(`/pressure/summary?limit=${limit}`),
  });
}

export interface PressureInput {
  systolic: number;
  diastolic: number;
  pulse?: number | null;
  note?: string;
}

export function useAddPressure() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: PressureInput) =>
      api<PressureEntry>("/pressure/entries", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pressure"] }),
  });
}

export function useDeletePressure() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/pressure/entries/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pressure"] }),
  });
}
