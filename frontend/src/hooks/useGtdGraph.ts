"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { GtdBoardInfo, GtdFeedTask, GtdGraphEdge, GtdGraphNode, GtdGraphSettings, GtdGraphSummary, GtdItem, GtdNodeKind } from "@/lib/types";

// board is the project id whose canvas we're viewing, or "" for the root board.
export function useGraph(board: string) {
  return useQuery<{ nodes: GtdGraphNode[]; edges: GtdGraphEdge[]; summary: GtdGraphSummary }>({
    queryKey: ["gtd", "graph", board || "root"],
    queryFn: () => api(`/gtd/graph?project=${encodeURIComponent(board)}`),
    refetchOnWindowFocus: false,
    staleTime: Infinity,
  });
}

export function useBoards() {
  return useQuery<{ items: GtdBoardInfo[] }>({
    queryKey: ["gtd", "graph", "boards"],
    queryFn: () => api("/gtd/graph/boards"),
  });
}

// useGtdFeed returns the ordered "to-do feed": open task nodes across all graphs,
// red first, then yellow/green, and by priority within a colour band.
export function useGtdFeed() {
  return useQuery<{ items: GtdFeedTask[] }>({
    queryKey: ["gtd", "graph", "feed"],
    queryFn: () => api("/gtd/graph/feed"),
  });
}

export function useSetProjectPriority() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, priority }: { id: string; priority: number }) =>
      api(`/gtd/projects/${id}/priority`, { method: "PUT", body: JSON.stringify({ priority }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gtd"] }),
  });
}

export function useSetItemPriority() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, priority }: { id: string; priority: number }) =>
      api<GtdItem>(`/gtd/items/${id}/priority`, { method: "PUT", body: JSON.stringify({ priority }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gtd"] }),
  });
}

export function useGraphSettings() {
  return useQuery<GtdGraphSettings>({
    queryKey: ["gtd", "graph", "settings"],
    queryFn: () => api("/gtd/graph/settings"),
  });
}

export function useSaveGraphSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (s: GtdGraphSettings) => api<GtdGraphSettings>("/gtd/graph/settings", { method: "PUT", body: JSON.stringify(s) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gtd", "graph"] }),
  });
}

export interface AddNodeInput {
  project: string;
  kind: GtdNodeKind;
  title: string;
  itemId?: string; // link an existing captured task instead of creating one
  deadline?: string;
  x: number;
  y: number;
}

export function useAddNode() {
  return useMutation({
    mutationFn: (input: AddNodeInput) => api<GtdGraphNode>("/gtd/graph/nodes", { method: "POST", body: JSON.stringify(input) }),
  });
}

export function useMoveNode() {
  return useMutation({
    mutationFn: ({ id, x, y }: { id: string; x: number; y: number }) =>
      api(`/gtd/graph/nodes/${id}`, { method: "PUT", body: JSON.stringify({ x, y }) }),
  });
}

// useUpdateNode changes a node's label, note and/or deadline ("" clears the deadline).
export function useUpdateNode() {
  return useMutation({
    mutationFn: ({ id, ...body }: { id: string; label?: string; note?: string; deadline?: string }) =>
      api(`/gtd/graph/nodes/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  });
}

export function useDeleteNode() {
  return useMutation({
    mutationFn: (id: string) => api(`/gtd/graph/nodes/${id}`, { method: "DELETE" }),
  });
}

export function useAddEdge() {
  return useMutation({
    mutationFn: ({ project, source, target }: { project: string; source: string; target: string }) =>
      api<GtdGraphEdge>("/gtd/graph/edges", { method: "POST", body: JSON.stringify({ project, source, target }) }),
  });
}

export function useDeleteEdge() {
  return useMutation({
    mutationFn: (id: string) => api(`/gtd/graph/edges/${id}`, { method: "DELETE" }),
  });
}

export function useAddNodeImage() {
  return useMutation({
    mutationFn: ({ id, key }: { id: string; key: string }) =>
      api(`/gtd/graph/nodes/${id}/images`, { method: "POST", body: JSON.stringify({ key }) }),
  });
}

export function useRemoveNodeImage() {
  return useMutation({
    mutationFn: ({ id, key }: { id: string; key: string }) =>
      api(`/gtd/graph/nodes/${id}/images`, { method: "DELETE", body: JSON.stringify({ key }) }),
  });
}

// uploadGraphImage uploads one image to object storage via a presigned URL and
// returns its key (to attach with useAddNodeImage).
export async function uploadGraphImage(file: File): Promise<string> {
  const { uploadUrl, key } = await api<{ uploadUrl: string; key: string }>("/gtd/graph/image-upload-url", {
    method: "POST",
    body: JSON.stringify({ contentType: file.type }),
  });
  const res = await fetch(uploadUrl, { method: "PUT", body: file, headers: { "content-type": file.type } });
  if (!res.ok) throw new Error("upload failed");
  return key;
}
