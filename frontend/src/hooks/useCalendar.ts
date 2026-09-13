"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { CalendarStatus, CalendarSyncResult, RemoteCalendar } from "@/lib/types";

export function useCalendarStatus() {
  return useQuery<CalendarStatus>({
    queryKey: ["calendar", "status"],
    queryFn: () => api("/calendar/status"),
  });
}

function useCalendarInvalidate() {
  const qc = useQueryClient();
  // A calendar change can add/remove GTD calendar items, so refresh both trees.
  return () => {
    qc.invalidateQueries({ queryKey: ["calendar"] });
    qc.invalidateQueries({ queryKey: ["gtd"] });
  };
}

export function useConnectCalendar() {
  const invalidate = useCalendarInvalidate();
  return useMutation({
    mutationFn: (input: { login: string; password: string }) =>
      api<CalendarStatus>("/calendar/connect", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useDisconnectCalendar() {
  const invalidate = useCalendarInvalidate();
  return useMutation({
    mutationFn: () => api("/calendar/disconnect", { method: "POST" }),
    onSuccess: invalidate,
  });
}

export function useCalendars(enabled: boolean) {
  return useQuery<{ items: RemoteCalendar[] }>({
    queryKey: ["calendar", "calendars"],
    queryFn: () => api("/calendar/calendars"),
    enabled,
  });
}

export function useSelectCalendar() {
  const invalidate = useCalendarInvalidate();
  return useMutation({
    mutationFn: (input: { url: string; name: string }) =>
      api<CalendarStatus>("/calendar/select", { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useSyncCalendar() {
  const invalidate = useCalendarInvalidate();
  return useMutation({
    mutationFn: () => api<CalendarSyncResult>("/calendar/sync", { method: "POST" }),
    onSuccess: invalidate,
  });
}
