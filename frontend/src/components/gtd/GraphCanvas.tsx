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
  useAddEdge,
  useDeleteEdge,
  useDeleteNode,
  useGraph,
  useMoveNode,
} from "@/hooks/useGtdGraph";
import { NodeEditor } from "@/components/gtd/NodeEditor";
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

type EditorState = { mode: "new"; kind: GtdNodeKind } | { mode: "edit"; nodeId: string } | null;

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
  const moveNode = useMoveNode();
  const deleteNode = useDeleteNode();
  const addEdgeMut = useAddEdge();
  const deleteEdgeMut = useDeleteEdge();
  const rf = useReactFlow();

  const [nodes, setNodes, onNodesChange] = useNodesState<AppNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [summary, setSummary] = useState<GtdGraphSummary | null>(null);
  const [editor, setEditor] = useState<EditorState>(null);

  const refresh = useCallback(() => {
    qc.invalidateQueries({ queryKey: ["gtd", "graph", board || "root"] });
    qc.invalidateQueries({ queryKey: ["gtd", "graph", "boards"] });
  }, [qc, board]);

  const openProject = useCallback((projectId: string) => router.push(`/gtd/graph/${projectId}`), [router]);
  const selectNode = useCallback((id: string) => setEditor({ mode: "edit", nodeId: id }), []);

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

  const nodeTypes = useMemo<NodeTypes>(() => ({ gtd: GtdNode }), []);

  const editorNode =
    editor?.mode === "edit" ? graph.data?.nodes.find((n) => n.id === editor.nodeId) ?? null : null;
  const editorKind: GtdNodeKind = editor?.mode === "new" ? editor.kind : editorNode?.kind ?? "task";

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-2xl font-bold">{title}</h1>
          {summary ? <SummaryBar s={summary} /> : null}
        </div>
        <div className="flex flex-wrap gap-2">
          <button onClick={() => setEditor({ mode: "new", kind: "task" })} className="btn-ghost">
            <Plus size={15} /> Задача
          </button>
          <button onClick={() => setEditor({ mode: "new", kind: "project" })} className="btn-ghost">
            <Plus size={15} /> Подпроект
          </button>
          <button onClick={() => setEditor({ mode: "new", kind: "note" })} className="btn-ghost">
            <Plus size={15} /> Заметка
          </button>
        </div>
      </div>

      <div className="h-[calc(100vh-18rem)] min-h-[26rem] overflow-hidden rounded-2xl border border-ink-800 bg-ink-950/40">
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

      {editor ? (
        <NodeEditor
          board={board}
          kind={editorKind}
          node={editorNode}
          onClose={() => setEditor(null)}
          onCreated={(id) => setEditor({ mode: "edit", nodeId: id })}
          onChanged={refresh}
        />
      ) : null}

      {graph.isLoading ? <p className="text-sm text-ink-500">Загрузка…</p> : null}
      {nodes.length === 0 && !graph.isLoading ? (
        <p className="text-sm text-ink-500">Пусто. Добавьте задачу, подпроект или заметку кнопками выше. Клик по ноде — редактор (дедлайн, заметка, фото).</p>
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
