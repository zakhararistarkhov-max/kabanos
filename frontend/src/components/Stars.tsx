"use client";

import { useState } from "react";

// Star rating. Read-only when onRate is omitted; interactive otherwise.
export function Stars({
  value,
  onRate,
  size = "text-lg",
}: {
  value: number;
  onRate?: (rating: number) => void;
  size?: string;
}) {
  const [hover, setHover] = useState(0);
  const active = hover || value;
  const interactive = Boolean(onRate);

  return (
    <div className={`inline-flex ${size}`} role={interactive ? "radiogroup" : undefined}>
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          disabled={!interactive}
          onMouseEnter={() => interactive && setHover(n)}
          onMouseLeave={() => interactive && setHover(0)}
          onClick={() => onRate?.(n)}
          className={`${interactive ? "cursor-pointer" : "cursor-default"} px-0.5 transition ${
            n <= active ? "text-warn" : "text-ink-700"
          }`}
          aria-label={`${n}`}
        >
          ★
        </button>
      ))}
    </div>
  );
}
