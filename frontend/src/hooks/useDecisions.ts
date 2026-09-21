"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Decision, DecisionInput, DecisionReview } from "@/lib/types";

export function useDecisions() {
  return useQuery<{ items: Decision[] }>({
    queryKey: ["decisions"],
    queryFn: () => api("/decisions"),
  });
}

function useInvalidate() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ["decisions"] });
}

export function useCreateDecision() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (input: DecisionInput) => api<Decision>("/decisions", { method: "POST", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useUpdateDecision() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, ...input }: DecisionInput & { id: string }) =>
      api<Decision>(`/decisions/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: invalidate,
  });
}

export function useReviewDecision() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, ...review }: DecisionReview & { id: string }) =>
      api<Decision>(`/decisions/${id}/review`, { method: "POST", body: JSON.stringify(review) }),
    onSuccess: invalidate,
  });
}

export function useDeleteDecision() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: (id: string) => api(`/decisions/${id}`, { method: "DELETE" }),
    onSuccess: invalidate,
  });
}
