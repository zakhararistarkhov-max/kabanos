"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  ReactFlowProvider,
  useEdgesState,
  useNodesState,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeProps,
  type NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { CalendarClock, ListChecks, Plus, StickyNote } from "lucide-react";
import {
  uploadGraphImage,
  useAddEdge,
  useAddNode,
  useAddNodeImage,
  useDeleteEdge,
  useDeleteNode,
  useGraph,
  useMoveNode,
  useRemoveNodeImage,
  useUpdateNode,
} from "@/hooks/useGtdGraph";
import type { GtdColor, GtdGraphNode, GtdGraphSummary, GtdNodeKind } from "@/lib/types";

interface NodeData extends Record<string, unknown> {
  label: string;
  kind: GtdNodeKind;
  color: GtdColor;
  done: boolean;
  deadline: string | null;
  note: string;
  images: number;
  refProjectId: string | null;
  onOpen: (projectId: string) => void;
  onSelect: (id: string) => void;
}
type AppNode = Node<NodeData>;

const COLOR_RING: Record<GtdColor, string> = {
  green: "border-good/70 bg-good/10",
  yellow: "border-warn/70 bg-warn/10",
  red: "border-bad/70 bg-bad/10",
};
const COLOR_DOT: Record<GtdColor, string> = { green: "bg-good", yellow: "bg-warn", red: "bg-bad" };

