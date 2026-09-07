"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type {
  CatalogComment,
  CatalogScope,
  CatalogSort,
  Exercise,
  TrainingMeta,
  Workout,
} from "@/lib/types";

// ---------------- meta ----------------

export function useTrainingMeta() {
  return useQuery<TrainingMeta>({
    queryKey: ["training", "meta"],
    queryFn: () => api<TrainingMeta>("/training/meta"),
    staleTime: 1000 * 60 * 60, // rarely changes
  });
}

interface ListResult<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

// ---------------- exercises ----------------

export interface ExerciseListParams {
  scope: CatalogScope;
  q: string;
  sort: CatalogSort;
  category?: string;
  difficulty?: string;
  equipment?: string;
  page: number;
}

export function useExercises(p: ExerciseListParams) {
  return useQuery<ListResult<Exercise>>({
    queryKey: ["exercises", p],
    queryFn: () => {
      const qs = new URLSearchParams({ scope: p.scope, sort: p.sort, page: String(p.page) });
      if (p.q) qs.set("q", p.q);
      if (p.category) qs.set("category", p.category);
      if (p.difficulty) qs.set("difficulty", p.difficulty);
      if (p.equipment) qs.set("equipment", p.equipment);
      return api<ListResult<Exercise>>(`/training/exercises?${qs.toString()}`);
    },
  });
}

export function useExercise(id: string) {
  return useQuery<Exercise>({
    queryKey: ["exercise", id],
    queryFn: () => api<Exercise>(`/training/exercises/${id}`),
    enabled: Boolean(id),
  });
}

export interface ExerciseInput {
  name: string;
  description: string;
  category: string;
  difficulty: string;
  jointImpact: string;
  equipment: string[];
  muscles: string[];
  imageKey?: string | null;
  videoUrl: string;
}

export function useCreateExercise() {
  return useMutation({
    mutationFn: (input: ExerciseInput) =>
      api<Exercise>("/training/exercises", { method: "POST", body: JSON.stringify(input) }),
  });
}

export function useUpdateExercise(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ExerciseInput) =>
      api<Exercise>(`/training/exercises/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["exercise", id] }),
  });
}

export function useDeleteExercise() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/training/exercises/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["exercises"] }),
  });
}

export function useRateExercise(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rating: number) =>
      api(`/training/exercises/${id}/rating`, { method: "PUT", body: JSON.stringify({ rating }) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["exercise", id] });
      qc.invalidateQueries({ queryKey: ["exercises"] });
    },
  });
}

export function useToggleExerciseFavorite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, favorite }: { id: string; favorite: boolean }) =>
      api(`/training/exercises/${id}/favorite`, { method: favorite ? "PUT" : "DELETE" }),
    onSuccess: (_d, v) => {
      qc.invalidateQueries({ queryKey: ["exercise", v.id] });
      qc.invalidateQueries({ queryKey: ["exercises"] });
    },
  });
}

export function usePublishExercise(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (publish: boolean) =>
      api(`/training/exercises/${id}/publish`, { method: publish ? "PUT" : "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["exercise", id] });
      qc.invalidateQueries({ queryKey: ["exercises"] });
    },
  });
}

export function useExerciseComments(id: string) {
  return useQuery<{ items: CatalogComment[] }>({
    queryKey: ["exercise", id, "comments"],
    queryFn: () => api(`/training/exercises/${id}/comments`),
    enabled: Boolean(id),
  });
}

export function useAddExerciseComment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) =>
      api(`/training/exercises/${id}/comments`, { method: "POST", body: JSON.stringify({ body }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["exercise", id, "comments"] }),
  });
}

export function useDeleteExerciseComment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) =>
      api(`/training/exercises/${id}/comments/${commentId}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["exercise", id, "comments"] }),
  });
}

export async function uploadExerciseImage(file: File): Promise<string> {
  return uploadMedia("/training/exercises/image-upload-url", file);
}

