"use client";

import React, { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api, getToken, homeFor } from "@/services/api";
import { useSession } from "@/lib/session";
import { Empty, PillTabs } from "@/components/ui";
import Classes from "./_classes";
import Prescription from "./_prescription";

type Tab = "aulas" | "treinos";

function ProfessorShell() {
  const router = useRouter();
  const params = useSearchParams();
  const { user, ready, can } = useSession();

  useEffect(() => {
    if (!ready) return;
    if (!getToken() || !user) { router.replace("/"); return; }
    if (user.role === "ALUNO") { router.replace(homeFor("ALUNO")); return; }
    api.getMe().catch(() => {});
  }, [ready, user?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const tabs = ([
    { id: "aulas" as Tab, label: "Minhas aulas", ok: can("agenda.view", "agenda.manage") },
    { id: "treinos" as Tab, label: "Treinos", ok: can("workouts.manage") },
  ]).filter((t) => t.ok);

  if (!ready || !user || user.role === "ALUNO") return <div className="flex-1 flex items-center justify-center text-ink-300 text-sm">Carregando…</div>;
  if (tabs.length === 0) return <div className="max-w-xl mx-auto p-8"><div className="card p-8"><Empty title="Seu cargo ainda não tem acesso a aulas ou treinos">Peça à gestão para liberar.</Empty></div></div>;

  const wanted = params.get("tab") as Tab | null;
  const current = tabs.find((t) => t.id === wanted)?.id || tabs[0].id;
  const go = (t: Tab) => router.replace(`/professor?tab=${t}`, { scroll: false });

  return (
    <div className="flex-1 w-full">
      {tabs.length > 1 && <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6"><PillTabs tabs={tabs.map(({ id, label }) => ({ id, label }))} value={current} onChange={go} /></div>}
      {current === "aulas" ? <Classes /> : <Prescription />}
    </div>
  );
}

export default function ProfessorPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-ink-300">Carregando…</div>}>
      <ProfessorShell />
    </Suspense>
  );
}
