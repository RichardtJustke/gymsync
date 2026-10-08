"use client";

import React, { useEffect, useState } from "react";
import { Download, Plus, ShieldCheck } from "lucide-react";
import { api, loadSession } from "@/services/api";
import { Avatar, ConfirmDialog, Empty, Field, Modal, MonthPicker, PillTabs, SectionHeader, StatCard, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { useToast } from "@/context/ToastContext";
import { brl, dateBR, dateTimeBR, downloadCSV, monthKey, monthLabel, num, ROLE_LABEL } from "@/lib/format";
import { Credentials, ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import type { PayrollRow, PayType, Permission, Role, StaffMember } from "@/types";

type Tab = "pessoas" | "permissoes" | "horas" | "folha";
const STAFF_ROLES: Role[] = ["ADMIN", "GERENTE", "RECEPCAO", "CONSULTOR", "PROFESSOR"];
const PAY_TYPES: { id: PayType; label: string; hint: string }[] = [
  { id: "FIXO", label: "Salário fixo", hint: "Valor mensal fixo" },
  { id: "HORA_AULA", label: "Por hora/aula", hint: "Paga pelas horas de aula dadas no mês" },
  { id: "MISTO", label: "Fixo + hora/aula", hint: "Salário fixo e mais as horas de aula" },
];

export default function Team() {
  const [tab, setTab] = useState<Tab>("pessoas");
  return (
    <div className="space-y-5">
      <PageHead title="Equipe" subtitle="Pessoas, o que cada cargo pode fazer, horas de aula e pagamento" />
      <PillTabs value={tab} onChange={setTab} tabs={[{ id: "pessoas", label: "Pessoas" }, { id: "permissoes", label: "Permissões" }, { id: "horas", label: "Horas/aula" }, { id: "folha", label: "Fechamento do mês" }]} />
      {tab === "pessoas" && <People />}
      {tab === "permissoes" && <Permissions />}
      {tab === "horas" && <Hours />}
      {tab === "folha" && <Payroll />}
    </div>
  );
}

/* ------------------------------- Pessoas ------------------------------- */
function People() {
  const toast = useToast();
  const [inactive, setInactive] = useState(false);
  const { data, loading, error, reload } = useAsync(() => api.listStaff(inactive), [inactive]);
  const [edit, setEdit] = useState<StaffMember | "new" | null>(null);
  const [off, setOff] = useState<StaffMember | null>(null);
  const [creds, setCreds] = useState<{ name: string; login: string; pass: string } | null>(null);
  const { busy, run } = useAction();
  const me = loadSession().user?.id;

  const reset = async (m: StaffMember) => {
    const r = await run(() => api.resetStaffPassword(m.id));
    if (r) setCreds({ name: m.name, login: r.login, pass: r.temp_password });
  };
  const toggle = async (m: StaffMember, active: boolean) => {
    const r = await run(() => api.setStaffActive(m.id, active), active ? "Acesso reativado" : "Acesso desativado");
    if (r) {
      if (!active && r.uncovered_classes) toast.info(`${r.uncovered_classes} aula(s) futuras ficaram sem professor(a). Ajuste na Agenda.`);
      setOff(null);
      reload();
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <label className="flex items-center gap-2 text-sm text-ink-200"><input type="checkbox" checked={inactive} onChange={(e) => setInactive(e.target.checked)} />Mostrar desativados</label>
        <button className="btn-lime" onClick={() => setEdit("new")}><Plus className="w-4 h-4" />Nova pessoa</button>
      </div>
      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading /> : (
        <div className="grid gap-3 md:grid-cols-2">
          {data?.map((m) => (
            <div key={m.id} className={`card p-4 space-y-3 ${m.active ? "" : "opacity-60"}`}>
              <div className="flex items-center gap-3">
                <Avatar name={m.name} size={46} tone={m.role === "ADMIN" ? "lime" : "dark"} />
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-semibold text-white truncate">{m.name}{m.id === me && <span className="text-ink-400 font-normal"> (você)</span>}</div>
                  <div className="text-xs text-ink-300 truncate">{m.job_title || ROLE_LABEL[m.role]} · {m.email}</div>
                </div>
                <Tag tone={m.active ? "ok" : "mute"}>{m.active ? ROLE_LABEL[m.role] : "Desativado"}</Tag>
              </div>
              <div className="grid grid-cols-3 gap-2 text-center text-xs">
                <div className="card-soft p-2"><div className="text-white font-semibold">{m.classes_month}</div><div className="text-ink-300">aulas no mês</div></div>
                <div className="card-soft p-2"><div className="text-white font-semibold">{m.sales_month}</div><div className="text-ink-300">vendas no mês</div></div>
                <div className="card-soft p-2"><div className="text-white font-semibold">{m.last_action ? dateBR(m.last_action) : "—"}</div><div className="text-ink-300">última ação</div></div>
              </div>
              <div className="flex flex-wrap gap-2">
                <button className="btn-dark px-3 py-2 text-xs" onClick={() => setEdit(m)}>Editar</button>
                {m.active && <button className="btn-dark px-3 py-2 text-xs" disabled={busy} onClick={() => reset(m)}>Nova senha</button>}
                {m.active && m.id !== me && <button className="btn-ghost px-3 py-2 text-xs" onClick={() => setOff(m)}>Desativar</button>}
                {!m.active && <button className="btn-lime px-3 py-2 text-xs" disabled={busy} onClick={() => toggle(m, true)}>Reativar</button>}
              </div>
            </div>
          ))}
        </div>
      )}
      {edit && <StaffModal member={edit === "new" ? null : edit} onClose={() => setEdit(null)} onDone={reload} onCreds={(c) => setCreds(c)} />}
      {off && <ConfirmDialog title="Desativar acesso" danger confirmLabel="Desativar" busy={busy}
        message={<>{off.name} perde o acesso agora (as sessões abertas são encerradas). O histórico e o fechamento do mês continuam. {off.role === "PROFESSOR" && "As aulas futuras dela(e) ficam sem professor(a)."}</>}
        onClose={() => setOff(null)} onConfirm={() => toggle(off, false)} />}
      {creds && <Modal title={`Acesso de ${creds.name}`} onClose={() => setCreds(null)} width="max-w-md" footer={<button className="btn-lime" onClick={() => setCreds(null)}>Concluir</button>}>
        <p className="text-sm text-ink-200">Passe estes dados à pessoa. Na primeira entrada ela cria uma senha nova.</p><Credentials login={creds.login} password={creds.pass} /></Modal>}
    </div>
  );
}

function StaffModal({ member, onClose, onDone, onCreds }: { member: StaffMember | null; onClose: () => void; onDone: () => void; onCreds: (c: { name: string; login: string; pass: string }) => void }) {
  const { busy, run } = useAction();
  const p = member?.profile;
  const [f, setF] = useState({
    name: member?.name || "", email: member?.email || "", document: member?.document || "", phone: member?.phone || "",
    role: (member?.role || "RECEPCAO") as Role, job_title: member?.job_title || "",
    pay_type: (p?.pay_type || "FIXO") as PayType, base_salary: p?.base_salary || 0, hourly_rate: p?.hourly_rate || 0,
    commission_percent: p?.commission_percent || 0, commission_fixed: p?.commission_fixed || 0,
    hire_date: p?.hire_date ? p.hire_date.slice(0, 10) : "", pix_key: p?.pix_key || "", notes: p?.notes || "",
  });
  const isOwner = loadSession().user?.role === "ADMIN";
  const roles = STAFF_ROLES.filter((r) => r !== "ADMIN" || isOwner);
  const set = <K extends keyof typeof f>(k: K, v: (typeof f)[K]) => setF((x) => ({ ...x, [k]: v }));
  const showSalary = f.pay_type !== "HORA_AULA";
  const showHourly = f.pay_type !== "FIXO";

  const save = async () => {
    const payload = {
      name: f.name, email: f.email, document: f.document, phone: f.phone, role: f.role, job_title: f.job_title,
      profile: { pay_type: f.pay_type, base_salary: f.base_salary, hourly_rate: f.hourly_rate, commission_percent: f.commission_percent, commission_fixed: f.commission_fixed, hire_date: f.hire_date, pix_key: f.pix_key, notes: f.notes },
    };
    if (member) {
      const r = await run(() => api.updateStaff(member.id, payload), "Cadastro atualizado");
      if (r) { onDone(); onClose(); }
    } else {
      const r = await run(() => api.createStaff(payload));
      if (r) { onDone(); onClose(); onCreds({ name: f.name, login: r.login, pass: r.temp_password }); }
    }
  };

  return (
    <Modal title={member ? `Editar ${member.name}` : "Nova pessoa da equipe"} onClose={onClose} width="max-w-2xl"
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || f.name.trim().length < 2 || !f.email} onClick={save}>{busy ? "Salvando…" : "Salvar"}</button></>}>
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="Nome completo"><input className="field" value={f.name} onChange={(e) => set("name", e.target.value)} autoFocus={!member} /></Field>
        <Field label="E-mail (é o login)"><input className="field" type="email" value={f.email} onChange={(e) => set("email", e.target.value)} /></Field>
        <Field label="CPF"><input className="field" value={f.document} onChange={(e) => set("document", e.target.value)} /></Field>
        <Field label="Telefone"><input className="field" value={f.phone} onChange={(e) => set("phone", e.target.value)} /></Field>
        <Field label="Cargo (define as permissões)"><select className="field" value={f.role} onChange={(e) => set("role", e.target.value as Role)}>{roles.map((r) => <option key={r} value={r}>{ROLE_LABEL[r]}</option>)}</select></Field>
        <Field label="Função exibida" hint="Ex.: Professora de Pilates"><input className="field" value={f.job_title} onChange={(e) => set("job_title", e.target.value)} /></Field>
      </div>

      <SectionHeader title="Remuneração" />
      <Field label="Como é paga" hint={PAY_TYPES.find((x) => x.id === f.pay_type)?.hint}>
        <div className="flex flex-wrap gap-2">{PAY_TYPES.map((t) => <button key={t.id} type="button" className={f.pay_type === t.id ? "pill-on" : "pill-off"} onClick={() => set("pay_type", t.id)}>{t.label}</button>)}</div>
      </Field>
      <div className="grid sm:grid-cols-2 gap-3">
        {showSalary && <Field label="Salário fixo (R$/mês)"><NumberInput value={f.base_salary} min={0} step={50} onChange={(n) => set("base_salary", n)} /></Field>}
        {showHourly && <Field label="Valor da hora/aula (R$)"><NumberInput value={f.hourly_rate} min={0} step={5} onChange={(n) => set("hourly_rate", n)} /></Field>}
        <Field label="Comissão (% da venda)" hint="Só conta para quem registra vendas"><NumberInput value={f.commission_percent} min={0} max={100} step={0.5} onChange={(n) => set("commission_percent", n)} /></Field>
        <Field label="Comissão fixa por venda (R$)"><NumberInput value={f.commission_fixed} min={0} step={1} onChange={(n) => set("commission_fixed", n)} /></Field>
        <Field label="Admissão"><input type="date" className="field" value={f.hire_date} onChange={(e) => set("hire_date", e.target.value)} /></Field>
        <Field label="Chave PIX para pagamento"><input className="field" value={f.pix_key} onChange={(e) => set("pix_key", e.target.value)} /></Field>
      </div>
      <p className="text-xs text-ink-400">Mudar valores vale daqui para frente: vendas já feitas e fechamentos já pagos não são recalculados.</p>
    </Modal>
  );
}

