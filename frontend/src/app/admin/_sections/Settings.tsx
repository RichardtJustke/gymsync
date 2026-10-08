"use client";

import React, { useEffect, useState } from "react";
import { Plus, Send } from "lucide-react";
import { api } from "@/services/api";
import { ConfirmDialog, Empty, Field, Modal, PillTabs, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { useToast } from "@/context/ToastContext";
import { brl, dateBR, dateTimeBR } from "@/lib/format";
import { ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import type { Equipment, Modality, Permission, Plan } from "@/types";

type Tab = "planos" | "modalidades" | "academia" | "avisos" | "equipamentos";

export default function Settings({ can }: { can: (...p: Permission[]) => boolean }) {
  const tabs = ([
    { id: "planos", label: "Planos", ok: can("catalog.manage") },
    { id: "modalidades", label: "Modalidades", ok: can("catalog.manage") },
    { id: "academia", label: "Academia e regras", ok: can("catalog.manage") },
    { id: "avisos", label: "Avisos", ok: can("announcements.send") },
    { id: "equipamentos", label: "Equipamentos", ok: can("equipment.manage") },
  ] as { id: Tab; label: string; ok: boolean }[]).filter((t) => t.ok);
  const [tab, setTab] = useState<Tab>(tabs[0]?.id || "planos");
  return (
    <div className="space-y-5">
      <PageHead title="Configurações" subtitle="Cadastros e regras da academia" />
      <PillTabs value={tab} onChange={setTab} tabs={tabs.map(({ id, label }) => ({ id, label }))} />
      {tab === "planos" && <Plans />}
      {tab === "modalidades" && <Modalities />}
      {tab === "academia" && <GymRules />}
      {tab === "avisos" && <Announcements />}
      {tab === "equipamentos" && <EquipmentList />}
    </div>
  );
}

/* -------------------------------- Planos -------------------------------- */
function Plans() {
  const { data, loading, error, reload } = useAsync(() => api.listPlans(true), []);
  const [edit, setEdit] = useState<Partial<Plan> | null>(null);
  const { busy, run } = useAction();
  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading />;
  return (
    <div className="space-y-4">
      <div className="flex justify-end"><button className="btn-lime" onClick={() => setEdit({ duration_months: 1, classes_per_week: 0, price: 0, active: true })}><Plus className="w-4 h-4" />Novo plano</button></div>
      <div className="grid gap-3 md:grid-cols-2">
        {data?.map(({ plan: p, students }) => (
          <div key={p.id} className={`card p-4 space-y-2 ${p.active ? "" : "opacity-60"}`}>
            <div className="flex items-center justify-between gap-3"><h3 className="text-sm font-semibold text-white">{p.name}</h3><Tag tone={p.active ? "ok" : "mute"}>{p.active ? "Ativo" : "Inativo"}</Tag></div>
            <div className="text-2xl font-semibold text-white">{brl(p.price)}<span className="text-xs text-ink-300 font-normal">/mês</span></div>
            <div className="text-xs text-ink-300">{p.classes_per_week ? `${p.classes_per_week} aula(s) por semana` : "Aulas ilimitadas"} · contrato de {p.duration_months} mês(es) · {students} aluno(s)</div>
            {p.description && <div className="text-xs text-ink-400">{p.description}</div>}
            <div className="flex gap-2 pt-1"><button className="btn-dark px-3 py-2 text-xs" onClick={() => setEdit(p)}>Editar</button>
              <button className="btn-ghost px-3 py-2 text-xs" disabled={busy} onClick={async () => { const r = await run(() => api.updatePlan(p.id, { ...p, active: !p.active }), p.active ? "Plano desativado" : "Plano reativado"); if (r) reload(); }}>{p.active ? "Desativar" : "Reativar"}</button></div>
          </div>
        ))}
      </div>
      {edit && <PlanModal initial={edit} onClose={() => setEdit(null)} onDone={reload} />}
    </div>
  );
}
function PlanModal({ initial, onClose, onDone }: { initial: Partial<Plan>; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [f, setF] = useState({ name: initial.name || "", description: initial.description || "", price: initial.price || 0, duration_months: initial.duration_months || 1, classes_per_week: initial.classes_per_week || 0, active: initial.active ?? true });
  return (
    <Modal title={initial.id ? "Editar plano" : "Novo plano"} onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || !f.name.trim()} onClick={async () => { const r = await run(() => (initial.id ? api.updatePlan(initial.id, f) : api.createPlan(f)), "Plano salvo"); if (r) { onDone(); onClose(); } }}>Salvar</button></>}>
      <Field label="Nome"><input className="field" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} autoFocus /></Field>
      <div className="grid grid-cols-3 gap-3">
        <Field label="Mensalidade (R$)"><NumberInput value={f.price} min={0} step={0.01} onChange={(n) => setF({ ...f, price: n })} /></Field>
        <Field label="Contrato (meses)"><NumberInput value={f.duration_months} min={1} onChange={(n) => setF({ ...f, duration_months: n })} /></Field>
        <Field label="Aulas/semana" hint="0 = ilimitado"><NumberInput value={f.classes_per_week} min={0} onChange={(n) => setF({ ...f, classes_per_week: n })} /></Field>
      </div>
      <Field label="Descrição"><input className="field" value={f.description} onChange={(e) => setF({ ...f, description: e.target.value })} /></Field>
      {initial.id && <p className="text-xs text-ink-400">Mudar o preço vale só para novas matrículas e trocas de plano; quem já está no plano mantém o valor atual.</p>}
    </Modal>
  );
}

