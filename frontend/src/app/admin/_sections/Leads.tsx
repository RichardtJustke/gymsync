"use client";

import React, { useState } from "react";
import { Plus } from "lucide-react";
import { api } from "@/services/api";
import { ConfirmDialog, Empty, Field, Modal, PillTabs, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { brl, dateBR, dateTimeBR, whatsappLink } from "@/lib/format";
import { Credentials, ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import type { Lead, LeadStatus, Permission } from "@/types";

const STATUS: { id: LeadStatus; label: string; tone: "ok" | "bad" | "warn" | "mute" }[] = [
  { id: "NOVO", label: "Novo", tone: "warn" },
  { id: "AULA_AGENDADA", label: "Aula agendada", tone: "mute" },
  { id: "COMPARECEU", label: "Compareceu", tone: "mute" },
  { id: "MATRICULADA", label: "Matriculado", tone: "ok" },
  { id: "PERDIDA", label: "Perdido", tone: "bad" },
];
const SOURCES: Record<string, string> = { site: "Site", instagram: "Instagram", indicacao: "Indicação", recepcao: "Recepção", whatsapp: "WhatsApp" };

export default function Leads({ can }: { can: (...p: Permission[]) => boolean }) {
  const [status, setStatus] = useState<"" | LeadStatus>("");
  const [mine, setMine] = useState(false);
  const { data, loading, error, reload } = useAsync(() => api.listLeads({ status, assigned_to: mine ? "me" : "" }), [status, mine]);
  const staff = useAsync(async () => (can("goals.manage", "staff.manage") ? (await api.ranking()).rows.map((r) => ({ id: r.user_id, name: r.name })) : []), []);
  const [edit, setEdit] = useState<Lead | null>(null);
  const [convert, setConvert] = useState<Lead | null>(null);
  const [showNew, setShowNew] = useState(false);
  const { run } = useAction();

  const assign = async (l: Lead, id: string) => {
    const r = await run(() => api.updateLead(l.id, { assigned_to_id: id, notes: l.notes }), "Responsável atualizado");
    if (r) reload();
  };

  return (
    <div className="space-y-5">
      <PageHead title="Leads" subtitle="Interessados vindos do site, Instagram e recepção"
        actions={<><button className={mine ? "pill-on" : "pill-off"} onClick={() => setMine(!mine)}>Só os meus</button><button className="btn-lime" onClick={() => setShowNew(true)}><Plus className="w-4 h-4" />Novo lead</button></>} />

      <PillTabs
        tabs={[{ id: "" as "" | LeadStatus, label: `Todos` }, ...STATUS.map((s) => ({ id: s.id as "" | LeadStatus, label: `${s.label}${data?.counts[s.id] ? ` (${data.counts[s.id]})` : ""}` }))]}
        value={status} onChange={setStatus} />

      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading /> : (
        <div className="card overflow-hidden">
          {data && data.data.length === 0 ? <Empty title="Nenhum lead por aqui" /> : (
            <ul className="divide-y divide-white/5">
              {data?.data.map((l) => {
                const st = STATUS.find((s) => s.id === l.status)!;
                const wa = whatsappLink(l.phone, `Oi ${l.name.split(" ")[0]}! Aqui é da academia 💜 Vi que você se interessou, posso te ajudar?`);
                return (
                  <li key={l.id} className="p-4 flex flex-wrap items-center gap-3">
                    <div className="flex-1 min-w-[200px]">
                      <div className="text-sm font-medium text-white">{l.name}</div>
                      <div className="text-xs text-ink-300">{l.phone || l.email} · {SOURCES[l.source] || l.source} · {dateBR(l.created_at)}</div>
                      {l.trial_session && <div className="text-xs text-ink-200 mt-0.5">Aula experimental: {l.trial_session.modality?.name} · {dateTimeBR(l.trial_session.starts_at)}</div>}
                      {l.message && <div className="text-xs text-ink-400 mt-0.5 line-clamp-1">“{l.message}”</div>}
                    </div>
                    {staff.data && staff.data.length > 0 ? (
                      <select className="field w-auto py-2 text-xs" value={l.assigned_to_id || ""} onChange={(e) => assign(l, e.target.value)} aria-label="Responsável">
                        <option value="">Sem responsável</option>
                        {staff.data.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                      </select>
                    ) : <span className="text-xs text-ink-300">{l.assigned_to?.name || "Sem responsável"}</span>}
                    <Tag tone={st.tone}>{st.label}</Tag>
                    <div className="flex gap-2">
                      {wa && <a className="btn-dark px-3 py-2 text-xs" href={wa} target="_blank" rel="noreferrer">WhatsApp</a>}
                      <button className="btn-dark px-3 py-2 text-xs" onClick={() => setEdit(l)}>Detalhes</button>
                      {l.status !== "MATRICULADA" && l.status !== "PERDIDA" && can("students.manage", "sales.manage") && <button className="btn-lime px-3 py-2 text-xs" onClick={() => setConvert(l)}>Matricular</button>}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}

      {showNew && <NewLeadModal onClose={() => setShowNew(false)} onDone={reload} />}
      {edit && <LeadModal lead={edit} onClose={() => setEdit(null)} onDone={reload} />}
      {convert && <ConvertModal lead={convert} sellers={staff.data || []} onClose={() => setConvert(null)} onDone={reload} />}
    </div>
  );
}

function NewLeadModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [f, setF] = useState({ name: "", phone: "", email: "", source: "instagram", message: "" });
  return (
    <Modal title="Novo lead" onClose={onClose} width="max-w-md"
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || f.name.trim().length < 2} onClick={async () => { const r = await run(() => api.createLead(f), "Lead cadastrado"); if (r) { onDone(); onClose(); } }}>Cadastrar</button></>}>
      <Field label="Nome"><input className="field" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} autoFocus /></Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="WhatsApp"><input className="field" value={f.phone} onChange={(e) => setF({ ...f, phone: e.target.value })} /></Field>
        <Field label="Origem"><select className="field" value={f.source} onChange={(e) => setF({ ...f, source: e.target.value })}>{Object.entries(SOURCES).filter(([k]) => k !== "site").map(([k, v]) => <option key={k} value={k}>{v}</option>)}</select></Field>
      </div>
      <Field label="E-mail"><input className="field" value={f.email} onChange={(e) => setF({ ...f, email: e.target.value })} /></Field>
      <Field label="Observação"><textarea className="field" rows={2} value={f.message} onChange={(e) => setF({ ...f, message: e.target.value })} /></Field>
    </Modal>
  );
}

function LeadModal({ lead, onClose, onDone }: { lead: Lead; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [status, setStatus] = useState<LeadStatus>(lead.status);
  const [notes, setNotes] = useState(lead.notes || "");
  return (
    <Modal title={lead.name} onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Fechar</button><button className="btn-lime" disabled={busy} onClick={async () => { const r = await run(() => api.updateLead(lead.id, { status, notes }), "Lead atualizado"); if (r) { onDone(); onClose(); } }}>Salvar</button></>}>
      <div className="text-sm text-ink-200 space-y-1"><div>{lead.phone} {lead.email && `· ${lead.email}`}</div><div className="text-ink-300">Origem: {SOURCES[lead.source] || lead.source} · {dateBR(lead.created_at)}</div>{lead.message && <div>“{lead.message}”</div>}</div>
      <Field label="Situação">
        <div className="flex flex-wrap gap-2">{STATUS.filter((s) => s.id !== "MATRICULADA" || lead.status === "MATRICULADA").map((s) => <button key={s.id} type="button" onClick={() => setStatus(s.id)} className={status === s.id ? "pill-on" : "pill-off"}>{s.label}</button>)}</div>
      </Field>
      <Field label="Anotações do atendimento"><textarea className="field" rows={4} value={notes} onChange={(e) => setNotes(e.target.value)} /></Field>
    </Modal>
  );
}

function ConvertModal({ lead, sellers, onClose, onDone }: { lead: Lead; sellers: { id: string; name: string }[]; onClose: () => void; onDone: () => void }) {
  const plans = useAsync(() => api.listPlans(), []);
  const { busy, run } = useAction();
  const [f, setF] = useState({ email: lead.email, document: "", phone: lead.phone, plan_id: "", due_day: new Date().getDate(), seller_id: lead.assigned_to_id || "" });
  const [creds, setCreds] = useState<{ login: string; pass: string } | null>(null);
  if (creds) return (
    <Modal title={`${lead.name} matriculado(a)`} onClose={onClose} width="max-w-md" footer={<button className="btn-lime" onClick={onClose}>Concluir</button>}>
      <p className="text-sm text-ink-200">Primeira mensalidade gerada e venda registrada. Passe o acesso ao aluno:</p>
      <Credentials login={creds.login} password={creds.pass} />
    </Modal>
  );
  const plan = plans.data?.find((p) => p.plan.id === f.plan_id)?.plan;
  return (
    <Modal title={`Matricular ${lead.name}`} onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || !f.plan_id || (!f.email && !f.document)}
        onClick={async () => { const r = await run(() => api.convertLead(lead.id, { ...f, seller_id: f.seller_id || null })); if (r) { setCreds({ login: r.login, pass: r.temp_password }); onDone(); } }}>Matricular</button></>}>
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="E-mail"><input className="field" value={f.email} onChange={(e) => setF({ ...f, email: e.target.value })} /></Field>
        <Field label="CPF"><input className="field" value={f.document} onChange={(e) => setF({ ...f, document: e.target.value })} /></Field>
        <Field label="Telefone"><input className="field" value={f.phone} onChange={(e) => setF({ ...f, phone: e.target.value })} /></Field>
        <Field label="Dia do vencimento"><NumberInput value={f.due_day} min={1} max={31} onChange={(n) => setF({ ...f, due_day: n })} /></Field>
      </div>
      <Field label="Plano" hint={plan ? `1ª mensalidade ${brl(plan.price)} · contrato ${brl(plan.price * Math.max(plan.duration_months, 1))}` : undefined}>
        <select className="field" value={f.plan_id} onChange={(e) => setF({ ...f, plan_id: e.target.value })}><option value="">Escolha o plano…</option>{plans.data?.map(({ plan: p }) => <option key={p.id} value={p.id}>{p.name} — {brl(p.price)}</option>)}</select>
      </Field>
      {sellers.length > 0 && <Field label="Vendedor(a) (comissão)"><select className="field" value={f.seller_id} onChange={(e) => setF({ ...f, seller_id: e.target.value })}><option value="">Responsável pelo lead / eu</option>{sellers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}</select></Field>}
      <p className="text-xs text-ink-400">É preciso e-mail ou CPF para o aluno entrar no app.</p>
    </Modal>
  );
}
