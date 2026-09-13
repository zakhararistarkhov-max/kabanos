"use client";

import { useEffect, useState } from "react";

// useGridColumns returns how many columns the dashboard grid currently shows,
// matching the Tailwind breakpoints used on it (1 / sm:2 / lg:3).
export function useGridColumns(): 1 | 2 | 3 {
  const [cols, setCols] = useState<1 | 2 | 3>(3);

  useEffect(() => {
    const lg = window.matchMedia("(min-width: 1024px)");
    const sm = window.matchMedia("(min-width: 640px)");
    const update = () => setCols(lg.matches ? 3 : sm.matches ? 2 : 1);
    update();
    lg.addEventListener("change", update);
    sm.addEventListener("change", update);
    return () => {
      lg.removeEventListener("change", update);
      sm.removeEventListener("change", update);
    };
  }, []);

  return cols;
}
