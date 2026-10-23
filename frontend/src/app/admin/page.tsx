"use client";

import React, { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  BarChart3, CalendarDays, FileClock, Handshake, LayoutDashboard, Megaphone, MoreHorizontal, QrCode, Settings as SettingsIcon, UserPlus, Users, UsersRound, Wallet,
} from "lucide-react";
import { api, getToken, homeFor } from "@/services/api";
import { useSession } from "@/lib/session";
import { Dock, Empty, Modal } from "@/components/ui";
import type { Permission } from "@/types";
import Overview from "./_sections/Overview";
import Students from "./_sections/Students";
import Agenda from "./_sections/Agenda";
import Reception from "./_sections/Reception";
import Leads from "./_sections/Leads";
import Sales from "./_sections/Sales";
import Finance from "./_sections/Finance";
import Team from "./_sections/Team";
import Reports from "./_sections/Reports";
import Audit from "./_sections/Audit";
import Settings from "./_sections/Settings";

type Section = { id: string; label: string; icon: React.ReactNode; perms: Permission[]; group: string };

const SECTIONS: Section[] = [
  { id: "visao", label: "Visão geral", icon: <LayoutDashboard className="w-[18px] h-[18px]" />, perms: ["dashboard.view"], group: "Dia a dia" },
  { id: "recepcao", label: "Recepção", icon: <QrCode className="w-[18px] h-[18px]" />, perms: ["checkin.manage"], group: "Dia a dia" },
  { id: "alunos", label: "Alunas", icon: <Users className="w-[18px] h-[18px]" />, perms: ["students.view", "students.manage"], group: "Dia a dia" },
  { id: "agenda", label: "Agenda", icon: <CalendarDays className="w-[18px] h-[18px]" />, perms: ["agenda.view", "agenda.manage"], group: "Dia a dia" },
  { id: "leads", label: "Leads", icon: <UserPlus className="w-[18px] h-[18px]" />, perms: ["leads.manage"], group: "Comercial" },
  { id: "vendas", label: "Vendas e metas", icon: <Handshake className="w-[18px] h-[18px]" />, perms: ["sales.manage", "goals.manage"], group: "Comercial" },
  { id: "financeiro", label: "Financeiro", icon: <Wallet className="w-[18px] h-[18px]" />, perms: ["finance.view"], group: "Gestão" },
  { id: "equipe", label: "Equipe", icon: <UsersRound className="w-[18px] h-[18px]" />, perms: ["staff.manage"], group: "Gestão" },
  { id: "relatorios", label: "Relatórios", icon: <BarChart3 className="w-[18px] h-[18px]" />, perms: ["reports.view"], group: "Gestão" },
  { id: "historico", label: "Histórico", icon: <FileClock className="w-[18px] h-[18px]" />, perms: ["audit.view"], group: "Gestão" },
  { id: "config", label: "Configurações", icon: <SettingsIcon className="w-[18px] h-[18px]" />, perms: ["catalog.manage", "announcements.send", "equipment.manage"], group: "Gestão" },
];