/* ----------------------------- Permissões ----------------------------- */
function Permissions() {
  const { data, loading, error, reload } = useAsync(() => api.getPermissions(), []);
  const { busy, run } = useAction();
  const [draft, setDraft] = useState<Record<string, Permission[]>>({});
  useEffect(() => {
    if (data) setDraft(Object.fromEntries(data.roles.map((r) => [r.role, r.permissions])));
  }, [data]);

  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading />;
  if (!data) return null;
  const groups = Array.from(new Set(data.catalog.map((c) => c.group)));

  const toggle = (role: Role, p: Permission) =>
    setDraft((d) => ({ ...d, [role]: d[role]?.includes(p) ? d[role].filter((x) => x !== p) : [...(d[role] || []), p] }));
  const dirty = (role: Role) => {
    const orig = data.roles.find((r) => r.role === role)?.permissions || [];
    const cur = draft[role] || [];
    return orig.length !== cur.length || orig.some((p) => !cur.includes(p));
  };
  const save = async (role: Role) => { const r = await run(() => api.setPermissions(role, draft[role] || []), "Permissões salvas"); if (r !== undefined) reload(); };
  const reset = async (role: Role) => { const r = await run(() => api.resetPermissions(role), "Permissões padrão restauradas"); if (r !== undefined) reload(); };

  return (
    <div className="space-y-4">
      <div className="card-soft p-4 flex gap-3 text-sm text-ink-200"><ShieldCheck className="w-5 h-5 text-lime shrink-0 mt-0.5" /><p>Cada cargo vê só o que estiver marcado. O(a) dono(a) sempre tem acesso a tudo. As mudanças valem na hora, sem a pessoa precisar sair do sistema.</p></div>
      <div className="grid gap-4 lg:grid-cols-2">
        {data.roles.map((role) => (
          <section key={role.role} className="card p-5 space-y-4">
            <div className="flex items-center justify-between gap-3">
              <div><h3 className="section-title">{role.label}</h3><p className="text-xs text-ink-300">{role.members} pessoa(s) ativa(s){role.custom ? " · personalizado" : " · padrão"}</p></div>
              {!role.editable && <Tag tone="ok">Acesso total</Tag>}
            </div>
            {groups.map((g) => (
              <div key={g}>
                <div className="eyebrow mb-1.5">{g}</div>
                <ul className="space-y-1">
                  {data.catalog.filter((c) => c.group === g).map((c) => {
                    const on = role.editable ? (draft[role.role] || []).includes(c.key) : true;
                    return (
                      <li key={c.key}>
                        <label className={`flex items-start gap-3 p-2 rounded-xl ${role.editable ? "hover:bg-white/5 cursor-pointer" : "opacity-70"}`}>
                          <input type="checkbox" className="mt-1" checked={on} disabled={!role.editable} onChange={() => toggle(role.role, c.key)} />
                          <span><span className="text-sm text-white">{c.label}</span><span className="block text-[11px] text-ink-400">{c.hint}</span></span>
                        </label>
                      </li>
                    );
                  })}
                </ul>
              </div>
            ))}
            {role.editable && (
              <div className="flex justify-end gap-2">
                {role.custom && <button className="btn-ghost" disabled={busy} onClick={() => reset(role.role)}>Voltar ao padrão</button>}
                <button className="btn-lime" disabled={busy || !dirty(role.role)} onClick={() => save(role.role)}>Salvar</button>
              </div>
            )}
          </section>
        ))}
      </div>
    </div>
  );
}

