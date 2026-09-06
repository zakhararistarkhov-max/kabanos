"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { authApi } from "@/lib/api";
import type { User } from "@/lib/types";

// useSession loads the current user from the BFF. A 401 is treated as
// "logged out" (user = null) rather than an error, so callers branch cleanly.
export function useSession() {
  return useQuery<{ user: User | null }>({
    queryKey: ["session"],
    queryFn: async () => {
      try {
        return await authApi<{ user: User | null }>("/session");
      } catch {
        return { user: null };
      }
    },
    staleTime: 60_000,
  });
}

export function useInvalidateSession() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ["session"] });
}
