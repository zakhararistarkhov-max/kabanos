"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { Habit, HabitKind, HabitLog, HabitReminder, HabitStatus } from "@/lib/types";

export function useHabits() {
  return useQuery<{ items: Habit[] }>({
    queryKey: ["habits"],
    queryFn: () => api(`/habits?tz=${encodeURIComponent(browserTZ())}`),
    refetchInterval: 5 * 60_000,
  });
}

function useInvalidate() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ["habits"] });
}

export function useCreateHabit() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input: { name: string; kind: HabitKind; description?: string }) =>
      api<Habit>("/habits", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useUpdateHabit() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, ...input }: { id: string; name: string; description?: string; archived?: boolean }) =>
      api<Habit>(`/habits/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useDeleteHabit() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (id: string) => api(`/habits/${id}`, { method: "DELETE" }),
    onSuccess: invalidate,
  });
}

// --- daily check-ins (tracker) ---

export function useHabitCheckins(habitId: string, from: string, to: string) {
  return useQuery<{ days: Record<string, boolean> }>({
    queryKey: ["habits", "checkins", habitId, from, to],
    queryFn: () => api(`/habits/${habitId}/checkins?from=${from}&to=${to}`),
    enabled: Boolean(habitId && from && to),
  });
}

export function useSetCheckin(habitId: string) {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input: { day: string; success: boolean }) =>
      api(`/habits/${habitId}/checkins`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useClearCheckin(habitId: string) {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (day: string) => api(`/habits/${habitId}/checkins?day=${day}`, { method: "DELETE" }),
    onSuccess: invalidate,
  });
}

// --- mini-diary ---

export function useHabitLogs(habitId: string, enabled: boolean) {
  return useQuery<{ items: HabitLog[] }>({
    queryKey: ["habits", "logs", habitId],
    queryFn: () => api(`/habits/${habitId}/logs`),
    enabled,
  });
}

export function useAddHabitLog(habitId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { note: string; status: HabitStatus }) =>
      api<HabitLog>(`/habits/${habitId}/logs`, { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["habits", "logs", habitId] });
      qc.invalidateQueries({ queryKey: ["habits"] });
    },
  });
}

export function useDeleteHabitLog(habitId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (logId: string) => api(`/habits/${habitId}/logs/${logId}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["habits", "logs", habitId] });
      qc.invalidateQueries({ queryKey: ["habits"] });
    },
  });
}

// --- reminders ---

export function useHabitReminders(habitId: string, enabled: boolean) {
  return useQuery<{ items: HabitReminder[] }>({
    queryKey: ["habits", "reminders", habitId],
    queryFn: () => api(`/habits/${habitId}/reminders`),
    enabled,
  });
}

export function useAddHabitReminder(habitId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { text: string; times: string[]; days: number[]; timezone: string }) =>
      api<HabitReminder>(`/habits/${habitId}/reminders`, { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["habits", "reminders", habitId] });
      qc.invalidateQueries({ queryKey: ["habits"] });
    },
  });
}

export function useToggleHabitReminder(habitId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api(`/habits/${habitId}/reminders/${id}`, { method: "PUT", body: JSON.stringify({ enabled }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["habits", "reminders", habitId] }),
  });
}

export function useDeleteHabitReminder(habitId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/habits/${habitId}/reminders/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["habits", "reminders", habitId] });
      qc.invalidateQueries({ queryKey: ["habits"] });
    },
  });
}