/* ----------------------------- Modalidades ----------------------------- */
function Modalities() {
  const { data, loading, error, reload } = useAsync(() => api.listModalities(true), []);
  const [edit, setEdit] = useState<Partial<Modality> | null>(null);
  const { busy, run } = useAction();
  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading />;
  return (
    <div className="space-y-4">
      <div className="flex justify-end"><button className="btn-lime" onClick={() => setEdit({ color: "#D7FF3A", allow_trial: true, active: true })}><Plus className="w-4 h-4" />Nova modalidade</button></div>
      <div className="grid gap-3 md:grid-cols-2">
        {data?.map((m) => (
          <div key={m.id} className={`card p-4 flex items-center gap-3 ${m.active ? "" : "opacity-60"}`}>
            <span className="w-3 self-stretch rounded-full" style={{ background: m.color }} />
            <div className="flex-1 min-w-0"><div className="text-sm font-semibold text-white">{m.name}</div><div className="text-xs text-ink-300">{m.allow_trial ? "Aceita aula experimental" : "Sem aula experimental"}{m.description ? ` · ${m.description}` : ""}</div></div>
            <button className="btn-dark px-3 py-2 text-xs" onClick={() => setEdit(m)}>Editar</button>
            <button className="btn-ghost px-3 py-2 text-xs" disabled={busy} onClick={async () => { const r = await run(() => api.updateModality(m.id, { ...m, active: !m.active })); if (r) reload(); }}>{m.active ? "Desativar" : "Reativar"}</button>
          </div>
        ))}
      </div>
      {edit && <ModalityModal initial={edit} onClose={() => setEdit(null)} onDone={reload} />}
    </div>
  );
}
function ModalityModal({ initial, onClose, onDone }: { initial: Partial<Modality>; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [f, setF] = useState({ name: initial.name || "", description: initial.description || "", color: initial.color || "#D7FF3A", allow_trial: initial.allow_trial ?? true, active: initial.active ?? true });
  return (
    <Modal title={initial.id ? "Editar modalidade" : "Nova modalidade"} onClose={onClose} width="max-w-md"
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || !f.name.trim()} onClick={async () => { const r = await run(() => (initial.id ? api.updateModality(initial.id, f) : api.createModality(f)), "Modalidade salva"); if (r) { onDone(); onClose(); } }}>Salvar</button></>}>
      <Field label="Nome"><input className="field" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} autoFocus /></Field>
      <Field label="Descrição"><input className="field" value={f.description} onChange={(e) => setF({ ...f, description: e.target.value })} /></Field>
      <div className="flex items-end gap-4"><Field label="Cor na agenda"><input type="color" className="h-11 w-20 rounded-xl bg-ink-750 border-0 p-1" value={f.color} onChange={(e) => setF({ ...f, color: e.target.value })} /></Field>
        <label className="flex items-center gap-2 text-sm text-ink-200 pb-3"><input type="checkbox" checked={f.allow_trial} onChange={(e) => setF({ ...f, allow_trial: e.target.checked })} />Aparece no site para aula experimental</label></div>
    </Modal>
  );
}