/* ----------------------------- Horas/aula ----------------------------- */
function Hours() {
  const [month, setMonth] = useState(monthKey());
  const { data, loading, error, reload } = useAsync(() => api.teacherHours(month), [month]);
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <MonthPicker value={month} onChange={setMonth} label={monthLabel} />
        {data && <button className="btn-dark" onClick={() => downloadCSV(`horas-aula-${month}.csv`, ["Professor(a)", "Aulas dadas", "Horas", "Aulas que cobriu", "Ocupação %", "Valor hora", "Total horas"], data.rows.map((r) => [r.name, r.given, r.hours, r.covered, Math.round(r.occupancy), r.hourly_rate, r.hours_amount]))}><Download className="w-4 h-4" />CSV</button>}
      </div>
      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading /> : !data || data.rows.length === 0 ? <div className="card"><Empty title="Sem aulas neste mês" /></div> : (
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-xs text-ink-300"><tr>{["Professor(a)", "Aulas dadas", "Horas", "Por vir", "Cobriu", "Ocupação", "Valor/hora", "Total"].map((h, i) => <th key={h} className={`p-3 font-medium ${i === 0 ? "text-left" : "text-right"}`}>{h}</th>)}</tr></thead>
            <tbody>
              {data.rows.map((r) => (
                <tr key={r.user_id} className="border-t border-white/5">
                  <td className="p-3 text-white">{r.name}{!r.active && <span className="text-ink-400"> (desativado)</span>}</td>
                  <td className="p-3 text-right">{r.given}</td><td className="p-3 text-right">{num(r.hours, 1)}h</td><td className="p-3 text-right text-ink-300">{r.upcoming}</td>
                  <td className="p-3 text-right">{r.covered}</td><td className="p-3 text-right">{Math.round(r.occupancy)}%</td>
                  <td className="p-3 text-right">{brl(r.hourly_rate)}</td><td className="p-3 text-right font-semibold text-white">{brl(r.hours_amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="text-xs text-ink-400">Conta só aulas já encerradas e não canceladas. “Cobriu” são aulas de outra turma/pessoa que ela(e) assumiu. “Ocupação” é presença ÷ vagas.</p>
    </div>
  );
}

/* ------------------------------ Fechamento ------------------------------ */
function Payroll() {
  const [month, setMonth] = useState(monthKey());
  const { data, loading, error, reload } = useAsync(() => api.payroll(month), [month]);
  const [row, setRow] = useState<PayrollRow | null>(null);
  const { busy, run } = useAction();

  const reopen = async (r: PayrollRow) => { const x = await run(() => api.savePayroll(r.user_id, { month, bonus: r.bonus, deductions: r.deductions, reopen: true }), "Fechamento reaberto"); if (x) reload(); };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <MonthPicker value={month} onChange={setMonth} label={monthLabel} />
        {data && <button className="btn-dark" onClick={() => downloadCSV(`fechamento-${month}.csv`, ["Nome", "Cargo", "Fixo", "Horas", "Valor horas", "Comissão", "Bônus", "Descontos", "Total", "Situação", "PIX"],
          data.rows.map((r) => [r.name, ROLE_LABEL[r.role], r.base_salary, r.hours, r.hours_amount, r.commission, r.bonus, r.deductions, r.total, r.status, r.pix_key]))}><Download className="w-4 h-4" />CSV</button>}
      </div>
      {data && (
        <div className="grid gap-4 sm:grid-cols-3">
          <StatCard icon={<ShieldCheck className="w-4 h-4" />} label="Total do mês" value={brl(data.total)} />
          <StatCard variant="lime" icon={<ShieldCheck className="w-4 h-4" />} label="Já pago" value={brl(data.paid)} />
          <StatCard icon={<ShieldCheck className="w-4 h-4" />} label="Falta pagar" value={brl(data.total - data.paid)} />
        </div>
      )}
      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading /> : (
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-xs text-ink-300"><tr>{["Pessoa", "Fixo", "Horas", "Comissão", "Bônus", "Descontos", "Total", ""].map((h, i) => <th key={h + i} className={`p-3 font-medium ${i === 0 ? "text-left" : "text-right"}`}>{h}</th>)}</tr></thead>
            <tbody>
              {data?.rows.map((r) => (
                <tr key={r.user_id} className="border-t border-white/5">
                  <td className="p-3"><div className="text-white">{r.name}</div><div className="text-[11px] text-ink-300">{r.job_title || ROLE_LABEL[r.role]}{r.pix_key ? ` · PIX ${r.pix_key}` : ""}</div></td>
                  <td className="p-3 text-right">{brl(r.base_salary)}</td>
                  <td className="p-3 text-right">{r.hours > 0 ? `${num(r.hours, 1)}h · ${brl(r.hours_amount)}` : "—"}</td>
                  <td className="p-3 text-right">{brl(r.commission)}</td><td className="p-3 text-right">{brl(r.bonus)}</td><td className="p-3 text-right">{brl(r.deductions)}</td>
                  <td className="p-3 text-right font-semibold text-white">{brl(r.total)}</td>
                  <td className="p-3 text-right whitespace-nowrap">
                    {r.status === "PAGO" ? <><Tag tone="ok">Pago {dateBR(r.paid_at)}</Tag> <button className="btn-ghost px-2 py-1 text-xs" disabled={busy} onClick={() => reopen(r)}>Reabrir</button></> : <button className="btn-lime px-3 py-2 text-xs" onClick={() => setRow(r)}>Fechar / pagar</button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="text-xs text-ink-400">Os valores acompanham as horas dadas e as vendas do mês até você marcar como pago. Depois de pago, o fechamento fica congelado.</p>
      {row && <PayrollModal row={row} month={month} onClose={() => setRow(null)} onDone={reload} />}
    </div>
  );
}

function PayrollModal({ row, month, onClose, onDone }: { row: PayrollRow; month: string; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [bonus, setBonus] = useState(row.bonus);
  const [deductions, setDeductions] = useState(row.deductions);
  const [notes, setNotes] = useState(row.notes || "");
  const total = row.base_salary + row.hours_amount + row.commission + bonus - deductions;
  const save = async (pay: boolean) => { const r = await run(() => api.savePayroll(row.user_id, { month, bonus, deductions, notes, mark_paid: pay }), pay ? "Marcado como pago" : "Ajustes salvos"); if (r) { onDone(); onClose(); } };
  return (
    <Modal title={`${row.name} — ${monthLabel(month)}`} onClose={onClose} width="max-w-md"
      footer={<><button className="btn-ghost" disabled={busy} onClick={() => save(false)}>Só salvar ajustes</button><button className="btn-lime" disabled={busy} onClick={() => save(true)}>Pagar {brl(total)}</button></>}>
      <ul className="text-sm space-y-1.5">
        <li className="flex justify-between"><span className="text-ink-300">Fixo</span><span className="text-white">{brl(row.base_salary)}</span></li>
        <li className="flex justify-between"><span className="text-ink-300">Horas/aula ({num(row.hours, 1)}h × {brl(row.hourly_rate)})</span><span className="text-white">{brl(row.hours_amount)}</span></li>
        <li className="flex justify-between"><span className="text-ink-300">Comissões</span><span className="text-white">{brl(row.commission)}</span></li>
      </ul>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Bônus (R$)"><NumberInput value={bonus} min={0} step={10} onChange={setBonus} /></Field>
        <Field label="Descontos (R$)"><NumberInput value={deductions} min={0} step={10} onChange={setDeductions} /></Field>
      </div>
      <Field label="Observação"><input className="field" value={notes} onChange={(e) => setNotes(e.target.value)} /></Field>
      {row.pix_key && <p className="text-xs text-ink-300">PIX para pagamento: <span className="text-white select-all">{row.pix_key}</span></p>}
    </Modal>
  );
}