// ---------------- workouts ----------------

export interface WorkoutListParams {
  scope: CatalogScope;
  q: string;
  sort: CatalogSort;
  difficulty?: string;
  page: number;
}

export function useWorkouts(p: WorkoutListParams) {
  return useQuery<ListResult<Workout>>({
    queryKey: ["workouts", p],
    queryFn: () => {
      const qs = new URLSearchParams({ scope: p.scope, sort: p.sort, page: String(p.page) });
      if (p.q) qs.set("q", p.q);
      if (p.difficulty) qs.set("difficulty", p.difficulty);
      return api<ListResult<Workout>>(`/training/workouts?${qs.toString()}`);
    },
  });
}

export function useWorkout(id: string) {
  return useQuery<Workout>({
    queryKey: ["workout", id],
    queryFn: () => api<Workout>(`/training/workouts/${id}`),
    enabled: Boolean(id),
  });
}

export interface WorkoutItemInput {
  exerciseId: string;
  sets?: number | null;
  reps?: number | null;
  durationSec?: number | null;
  restSec?: number | null;
  weightKg?: number | null;
  note?: string;
}

export interface WorkoutInput {
  name: string;
  description: string;
  difficulty: string;
  imageKey?: string | null;
  items: WorkoutItemInput[];
}

export function useCreateWorkout() {
  return useMutation({
    mutationFn: (input: WorkoutInput) =>
      api<Workout>("/training/workouts", { method: "POST", body: JSON.stringify(input) }),
  });
}

export function useUpdateWorkout(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: WorkoutInput) =>
      api<Workout>(`/training/workouts/${id}`, { method: "PUT", body: JSON.stringify(input) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workout", id] }),
  });
}

export function useDeleteWorkout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api(`/training/workouts/${id}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workouts"] }),
  });
}

export function useRateWorkout(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rating: number) =>
      api(`/training/workouts/${id}/rating`, { method: "PUT", body: JSON.stringify({ rating }) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workout", id] });
      qc.invalidateQueries({ queryKey: ["workouts"] });
    },
  });
}

export function useToggleWorkoutFavorite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, favorite }: { id: string; favorite: boolean }) =>
      api(`/training/workouts/${id}/favorite`, { method: favorite ? "PUT" : "DELETE" }),
    onSuccess: (_d, v) => {
      qc.invalidateQueries({ queryKey: ["workout", v.id] });
      qc.invalidateQueries({ queryKey: ["workouts"] });
    },
  });
}

export function usePublishWorkout(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (publish: boolean) =>
      api(`/training/workouts/${id}/publish`, { method: publish ? "PUT" : "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workout", id] });
      qc.invalidateQueries({ queryKey: ["workouts"] });
    },
  });
}

export function useWorkoutComments(id: string) {
  return useQuery<{ items: CatalogComment[] }>({
    queryKey: ["workout", id, "comments"],
    queryFn: () => api(`/training/workouts/${id}/comments`),
    enabled: Boolean(id),
  });
}

export function useAddWorkoutComment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) =>
      api(`/training/workouts/${id}/comments`, { method: "POST", body: JSON.stringify({ body }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workout", id, "comments"] }),
  });
}

export function useDeleteWorkoutComment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) =>
      api(`/training/workouts/${id}/comments/${commentId}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workout", id, "comments"] }),
  });
}

export async function uploadWorkoutImage(file: File): Promise<string> {
  return uploadMedia("/training/workouts/image-upload-url", file);
}

// ---------------- shared media upload ----------------

async function uploadMedia(presignPath: string, file: File): Promise<string> {
  const { uploadUrl, key } = await api<{ uploadUrl: string; key: string }>(presignPath, {
    method: "POST",
    body: JSON.stringify({ contentType: file.type }),
  });
  const res = await fetch(uploadUrl, { method: "PUT", body: file, headers: { "content-type": file.type } });
  if (!res.ok) throw new Error("upload failed");
  return key;
}
