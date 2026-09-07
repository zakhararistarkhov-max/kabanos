"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, browserTZ } from "@/lib/api";
import type { Reminder } from "@/lib/types";

export interface ReminderInput {
  title: string;
  body: string;
  url: string;
  mode: "interval" | "times";
  intervalMinutes?: number | null;
  windowStart?: string;
  windowEnd?: string;
  times?: string[];
  days?: number[];
  condition?: "" | "water_below_goal" | "meds_due";
  timezone: string;
  enabled?: boolean;
}

export function useReminders() {
  return useQuery<{ items: Reminder[] }>({
    queryKey: ["reminders"],
    queryFn: () => api("/reminders"),
  });
}

export function useCreateReminder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ReminderInput) =>
      api<Reminder>("/reminders", { method: "POST", body: JSON.stringify({ ...input, timezone: input.timezone || browserTZ() }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["reminders"] }),
  });
}

export function useUpdateReminder(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ReminderInput) =>
      api<Reminder>(`/reminders/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["reminders"] }),
  });
}

export function useDeleteReminder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/reminders/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["reminders"] }),
  });
}

// reminderToInput converts an existing reminder back into an editable input,
// used to toggle `enabled` or prefill the edit form.
export function reminderToInput(r: Reminder): ReminderInput {
  return {
    title: r.title,
    body: r.body,
    url: r.url,
    mode: r.mode,
    intervalMinutes: r.intervalMinutes,
    windowStart: r.windowStart,
    windowEnd: r.windowEnd,
    times: r.times,
    days: r.days,
    condition: r.condition,
    timezone: r.timezone,
    enabled: r.enabled,
  };
}
