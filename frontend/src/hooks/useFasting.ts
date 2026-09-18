"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { FastingSchedule, FastingSession, FastingState } from "@/lib/types";

export function useFasting() {
  return useQuery<FastingState>({
    queryKey: ["fasting"],
    queryFn: () => api("/fasting"),
    // resync occasionally so a phase flip (fast→eat) reaches the UI even if the
    // tab stays open; the ring itself ticks client-side every second.
    refetchInterval: 60_000,
  });
}

function useInvalidate() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ["fasting"] });
}

export function useStartFast() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input?: { startedAt?: string }) => api<FastingState>("/fasting/start", { method: "POST", body: JSON.stringify(input ?? {}) }),
    onSuccess: invalidate,
  });
}

export function useStopFast() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input?: { endedAt?: string }) => api<FastingState>("/fasting/stop", { method: "POST", body: JSON.stringify(input ?? {}) }),
    onSuccess: invalidate,
  });
}

export function useSetActiveStart() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (startedAt: string) => api<FastingState>("/fasting/active", { method: "PUT", body: JSON.stringify({ startedAt }) }),
    onSuccess: invalidate,
  });
}

export function useSetFastingSettings() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input: { fastingHours: number; eatingHours: number }) =>
      api<FastingState>("/fasting/settings", { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useSetFastingSchedule() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input: FastingSchedule) => api<FastingState>("/fasting/schedule", { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useFastingHistory(limit = 30) {
  return useQuery<{ items: FastingSession[] }>({
    queryKey: ["fasting", "history", limit],
    queryFn: () => api(`/fasting/history?limit=${limit}`),
  });
}

export function useDeleteFastSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/fasting/sessions/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["fasting"] }),
  });
}
