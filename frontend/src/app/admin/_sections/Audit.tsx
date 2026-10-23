"use client";

import React, { useState } from "react";
import { Search } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, Empty, Tag } from "@/components/ui";
import { useAsync } from "@/lib/hooks";
import { dateTimeBR, ROLE_LABEL } from "@/lib/format";
import { ErrorBox, Loading, PageHead } from "./shared";

const ENTITIES: { id: string; label: string }[] = [
  { id: "", label: "Tudo" }, { id: "student", label: "Alunas" }, { id: "invoice", label: "Cobranças" }, { id: "sale", label: "Vendas" },
  { id: "staff", label: "Equipe" }, { id: "permissions", label: "Permissões" }, { id: "payroll", label: "Folha" }, { id: "session", label: "Aulas" },
  { id: "lead", label: "Leads" }, { id: "plan", label: "Planos" },
];

export default function Audit() {
  const [entity, setEntity] = useState("");
  const [q, setQ] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(1);
  const { data, loading, error, reload } = useAsync(() => api.audit({ entity, q, from, to, page }), [entity, q, from, to, page]);
  const pages = data ? Math.max(1, Math.ceil(data.total / data.limit)) : 1;

  return (
    <div className="space-y-5">
      <PageHead title="Histórico de ações" subtitle="Quem fez o quê no sistema: descontos, cancelamentos, trocas e permissões" />
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px]"><Search className="w-4 h-4 absolute left-4 top-1/2 -translate-y-1/2 text-ink-400" /><input className="field pl-11" placeholder="Buscar no texto (ex.: nome da aluna)" value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} /></div>
        <select className="field w-auto" value={entity} onChange={(e) => { setEntity(e.target.value); setPage(1); }} aria-label="Tipo">{ENTITIES.map((e) => <option key={e.id} value={e.id}>{e.label}</option>)}</select>
        <input type="date" className="field w-auto" value={from} onChange={(e) => { setFrom(e.target.value); setPage(1); }} aria-label="De" />
        <input type="date" className="field w-auto" value={to} onChange={(e) => { setTo(e.target.value); setPage(1); }} aria-label="Até" />
      </div>
      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading /> : (
        <div className="card overflow-hidden">
          {data && data.data.length === 0 ? <Empty title="Nada registrado" /> : (
            <ul className="divide-y divide-white/5">
              {data?.data.map((a) => (
                <li key={a.id} className="p-4 flex items-start gap-3">
                  <Avatar name={a.user_name} size={36} tone="dark" />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm text-white">{a.summary}</div>
                    <div className="text-xs text-ink-300 mt-0.5">{a.user_name} · {ROLE_LABEL[a.user_role] || a.user_role} · {dateTimeBR(a.created_at)}</div>
                  </div>
                  <Tag tone="mute">{a.action}</Tag>
                </li>
              ))}
            </ul>
          )}
          {pages > 1 && (
            <div className="flex items-center justify-between p-3 border-t border-white/5 text-xs text-ink-300">
              <span>Página {page} de {pages} · {data?.total} registro(s)</span>
              <div className="flex gap-2"><button className="btn-dark px-4 py-2" disabled={page <= 1} onClick={() => setPage(page - 1)}>Anterior</button><button className="btn-dark px-4 py-2" disabled={page >= pages} onClick={() => setPage(page + 1)}>Próxima</button></div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
