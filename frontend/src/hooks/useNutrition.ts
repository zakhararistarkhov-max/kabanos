"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { ActivityType, DaySummary, DayTotals, Macros, Meal, NutritionGoal } from "@/lib/types";

export function useNutritionDay(date?: string) {
  return useQuery<DaySummary>({
    queryKey: ["nutrition", "day", date ?? "today"],
    queryFn: () => {
      const qs = new URLSearchParams({ tz: browserTZ() });
      if (date) qs.set("date", date);
      return api<DaySummary>(`/nutrition/day?${qs.toString()}`);
    },
  });
}

export function useNutritionHistory(from: string, to: string) {
  return useQuery<{ goal: NutritionGoal; series: DayTotals[] }>({
    queryKey: ["nutrition", "history", from, to],
    queryFn: () => {
      const qs = new URLSearchParams({ from, to, tz: browserTZ() });
      return api(`/nutrition/history?${qs.toString()}`);
    },
  });
}

export function useActivityTypes() {
  return useQuery<{ types: ActivityType[] }>({
    queryKey: ["nutrition", "activity-types"],
    queryFn: () => api("/nutrition/activity-types"),
    staleTime: Infinity,
  });
}

function invalidateNutrition(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ["nutrition"] });
}

export function useSetNutritionGoal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (g: NutritionGoal) => api("/nutrition/goal", { method: "PUT", body: JSON.stringify(g) }),
    onSuccess: () => invalidateNutrition(qc),
  });
}

export function useAddManualEntry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: Macros & { name: string; meal?: Meal | null }) =>
      api("/nutrition/diet", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => invalidateNutrition(qc),
  });
}

export function useDeleteDietEntry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/nutrition/diet/${id}`, { method: "DELETE" }),
    onSuccess: () => invalidateNutrition(qc),
  });
}

export function useAddActivity() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { type?: string; met?: number; durationMin?: number; kcal?: number }) =>
      api("/nutrition/activities", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => invalidateNutrition(qc),
  });
}

export function useDeleteActivity() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/nutrition/activities/${id}`, { method: "DELETE" }),
    onSuccess: () => invalidateNutrition(qc),
  });
}