/* ----------------------------- Academia e regras ----------------------------- */
function GymRules() {
  const { data, loading, error, reload } = useAsync(() => api.getGym(), []);
  const { busy, run } = useAction();
  const [f, setF] = useState({ name: "", phone: "", email: "", address: "", booking_window_days: 7, cancel_deadline_hours: 2, overdue_block_days: 0 });
  useEffect(() => { if (data) setF({ name: data.name, phone: data.phone || "", email: data.email || "", address: data.address || "", booking_window_days: data.booking_window_days, cancel_deadline_hours: data.cancel_deadline_hours, overdue_block_days: data.overdue_block_days }); }, [data]);
  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading />;
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section className="card p-5 space-y-3">
        <h3 className="section-title">Dados</h3>
        <Field label="Nome da academia"><input className="field" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} /></Field>
        <div className="grid grid-cols-2 gap-3"><Field label="Telefone"><input className="field" value={f.phone} onChange={(e) => setF({ ...f, phone: e.target.value })} /></Field><Field label="E-mail"><input className="field" value={f.email} onChange={(e) => setF({ ...f, email: e.target.value })} /></Field></div>
        <Field label="Endereço"><input className="field" value={f.address} onChange={(e) => setF({ ...f, address: e.target.value })} /></Field>
      </section>
      <section className="card p-5 space-y-3">
        <h3 className="section-title">Regras</h3>
        <Field label="Reservas abrem quantos dias antes" hint="1 a 60 dias"><NumberInput value={f.booking_window_days} min={1} max={60} onChange={(n) => setF({ ...f, booking_window_days: n })} /></Field>
        <Field label="Cancelar reserva até quantas horas antes" hint="0 a 72 horas"><NumberInput value={f.cancel_deadline_hours} min={0} max={72} onChange={(n) => setF({ ...f, cancel_deadline_hours: n })} /></Field>
        <Field label="Bloquear entrada após quantos dias de atraso" hint="0 = nunca bloqueia sozinho. Ao pagar, o acesso volta na hora."><NumberInput value={f.overdue_block_days} min={0} max={90} onChange={(n) => setF({ ...f, overdue_block_days: n })} /></Field>
      </section>
      <div className="lg:col-span-2 flex justify-end"><button className="btn-lime" disabled={busy} onClick={async () => { const r = await run(() => api.updateGym(f), "Configurações salvas"); if (r) reload(); }}>Salvar</button></div>
    </div>
  );
}

/* --------------------------------- Avisos --------------------------------- */
function Announcements() {
  const toast = useToast();
  const list = useAsync(() => api.listAnnouncements(), []);
  const mods = useAsync(() => api.listModalities(), []);
  const { busy, run } = useAction();
  const [f, setF] = useState({ kind: "AVISO", title: "", body: "", modality_id: "" });
  const send = async () => {
    const r = await run(() => api.createAnnouncement({ ...f, modality_id: f.modality_id || undefined }));
    if (r) { setF({ kind: "AVISO", title: "", body: "", modality_id: "" }); list.reload(); toast.success(`Enviado para ${r.sent_count} aluno(s).`); }
  };
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section className="card p-5 space-y-3">
        <h3 className="section-title">Novo aviso</h3>
        <PillTabs value={f.kind} onChange={(kind) => setF({ ...f, kind })} tabs={[{ id: "AVISO", label: "Aviso" }, { id: "EVENTO", label: "Evento" }]} />
        <Field label="Título"><input className="field" maxLength={120} value={f.title} onChange={(e) => setF({ ...f, title: e.target.value })} /></Field>
        <Field label="Mensagem"><textarea className="field" rows={4} maxLength={1000} value={f.body} onChange={(e) => setF({ ...f, body: e.target.value })} /></Field>
        <Field label="Quem recebe"><select className="field" value={f.modality_id} onChange={(e) => setF({ ...f, modality_id: e.target.value })}><option value="">Todos os alunos ativos</option>{mods.data?.map((m) => <option key={m.id} value={m.id}>Quem fez {m.name} (60 dias)</option>)}</select></Field>
        <button className="btn-lime w-full" disabled={busy || !f.title.trim() || !f.body.trim()} onClick={send}><Send className="w-4 h-4" />Enviar agora</button>
      </section>
      <section className="card p-5">
        <h3 className="section-title mb-3">Enviados</h3>
        {!list.data || list.data.length === 0 ? <Empty title="Nenhum aviso enviado" /> : (
          <ul className="space-y-2">{list.data.map((a) => <li key={a.id} className="card-soft p-3"><div className="flex items-center justify-between gap-2"><span className="text-sm font-medium text-white">{a.title}</span><Tag tone="mute">{a.kind === "EVENTO" ? "Evento" : "Aviso"}</Tag></div><div className="text-xs text-ink-300 mt-1 line-clamp-2">{a.body}</div><div className="text-[11px] text-ink-400 mt-1">{dateTimeBR(a.created_at)} · {a.sent_count} aluno(s)</div></li>)}</ul>
        )}
      </section>
    </div>
  );
}

