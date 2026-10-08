"use client";

import { useCallback, useEffect, useState } from "react";
import { loadSession } from "@/services/api";
import type { Permission, User } from "@/types";

/** Usuário logado + permissões (atualiza quando a sessão muda em outra parte do app) */
export function useSession() {
  const [state, setState] = useState<{ user: User | null; permissions: Permission[]; ready: boolean }>({
    user: null,
    permissions: [],
    ready: false,
  });

  useEffect(() => {
    const sync = () => setState({ ...loadSession(), ready: true });
    sync();
    window.addEventListener("gymsync:session", sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener("gymsync:session", sync);
      window.removeEventListener("storage", sync);
    };
  }, []);

  const can = useCallback(
    (...perms: Permission[]) => state.user?.role === "ADMIN" || perms.some((p) => state.permissions.includes(p)),
    [state.user, state.permissions],
  );

  return { ...state, can };
}
