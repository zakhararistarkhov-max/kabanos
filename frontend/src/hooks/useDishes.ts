"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Dish, DishComment, DishScope, DishSort, Meal } from "@/lib/types";

export interface DishListResult {
  items: Dish[];
  total: number;
  page: number;
  limit: number;
}

export function useDishes(params: { scope: DishScope; q: string; sort: DishSort; page: number }) {
  const { scope, q, sort, page } = params;
  return useQuery<DishListResult>({
    queryKey: ["dishes", scope, q, sort, page],
    queryFn: () => {
      const qs = new URLSearchParams({ scope, sort, page: String(page) });
      if (q) qs.set("q", q);
      return api<DishListResult>(`/nutrition/dishes?${qs.toString()}`);
    },
  });
}

export function useDish(id: string) {
  return useQuery<Dish>({
    queryKey: ["dish", id],
    queryFn: () => api<Dish>(`/nutrition/dishes/${id}`),
    enabled: Boolean(id),
  });
}

export interface DishInput {
  name: string;
  description: string;
  recipe: string;
  imageKey?: string | null;
  kcalPer100: number;
  proteinPer100: number;
  fatPer100: number;
  carbsPer100: number;
  servingGrams?: number | null;
  // When present, the dish is composed of other dishes and its macros are
  // computed server-side from these.
  ingredients?: { dishId: string; grams: number }[];
}

export function useCreateDish() {
  return useMutation({
    mutationFn: (input: DishInput) => api<Dish>("/nutrition/dishes", { method: "POST", body: JSON.stringify(input) }),
  });
}

export function useUpdateDish(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: DishInput) => api<Dish>(`/nutrition/dishes/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dish", id] });
      qc.invalidateQueries({ queryKey: ["dishes"] });
    },
  });
}

export function useDeleteDish() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/nutrition/dishes/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["dishes"] }),
  });
}

export function usePublishDish(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (publish: boolean) =>
      api(`/nutrition/dishes/${id}/publish`, { method: publish ? "PUT" : "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dish", id] });
      qc.invalidateQueries({ queryKey: ["dishes"] });
    },
  });
}

export function useRateDish(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rating: number) => api(`/nutrition/dishes/${id}/rating`, { method: "PUT", body: JSON.stringify({ rating }) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dish", id] });
      qc.invalidateQueries({ queryKey: ["dishes"] });
    },
  });
}

export function useToggleFavorite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, favorite }: { id: string; favorite: boolean }) =>
      api(`/nutrition/dishes/${id}/favorite`, { method: favorite ? "PUT" : "DELETE" }),
    onSuccess: (_d, v) => {
      qc.invalidateQueries({ queryKey: ["dish", v.id] });
      qc.invalidateQueries({ queryKey: ["dishes"] });
    },
  });
}

export function useDishComments(id: string) {
  return useQuery<{ items: DishComment[] }>({
    queryKey: ["dish", id, "comments"],
    queryFn: () => api(`/nutrition/dishes/${id}/comments`),
    enabled: Boolean(id),
  });
}

export function useAddComment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => api(`/nutrition/dishes/${id}/comments`, { method: "POST", body: JSON.stringify({ body }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["dish", id, "comments"] }),
  });
}

export function useDeleteComment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) => api(`/nutrition/dishes/${id}/comments/${commentId}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["dish", id, "comments"] }),
  });
}

export function useAddDishToDiet(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { grams?: number; servings?: number; meal?: Meal | null }) =>
      api(`/nutrition/dishes/${id}/diet`, { method: "POST", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["nutrition"] }),
  });
}

// Uploads an image to object storage via a presigned URL and returns its key.
export async function uploadDishImage(file: File): Promise<string> {
  const { uploadUrl, key } = await api<{ uploadUrl: string; key: string }>("/nutrition/dishes/image-upload-url", {
    method: "POST",
    body: JSON.stringify({ contentType: file.type }),
  });
  const res = await fetch(uploadUrl, { method: "PUT", body: file, headers: { "content-type": file.type } });
  if (!res.ok) throw new Error("upload failed");
  return key;
}
