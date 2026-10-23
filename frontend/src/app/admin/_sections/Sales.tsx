"use client";

import React, { useState } from "react";
import { Plus, Target, Trophy } from "lucide-react";
import { api, loadSession } from "@/services/api";
import { Avatar, ConfirmDialog, Empty, Field, Modal, MonthPicker, Progress, SectionHeader, StatCard, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { brl, dateBR, monthKey, monthLabel, ROLE_LABEL } from "@/lib/format";
import { ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import type { Permission, RankingRow, Sale, SaleKind } from "@/types";

const KINDS: { id: SaleKind; label: string }[] = [
  { id: "RENOVACAO", label: "Renovação" },
  { id: "UPGRADE", label: "Troca de plano" },
  { id: "MATRICULA", label: "Nova matrícula" },
  { id: "AVULSA", label: "Venda avulsa" },
];

export default function Sales({ can }: { can: (...p: Permission[]) => boolean }) {
  const [month, setMonth] = useState(monthKey());
  const ranking = useAsync(() => api.ranking(month), [month]);
  const sales = useAsync(() => api.listSales(month), [month]);
  const [showNew, setShowNew] = useState(false);
  const [goalFor, setGoalFor] = useState<{ id: string | null; name: string; row?: RankingRow } | null>(null);
  const [reverse, setReverse] = useState<Sale | null>(null);
  const { busy, run } = useAction();
  const allSeeing = can("goals.manage");
  const me = loadSession().user?.id;
  const r = ranking.data;
  const refresh = () => { ranking.reload(); sales.reload(); };

  return (
    <div className="space-y-6">
      <PageHead title="Vendas e metas" subtitle="Matrículas, renovações, comissões e desempenho da equipe comercial"
        actions={<><MonthPicker value={month} onChange={setMonth} label={monthLabel} /><button className="btn-brand" onClick={() => setShowNew(true)}><Plus className="w-4 h-4" />Registrar venda</button></>} />

      {ranking.error ? <ErrorBox message={ranking.error} onRetry={ranking.reload} /> : !r ? <Loading rows={2} /> : (
        <>
          <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
            <StatCard icon={<Trophy className="w-4 h-4" />} label="Vendas no mês" value={r.total_sales} hint={r.gym_goal?.target_sales ? `meta ${r.gym_goal.target_sales}` : "sem meta da academia"} progress={r.gym_goal ? r.gym_progress : undefined} />
            <StatCard variant="brand" icon={<Target className="w-4 h-4" />} label="Faturamento vendido" value={brl(r.total_amount)} hint={r.gym_goal?.target_revenue ? `meta ${brl(r.gym_goal.target_revenue)}` : undefined} />
            <StatCard icon={<Trophy className="w-4 h-4" />} label="Comissões" value={allSeeing ? brl(r.total_commission) : "—"} hint={allSeeing ? "a pagar no fechamento" : "visível só para a gestão"} />
            <div className="card p-5 flex flex-col justify-between gap-3">
              <div className="text-sm text-ink-200">Meta da academia</div>
              {allSeeing ? <button className="btn-dark" onClick={() => setGoalFor({ id: null, name: "da academia" })}>{r.gym_goal ? "Ajustar meta" : "Definir meta"}</button> : <span className="text-xs text-ink-400">Definida pela gestão</span>}
              {r.gym_goal && <Progress value={r.gym_progress} />}
            </div>
          </div>

          <section className="card p-5">
            <SectionHeader title="Ranking do mês" />
            {r.rows.length === 0 ? <Empty title="Sem vendas neste mês" /> : (
              <ul className="space-y-3">
                {r.rows.map((row, i) => (
                  <li key={row.user_id} className="card-soft p-4">
                    <div className="flex flex-wrap items-center gap-3">
                      <span className="w-6 text-center text-sm font-semibold text-ink-300">{i + 1}º</span>
                      <Avatar name={row.name} size={40} tone={i === 0 ? "brand" : "dark"} />
                      <div className="flex-1 min-w-[150px]">
                        <div className="text-sm font-medium text-white">{row.name}{row.user_id === me && <span className="text-ink-400"> (você)</span>}</div>
                        <div className="text-xs text-ink-300">{ROLE_LABEL[row.role]} · {row.new_sales} matrícula(s) · {row.renewals} renovação(ões)</div>
                      </div>
                      <div className="text-right"><div className="text-sm font-semibold text-white">{brl(row.revenue)}</div><div className="text-[11px] text-ink-300">{row.sales} venda(s){(allSeeing || row.user_id === me) ? ` · comissão ${brl(row.commission)}` : ""}</div></div>
                      {row.leads_assigned > 0 && <Tag tone="mute">{row.leads_converted}/{row.leads_assigned} leads · {Math.round(row.conversion)}%</Tag>}
                      {allSeeing && <button className="btn-ghost px-3 py-2 text-xs" onClick={() => setGoalFor({ id: row.user_id, name: `de ${row.name}`, row })}>Meta</button>}
                    </div>
                    {(row.goal_revenue > 0 || row.goal_sales > 0) && (
                      <div className="mt-3 flex items-center gap-3"><Progress value={row.progress} /><span className="text-xs text-ink-200 w-32 text-right">{Math.round(row.progress)}% da meta</span></div>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}

      <section className="card p-5">
        <SectionHeader title="Vendas registradas" />
        {sales.error ? <ErrorBox message={sales.error} onRetry={sales.reload} /> : !sales.data ? <Loading rows={2} /> : sales.data.length === 0 ? <Empty title="Nenhuma venda" /> : (
          <ul className="divide-y divide-white/5">
            {sales.data.map((s) => (
              <li key={s.id} className={`py-3 flex flex-wrap items-center gap-3 ${s.status === "ESTORNADA" ? "opacity-50" : ""}`}>
                <div className="flex-1 min-w-[200px]">
                  <div className="text-sm text-white">{s.student?.user?.name || s.description}</div>
                  <div className="text-xs text-ink-300">{KINDS.find((k) => k.id === s.kind)?.label} · {s.description} · {s.seller?.name} · {dateBR(s.created_at)}</div>
                </div>
                <div className="text-sm font-semibold text-white">{brl(s.amount)}</div>
                <div className="text-xs text-ink-300 w-24 text-right">com. {brl(s.commission)}</div>
                {s.status === "ESTORNADA" ? <Tag tone="mute">Estornada</Tag> : allSeeing && <button className="btn-ghost px-3 py-2 text-xs" onClick={() => setReverse(s)}>Estornar</button>}
              </li>
            ))}
          </ul>
        )}
      </section>

      {showNew && <NewSaleModal can={can} onClose={() => setShowNew(false)} onDone={refresh} />}
      {goalFor && <GoalModal month={month} target={goalFor} onClose={() => setGoalFor(null)} onDone={refresh} />}
      {reverse && <ConfirmDialog title="Estornar venda" danger confirmLabel="Estornar" askReason="Motivo do estorno" reasonRequired busy={busy}
        message={`${reverse.description} — ${brl(reverse.amount)}. Ela sai do ranking e da comissão.`} onClose={() => setReverse(null)}
        onConfirm={async (reason) => { const x = await run(() => api.reverseSale(reverse.id, reason), "Venda estornada"); if (x !== undefined) { setReverse(null); refresh(); } }} />}
    </div>
  );
}

function NewSaleModal({ can, onClose, onDone }: { can: (...p: Permission[]) => boolean; onClose: () => void; onDone: () => void }) {
  const plans = useAsync(() => api.listPlans(), []);
  const sellers = useAsync(async () => (can("goals.manage") ? (await api.ranking()).rows.map((r) => ({ id: r.user_id, name: r.name })) : []), []);
  const { busy, run } = useAction();
  const [search, setSearch] = useState("");
  const found = useAsync(() => (search.trim().length >= 2 ? api.listStudents(search, "", 1, 6) : Promise.resolve(null)), [search]);
  const [student, setStudent] = useState<{ id: string; name: string } | null>(null);
  const [f, setF] = useState({ kind: "RENOVACAO" as SaleKind, plan_id: "", amount: 0, description: "", seller_id: "", apply_plan: true });
  const plan = plans.data?.find((p) => p.plan.id === f.plan_id)?.plan;
  const contract = plan ? plan.price * Math.max(plan.duration_months, 1) : 0;

  const submit = async () => {
    const r = await run(() => api.createSale({
      kind: f.kind, student_id: student?.id, plan_id: f.plan_id || undefined, amount: f.amount || undefined,
      description: f.description || undefined, seller_id: f.seller_id || undefined, apply_plan: !!student && !!f.plan_id && f.apply_plan && (f.kind === "RENOVACAO" || f.kind === "UPGRADE"),
    }), "Venda registrada");
    if (r) { onDone(); onClose(); }
  };

  return (
    <Modal title="Registrar venda" onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-brand" disabled={busy || (!f.plan_id && f.amount <= 0)} onClick={submit}>Registrar</button></>}>
      <Field label="Tipo"><div className="flex flex-wrap gap-2">{KINDS.map((k) => <button key={k.id} type="button" className={f.kind === k.id ? "pill-on" : "pill-off"} onClick={() => setF({ ...f, kind: k.id })}>{k.label}</button>)}</div></Field>
      {student ? (
        <div className="card-soft p-3 flex items-center justify-between text-sm"><span className="text-white">{student.name}</span><button className="btn-ghost px-3 py-1.5" onClick={() => setStudent(null)}>Trocar</button></div>
      ) : (
        <Field label="Aluna (opcional)" hint="Matrículas novas são registradas automaticamente ao cadastrar a aluna">
          <input className="field" placeholder="Digite o nome" value={search} onChange={(e) => setSearch(e.target.value)} />
          {found.data && <ul className="mt-2 space-y-1">{found.data.data.map((s) => <li key={s.id}><button className="w-full text-left p-2.5 rounded-xl hover:bg-white/5 text-sm text-white" onClick={() => setStudent({ id: s.id, name: s.user.name })}>{s.user.name}</button></li>)}</ul>}
        </Field>
      )}
      <Field label="Plano" hint={plan ? `Contrato: ${brl(contract)}` : undefined}>
        <select className="field" value={f.plan_id} onChange={(e) => setF({ ...f, plan_id: e.target.value })}><option value="">Sem plano</option>{plans.data?.map(({ plan: p }) => <option key={p.id} value={p.id}>{p.name} — {brl(p.price)}</option>)}</select>
      </Field>
      {student && f.plan_id && (f.kind === "RENOVACAO" || f.kind === "UPGRADE") && (
        <label className="flex items-center gap-2 text-sm text-ink-200"><input type="checkbox" checked={f.apply_plan} onChange={(e) => setF({ ...f, apply_plan: e.target.checked })} />Trocar o plano da aluna para este</label>
      )}
      <div className="grid grid-cols-2 gap-3">
        <Field label="Valor (R$)" hint={plan ? "Vazio = valor do contrato" : undefined}><NumberInput value={f.amount} min={0} step={0.01} onChange={(n) => setF({ ...f, amount: n })} /></Field>
        {(sellers.data?.length ?? 0) > 0 && <Field label="Vendedor(a)"><select className="field" value={f.seller_id} onChange={(e) => setF({ ...f, seller_id: e.target.value })}><option value="">Eu mesmo(a)</option>{sellers.data?.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}</select></Field>}
      </div>
      <Field label="Descrição"><input className="field" value={f.description} onChange={(e) => setF({ ...f, description: e.target.value })} /></Field>
    </Modal>
  );
}

function GoalModal({ month, target, onClose, onDone }: { month: string; target: { id: string | null; name: string; row?: RankingRow }; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [sales, setSales] = useState(target.row?.goal_sales || 0);
  const [revenue, setRevenue] = useState(target.row?.goal_revenue || 0);
  return (
    <Modal title={`Meta ${target.name} — ${monthLabel(month)}`} onClose={onClose} width="max-w-md"
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-brand" disabled={busy}
        onClick={async () => { const r = await run(() => api.setGoal({ user_id: target.id, month, target_sales: sales, target_revenue: revenue }), "Meta salva"); if (r) { onDone(); onClose(); } }}>Salvar</button></>}>
      <p className="text-xs text-ink-300">Deixe os dois campos zerados para remover a meta.</p>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Meta de vendas (qtd)"><NumberInput value={sales} min={0} onChange={setSales} /></Field>
        <Field label="Meta de faturamento (R$)"><NumberInput value={revenue} min={0} step={50} onChange={setRevenue} /></Field>
      </div>
    </Modal>
  );
}
