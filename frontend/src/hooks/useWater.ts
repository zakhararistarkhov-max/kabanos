"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { WaterDay, WaterHistory, WaterIntake, WaterSource } from "@/lib/types";

export function useWaterDay(date?: string) {
  return useQuery<WaterDay>({
    queryKey: ["water", "day", date ?? "today"],
    queryFn: () => {
      const qs = new URLSearchParams({ tz: browserTZ() });
      if (date) qs.set("date", date);
      return api<WaterDay>(`/water/day?${qs.toString()}`);
    },
  });
}

export function useWaterHistory(from: string, to: string) {
  return useQuery<WaterHistory>({
    queryKey: ["water", "history", from, to],
    queryFn: () => {
      const qs = new URLSearchParams({ from, to, tz: browserTZ() });
      return api<WaterHistory>(`/water/history?${qs.toString()}`);
    },
  });
}

export function useAddIntake() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { amountMl: number; source: WaterSource }) =>
      api<WaterIntake>("/water/intake", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["water"] }),
  });
}

export function useDeleteIntake() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/water/intake/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["water"] }),
  });
}

export function useSetWaterGoal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (dailyMl: number) => api("/water/goal", { method: "PUT", body: JSON.stringify({ dailyMl }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["water"] }),
  });
}
