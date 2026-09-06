"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { WeightEntry, WeightSummary } from "@/lib/types";

export function useWeightSummary(limit = 60) {
  return useQuery<WeightSummary>({
    queryKey: ["weight", "summary", limit],
    queryFn: () => api<WeightSummary>(`/weight/summary?limit=${limit}`),
  });
}

export function useAddWeightEntry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { weightKg: number; note?: string; measuredOn?: string }) =>
      api<WeightEntry>("/weight/entries", {
        method: "POST",
        body: JSON.stringify({ ...input, tz: browserTZ() }),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["weight"] }),
  });
}

export function useDeleteWeightEntry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/weight/entries/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["weight"] }),
  });
}

export function useSetWeightGoal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (targetKg: number) => api("/weight/goal", { method: "PUT", body: JSON.stringify({ targetKg }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["weight"] }),
  });
}