function fmtDate(iso: string | null): string {
  if (!iso) return "";
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}`;
}

function GtdNode({ id, data, selected }: NodeProps<AppNode>) {
  const Icon = data.kind === "project" ? ListChecks : data.kind === "note" ? StickyNote : CalendarClock;
  return (
    <div
      className={`min-w-[9rem] max-w-[15rem] cursor-pointer rounded-xl border-2 px-3 py-2 text-sm shadow-sm ${COLOR_RING[data.color]} ${
        selected ? "ring-2 ring-brand" : ""
      }`}
      onClick={() => data.onSelect(id)}
    >
      <Handle type="target" position={Position.Left} className="!h-2 !w-2 !border-0 !bg-brand" />
      <div className="flex items-start gap-1.5">
        <Icon size={14} className="mt-0.5 shrink-0 opacity-70" />
        <span className={`flex-1 break-words ${data.done ? "text-ink-500 line-through" : ""}`}>{data.label || "…"}</span>
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-ink-500">
        {data.deadline ? (
          <span className="inline-flex items-center gap-1">
            <span className={`h-1.5 w-1.5 rounded-full ${COLOR_DOT[data.color]}`} /> {fmtDate(data.deadline)}
          </span>
        ) : null}
        {data.note ? <span title="есть заметка">📝</span> : null}
        {data.images > 0 ? <span title="фото">🖼 {data.images}</span> : null}
        {data.kind === "project" && data.refProjectId ? (
          <button
            className="nodrag ml-auto text-good hover:underline"
            onClick={(e) => {
              e.stopPropagation();
              data.onOpen(data.refProjectId!);
            }}
          >
            открыть →
          </button>
        ) : null}
      </div>
      <Handle type="source" position={Position.Right} className="!h-2 !w-2 !border-0 !bg-brand" />
    </div>
  );
}

function toAppNode(n: GtdGraphNode, onOpen: (p: string) => void, onSelect: (id: string) => void): AppNode {
  return {
    id: n.id,
    type: "gtd",
    position: { x: n.x, y: n.y },
    data: {
      label: n.label,
      kind: n.kind,
      color: n.color,
      done: n.done,
      deadline: n.deadline,
      note: n.note,
      images: n.imageUrls.length,
      refProjectId: n.refProjectId,
      onOpen,
      onSelect,
    },
  };
}

function Inner({ board, title }: { board: string; title: string }) {
  const router = useRouter();
  const qc = useQueryClient();
  const graph = useGraph(board);
  const addNode = useAddNode();
  const moveNode = useMoveNode();
  const deleteNode = useDeleteNode();
  const addEdgeMut = useAddEdge();
  const deleteEdgeMut = useDeleteEdge();
  const rf = useReactFlow();

  const [nodes, setNodes, onNodesChange] = useNodesState<AppNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [summary, setSummary] = useState<GtdGraphSummary | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const refresh = useCallback(() => {
    qc.invalidateQueries({ queryKey: ["gtd", "graph", board || "root"] });
    qc.invalidateQueries({ queryKey: ["gtd", "graph", "boards"] });
  }, [qc, board]);

  const openProject = useCallback((projectId: string) => router.push(`/gtd/graph/${projectId}`), [router]);
  const selectNode = useCallback((id: string) => setSelectedId(id), []);

  // Rebuild from server whenever the graph query updates (mount + invalidations).
  useEffect(() => {
    if (!graph.data) return;
    setNodes(graph.data.nodes.map((n) => toAppNode(n, openProject, selectNode)));
    setEdges(graph.data.edges.map((e) => ({ id: e.id, source: e.source, target: e.target })));
    setSummary(graph.data.summary);
  }, [graph.data, openProject, selectNode, setNodes, setEdges]);

  const onConnect = useCallback(
    async (c: Connection) => {
      if (!c.source || !c.target || c.source === c.target) return;
      try {
        await addEdgeMut.mutateAsync({ project: board, source: c.source, target: c.target });
        refresh();
      } catch {
        /* ignore */
      }
    },
    [addEdgeMut, board, refresh],
  );

  async function spawn(kind: GtdNodeKind) {
    const label = window.prompt(kind === "note" ? "Текст заметки" : kind === "project" ? "Название подпроекта" : "Название задачи");
    if (!label || !label.trim()) return;
    const pos = rf.screenToFlowPosition({ x: window.innerWidth / 2, y: 280 });
    try {
      const n = await addNode.mutateAsync({ project: board, kind, title: label.trim(), x: pos.x, y: pos.y });
      refresh();
      setSelectedId(n.id);
    } catch {
      /* ignore */
    }
  }

  const nodeTypes = useMemo<NodeTypes>(() => ({ gtd: GtdNode }), []);
  const selected = graph.data?.nodes.find((n) => n.id === selectedId) ?? null;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-2xl font-bold">{title}</h1>
          {summary ? <SummaryBar s={summary} /> : null}
        </div>
        <div className="flex flex-wrap gap-2">
          <button onClick={() => spawn("task")} className="btn-ghost">
            <Plus size={15} /> Задача
          </button>
          <button onClick={() => spawn("project")} className="btn-ghost">
            <Plus size={15} /> Подпроект
          </button>
          <button onClick={() => spawn("note")} className="btn-ghost">
            <Plus size={15} /> Заметка
          </button>
        </div>
      </div>

      <div className="relative flex gap-3">
        <div className="h-[calc(100vh-18rem)] min-h-[26rem] flex-1 overflow-hidden rounded-2xl border border-ink-800 bg-ink-950/40">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeDragStop={(_, node) => moveNode.mutate({ id: node.id, x: node.position.x, y: node.position.y })}
            onNodesDelete={(deleted) => {
              deleted.forEach((n) => deleteNode.mutate(n.id));
              if (deleted.some((n) => n.id === selectedId)) setSelectedId(null);
              refresh();
            }}
            onEdgesDelete={(deleted) => {
              deleted.forEach((e) => deleteEdgeMut.mutate(e.id));
              refresh();
            }}
            onNodeDoubleClick={(_, node) => {
              const d = node.data as NodeData;
              if (d.kind === "project" && d.refProjectId) openProject(d.refProjectId);
            }}
            fitView
            proOptions={{ hideAttribution: true }}
          >
            <Background color="#334155" gap={18} />
            <MiniMap pannable zoomable className="!bg-ink-900" />
            <Controls className="!shadow-none" />
          </ReactFlow>
        </div>

        {selected ? (
          <NodeInspector
            key={selected.id}
            node={selected}
            onClose={() => setSelectedId(null)}
            onChanged={refresh}
            onDelete={() => {
              rf.deleteElements({ nodes: [{ id: selected.id }] });
            }}
          />
        ) : null}
      </div>

      {graph.isLoading ? <p className="text-sm text-ink-500">Загрузка…</p> : null}
      {nodes.length === 0 && !graph.isLoading ? (
        <p className="text-sm text-ink-500">Пусто. Добавьте задачу, подпроект или заметку кнопками выше, кликните по ноде — откроется панель с дедлайном, заметкой и фото.</p>
      ) : null}
    </div>
  );
}

function SummaryBar({ s }: { s: GtdGraphSummary }) {
  return (
    <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-ink-500">
      <span className="inline-flex items-center gap-1">
        <span className="h-2 w-2 rounded-full bg-bad" /> просрочено: {s.red}
      </span>
      <span className="inline-flex items-center gap-1">
        <span className="h-2 w-2 rounded-full bg-warn" /> скоро: {s.yellow}
      </span>
      <span className="inline-flex items-center gap-1">
        <span className="h-2 w-2 rounded-full bg-good" /> в запасе: {s.green}
      </span>
      {s.projectedCompletion ? <span>· план. окончание: {s.projectedCompletion}</span> : null}
    </div>
  );
}

function NodeInspector({
  node,
  onClose,
  onChanged,
  onDelete,
}: {
  node: GtdGraphNode;
  onClose: () => void;
  onChanged: () => void;
  onDelete: () => void;
}) {
  const update = useUpdateNode();
  const addImg = useAddNodeImage();
  const removeImg = useRemoveNodeImage();
  const [note, setNote] = useState(node.note);
  const [deadline, setDeadline] = useState(node.deadline ?? "");
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    setNote(node.note);
    setDeadline(node.deadline ?? "");
  }, [node.id, node.note, node.deadline]);

  async function saveDeadline(v: string) {
    setDeadline(v);
    await update.mutateAsync({ id: node.id, deadline: v }).catch(() => undefined);
    onChanged();
  }
  async function saveNote() {
    if (note === node.note) return;
    await update.mutateAsync({ id: node.id, note }).catch(() => undefined);
    onChanged();
  }
  async function onFiles(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? []);
    if (files.length === 0) return;
    setUploading(true);
    try {
      for (const f of files) {
        const key = await uploadGraphImage(f);
        await addImg.mutateAsync({ id: node.id, key });
      }
      onChanged();
    } catch {
      /* ignore */
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  }
  async function removeImage(key: string) {
    await removeImg.mutateAsync({ id: node.id, key }).catch(() => undefined);
    onChanged();
  }

  return (
    <div className="w-72 shrink-0 space-y-3 rounded-2xl border border-ink-800 bg-ink-950/60 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="text-sm font-semibold">{node.label || "Нода"}</div>
        <button onClick={onClose} className="text-xs text-ink-500 hover:text-ink-100">
          закрыть
        </button>
      </div>

      <div>
        <label className="label">Дедлайн</label>
        <input type="date" className="input" value={deadline} onChange={(e) => saveDeadline(e.target.value)} />
      </div>

      <div>
        <label className="label">Заметка</label>
        <textarea className="input min-h-20" value={note} onChange={(e) => setNote(e.target.value)} onBlur={saveNote} placeholder="Текст…" />
      </div>

      <div>
        <label className="label">Фото</label>
        <div className="grid grid-cols-3 gap-2">
          {node.imageUrls.map((url, i) => (
            <div key={url} className="group relative overflow-hidden rounded-lg border border-ink-800">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={url} alt="" className="h-16 w-full object-cover" />
              <button
                onClick={() => removeImage(node.imageKeys[i])}
                className="absolute right-0.5 top-0.5 grid h-5 w-5 place-items-center rounded bg-ink-950/80 text-xs text-ink-300 hover:text-bad"
                aria-label="Удалить фото"
              >
                ✕
              </button>
            </div>
          ))}
          <label className="grid h-16 cursor-pointer place-items-center rounded-lg border border-dashed border-ink-700 text-ink-500 hover:border-brand/50">
            {uploading ? "…" : <Plus size={16} />}
            <input type="file" accept="image/png,image/jpeg,image/webp" multiple className="hidden" onChange={onFiles} disabled={uploading} />
          </label>
        </div>
      </div>

      <button onClick={onDelete} className="btn-ghost w-full text-bad">
        Удалить ноду
      </button>
    </div>
  );
}

// GraphCanvas is the mind-map editor for one board (a project's canvas, or the
// root board of projects when board === "").
export function GraphCanvas({ board, title }: { board: string; title: string }) {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  if (!mounted) return <div className="card h-[60vh] animate-pulse bg-ink-800/40" />;
  return (
    <ReactFlowProvider>
      <Inner board={board} title={title} />
    </ReactFlowProvider>
  );
}