/* ------------------------------ Equipamentos ------------------------------ */
const EQ_STATUS: Record<Equipment["status"], { label: string; tone: "ok" | "warn" | "bad" }> = { OPERACIONAL: { label: "Operacional", tone: "ok" }, ATENCAO: { label: "Atenção", tone: "warn" }, MANUTENCAO: { label: "Em manutenção", tone: "bad" } };
function EquipmentList() {
  const { data, loading, error, reload } = useAsync(() => api.listEquipment(), []);
  const [edit, setEdit] = useState<Partial<Equipment> | null>(null);
  const [del, setDel] = useState<Equipment | null>(null);
  const { busy, run } = useAction();
  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading />;
  return (
    <div className="space-y-4">
      <div className="flex justify-end"><button className="btn-lime" onClick={() => setEdit({ status: "OPERACIONAL" })}><Plus className="w-4 h-4" />Novo equipamento</button></div>
      {data && data.length === 0 ? <div className="card"><Empty title="Nenhum equipamento cadastrado" /></div> : (
        <div className="card divide-y divide-white/5">
          {data?.map((e) => { const st = EQ_STATUS[e.status]; const late = e.next_maintenance && new Date(e.next_maintenance) < new Date(); return (
            <div key={e.id} className="p-4 flex flex-wrap items-center gap-3">
              <div className="flex-1 min-w-[200px]"><div className="text-sm font-medium text-white">{e.name}</div><div className="text-xs text-ink-300">{e.sector || "Sem setor"}{e.notes ? ` · ${e.notes}` : ""}{e.next_maintenance ? ` · próxima revisão ${dateBR(e.next_maintenance)}` : ""}</div></div>
              {late && <Tag tone="bad">Revisão atrasada</Tag>}<Tag tone={st.tone}>{st.label}</Tag>
              <div className="flex gap-2"><button className="btn-dark px-3 py-2 text-xs" onClick={() => setEdit(e)}>Editar</button><button className="btn-ghost px-3 py-2 text-xs" onClick={() => setDel(e)}>Remover</button></div>
            </div>); })}
        </div>
      )}
      {edit && <EquipmentModal initial={edit} onClose={() => setEdit(null)} onDone={reload} />}
      {del && <ConfirmDialog title="Remover equipamento" danger confirmLabel="Remover" busy={busy} message={`${del.name} sai da lista.`} onClose={() => setDel(null)} onConfirm={async () => { const r = await run(() => api.deleteEquipment(del.id), "Removido"); if (r !== undefined) { setDel(null); reload(); } }} />}
    </div>
  );
}
function EquipmentModal({ initial, onClose, onDone }: { initial: Partial<Equipment>; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [f, setF] = useState({ name: initial.name || "", sector: initial.sector || "", status: initial.status || "OPERACIONAL", notes: initial.notes || "", last_maintenance: initial.last_maintenance?.slice(0, 10) || "", next_maintenance: initial.next_maintenance?.slice(0, 10) || "" });
  return (
    <Modal title={initial.id ? "Editar equipamento" : "Novo equipamento"} onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || !f.name.trim()} onClick={async () => { const r = await run(() => (initial.id ? api.updateEquipment(initial.id, f) : api.createEquipment(f)), "Equipamento salvo"); if (r) { onDone(); onClose(); } }}>Salvar</button></>}>
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="Nome"><input className="field" value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} autoFocus /></Field>
        <Field label="Setor"><input className="field" value={f.sector} onChange={(e) => setF({ ...f, sector: e.target.value })} /></Field>
        <Field label="Situação"><select className="field" value={f.status} onChange={(e) => setF({ ...f, status: e.target.value as Equipment["status"] })}>{Object.entries(EQ_STATUS).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}</select></Field>
        <div />
        <Field label="Última revisão"><input type="date" className="field" value={f.last_maintenance} onChange={(e) => setF({ ...f, last_maintenance: e.target.value })} /></Field>
        <Field label="Próxima revisão"><input type="date" className="field" value={f.next_maintenance} onChange={(e) => setF({ ...f, next_maintenance: e.target.value })} /></Field>
      </div>
      <Field label="Observações"><input className="field" value={f.notes} onChange={(e) => setF({ ...f, notes: e.target.value })} /></Field>
    </Modal>
  );
}