function AdminShell() {
  const router = useRouter();
  const params = useSearchParams();
  const { user, ready, can } = useSession();
  const [moreOpen, setMoreOpen] = useState(false);

  // Sessão: sem login volta para o início; aluno/professor vão para o app deles; permissões são atualizadas do servidor
  useEffect(() => {
    if (!ready) return;
    if (!getToken() || !user) { router.replace("/"); return; }
    if (user.role === "ALUNO" || user.role === "PROFESSOR") { router.replace(homeFor(user.role)); return; }
    api.getMe().catch(() => {});
  }, [ready, user?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const allowed = useMemo(() => SECTIONS.filter((s) => can(...s.perms)), [can]);
  const wanted = params.get("tab");
  const current = allowed.find((s) => s.id === wanted) || allowed[0];
  const go = (id: string) => { setMoreOpen(false); router.replace(`/admin?tab=${id}`, { scroll: false }); window.scrollTo({ top: 0 }); };

  if (!ready || !user || user.role === "ALUNO" || user.role === "PROFESSOR") {
    return <div className="flex-1 flex items-center justify-center text-ink-300 text-sm">Carregando…</div>;
  }
  if (allowed.length === 0) {
    return <div className="max-w-xl mx-auto p-8"><div className="card p-8"><Empty icon={<Megaphone className="w-6 h-6" />} title="Seu cargo ainda não tem acesso a nenhuma área">Peça à gestão para liberar as permissões do seu cargo.</Empty></div></div>;
  }

  const groups = Array.from(new Set(allowed.map((s) => s.group)));
  // Celular: as 4 primeiras áreas ficam na dock, o resto vai para "Mais"
  const docked = allowed.length <= 5 ? allowed : allowed.slice(0, 4);
  const extra = allowed.filter((s) => !docked.includes(s));
  const view = (() => {
    switch (current.id) {
      case "visao": return <Overview can={can} go={go} />;
      case "recepcao": return <Reception />;
      case "alunos": return <Students can={can} />;
      case "agenda": return <Agenda can={can} />;
      case "leads": return <Leads can={can} />;
      case "vendas": return <Sales can={can} />;
      case "financeiro": return <Finance can={can} />;
      case "equipe": return <Team />;
      case "relatorios": return <Reports />;
      case "historico": return <Audit />;
      case "config": return <Settings can={can} />;
      default: return null;
    }
  })();

  return (
    <div className="flex-1 w-full max-w-[1400px] mx-auto px-4 sm:px-6 pt-4 sm:pt-6 pb-dock lg:pb-6 lg:flex lg:gap-8">
      <nav aria-label="Áreas da gestão" className="hidden lg:block lg:w-60 lg:shrink-0">
        {/* celular: dock inferior; desktop: menu lateral agrupado */}
        <div className="sticky top-24 space-y-5">
          <div className="px-3"><div className="eyebrow">{user.gym?.name || "Gestão"}</div><div className="text-sm font-semibold text-white mt-0.5 truncate">{user.name}</div></div>
          {groups.map((g) => (
            <div key={g} className="space-y-1">
              <div className="eyebrow px-3 pb-1">{g}</div>
              {allowed.filter((s) => s.group === g).map((s) => (
                <button key={s.id} onClick={() => go(s.id)} aria-current={current.id === s.id ? "page" : undefined}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-2xl text-sm font-medium transition-colors ${current.id === s.id ? "bg-brand text-black" : "text-ink-200 hover:bg-white/5 hover:text-white"}`}>
                  {s.icon}{s.label}
                </button>
              ))}
            </div>
          ))}
        </div>
      </nav>
      <main className="flex-1 min-w-0" key={current.id}>{view}</main>

      <Dock
        items={[
          ...docked.map(({ id, label, icon }) => ({ id, label: label.split(" ")[0], icon })),
          ...(extra.length ? [{ id: "__mais", label: "Mais", icon: <MoreHorizontal className="w-[18px] h-[18px]" /> }] : []),
        ]}
        value={docked.some((s) => s.id === current.id) ? current.id : "__mais"}
        onChange={(id) => (id === "__mais" ? setMoreOpen(true) : go(id))}
      />
      {moreOpen && (
        <Modal title="Todas as áreas" onClose={() => setMoreOpen(false)}>
          {groups.map((g) => (
            <div key={g} className="space-y-1.5">
              <div className="eyebrow px-1">{g}</div>
              <div className="grid grid-cols-2 gap-2">
                {allowed.filter((s) => s.group === g).map((s) => (
                  <button key={s.id} onClick={() => go(s.id)}
                    className={`flex items-center gap-2.5 px-3.5 py-3 rounded-2xl text-sm font-medium text-left transition-colors ${current.id === s.id ? "bg-brand text-black" : "bg-ink-750 text-ink-100 active:bg-ink-700"}`}>
                    {s.icon}<span className="truncate">{s.label}</span>
                  </button>
                ))}
              </div>
            </div>
          ))}
        </Modal>
      )}
    </div>
  );
}

export default function AdminPage() {
  return (
    <Suspense fallback={<div className="flex-1 flex items-center justify-center text-ink-300 text-sm">Carregando…</div>}>
      <AdminShell />
    </Suspense>
  );
}
