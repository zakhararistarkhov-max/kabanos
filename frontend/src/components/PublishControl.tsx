"use client";

import { Globe, Lock } from "lucide-react";

// PublishControl toggles a catalog item between private draft and public.
export function PublishControl({
  isPublic,
  pending,
  onToggle,
}: {
  isPublic: boolean;
  pending: boolean;
  onToggle: (publish: boolean) => void;
}) {
  return isPublic ? (
    <button onClick={() => onToggle(false)} disabled={pending} className="btn-ghost">
      <Lock size={16} /> Снять с публикации
    </button>
  ) : (
    <button onClick={() => onToggle(true)} disabled={pending} className="btn-primary">
      <Globe size={16} /> Опубликовать
    </button>
  );
}

// PublishBadge shows whether an item is published or a private draft.
export function PublishBadge({ isPublic }: { isPublic: boolean }) {
  return isPublic ? (
    <span className="inline-flex items-center gap-1 rounded-full bg-good/15 px-2 py-0.5 text-xs font-medium text-good">
      <Globe size={12} /> Опубликовано
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 rounded-full bg-warn/15 px-2 py-0.5 text-xs font-medium text-warn">
      <Lock size={12} /> Черновик
    </span>
  );
}
