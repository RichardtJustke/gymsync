"use client";

import React, { useState } from "react";
import { Download, FilePlus2, RefreshCw, Search } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, Bars, ConfirmDialog, Empty, Field, Modal, MonthPicker, PillTabs, SectionHeader, StatCard, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { useToast } from "@/context/ToastContext";
import { brl, dateBR, downloadCSV, INVOICE_STATUS, isoDate, methodLabel, monthKey, monthLabel, whatsappLink } from "@/lib/format";
import { ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import PayModal from "./PayModal";
import { Wallet, AlertCircle, Clock, Percent } from "lucide-react";
import type { Invoice, InvoiceStatus, Permission } from "@/types";

type Filter = "" | InvoiceStatus;
const FILTERS: { id: Filter; label: string }[] = [
  { id: "", label: "Todas" },
  { id: "PENDENTE", label: "Em aberto" },
  { id: "VENCIDO", label: "Vencidas" },
  { id: "PAGO", label: "Pagas" },
  { id: "CANCELADO", label: "Canceladas" },
];

export default function Finance({ can }: { can: (...p: Permission[]) => boolean }) {
  const [month, setMonth] = useState(monthKey());
  const [filter, setFilter] = useState<Filter>("");
  const [q, setQ] = useState("");
  const summary = useAsync(() => api.financeSummary(month), [month]);
  const list = useAsync(() => api.listInvoices({ month, status: filter, q }), [month, filter, q]);
  const { busy, run } = useAction();
  const toast = useToast();
  const [pay, setPay] = useState<Invoice | null>(null);
  const [cancel, setCancel] = useState<Invoice | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [boleto, setBoleto] = useState<string | null>(null);

  const refresh = () => { summary.reload(); list.reload(); };
  const s = summary.data;
  const manage = can("finance.manage");

  const generate = async () => {
    const r = await run(() => api.generateInvoices(month));
    if (r) {
      refresh();
      toast.success(r.created > 0 ? `${r.created} mensalidade(s) gerada(s) para ${monthLabel(month)}.` : "Todas as mensalidades deste mês já estavam geradas.");
    }
  };
  const openBoleto = async (inv: Invoice) => {
    const html = await run(() => api.getBoletoHtml(inv.id));
    if (html) setBoleto(html);
  };
  const exportCsv = () =>
    downloadCSV(`financeiro-${month}.csv`, ["Aluno", "Descrição", "Vencimento", "Valor", "Desconto", "Situação", "Pago em", "Forma"],
      (list.data || []).map((i) => [i.student?.user?.name || "", i.description, dateBR(i.due_date), i.amount, i.discount, INVOICE_STATUS[i.status].label, dateBR(i.paid_at), methodLabel(i.payment_method)]));

  return (
    <div className="space-y-6">
      <PageHead
        title="Financeiro"
        subtitle={s?.gateway === "mock" ? "Cobrança em modo de teste — o gateway de pagamento ainda não foi conectado" : undefined}
        actions={
          <>
            <MonthPicker value={month} onChange={setMonth} label={monthLabel} />
            {manage && <button className="btn-dark" disabled={busy} onClick={generate}><RefreshCw className="w-4 h-4" />Gerar mensalidades</button>}
            {manage && <button className="btn-lime" onClick={() => setShowNew(true)}><FilePlus2 className="w-4 h-4" />Cobrança avulsa</button>}
          </>
        }
      />

      {summary.error ? <ErrorBox message={summary.error} onRetry={summary.reload} /> : !s ? <Loading rows={2} /> : (
        <>
          <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
            <StatCard variant="lime" icon={<Wallet className="w-4 h-4" />} label="Recebido" value={brl(s.received)} hint={`${s.received_count} pagamento(s)`} progress={s.expected ? (s.received / s.expected) * 100 : 0} />
            <StatCard icon={<Clock className="w-4 h-4" />} label="A receber no mês" value={brl(s.pending)} hint={`${s.pending_count} cobrança(s) em aberto`} />
            <StatCard icon={<AlertCircle className="w-4 h-4" />} label="Total vencido" value={brl(s.overdue_total)} hint={`${s.delinquent_students} aluno(s) · todos os meses`} accent="#FF7A5C" />
            <StatCard icon={<Percent className="w-4 h-4" />} label="Descontos dados" value={brl(s.discounts)} hint="no mês" />
          </div>
          <div className="grid gap-6 lg:grid-cols-3">
            <section className="card p-5 lg:col-span-2">
              <SectionHeader title="Recebimentos por dia" />
              {s.received === 0 ? <Empty title="Nenhum recebimento neste mês" /> : <Bars values={s.daily_received.map((d) => d.value)} labels={s.daily_received.map((d) => (d.day % 5 === 0 || d.day === 1 ? String(d.day) : ""))} height={96} />}
            </section>
            <section className="card p-5">
              <SectionHeader title="Por forma de pagamento" />
              {Object.keys(s.by_method).length === 0 ? <Empty title="Sem recebimentos" /> : (
                <ul className="space-y-2.5 text-sm">
                  {Object.entries(s.by_method).sort((a, b) => b[1] - a[1]).map(([m, v]) => (
                    <li key={m} className="flex items-center justify-between"><span className="text-ink-200">{methodLabel(m)}</span><span className="text-white font-medium">{brl(v)}</span></li>
                  ))}
                </ul>
              )}
            </section>
          </div>
        </>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[220px]">
          <Search className="w-4 h-4 absolute left-4 top-1/2 -translate-y-1/2 text-ink-400" />
          <input className="field pl-11" placeholder="Buscar aluno ou descrição" value={q} onChange={(e) => setQ(e.target.value)} />
        </div>
        <PillTabs tabs={FILTERS} value={filter} onChange={setFilter} />
        <button className="btn-dark" onClick={exportCsv} disabled={!list.data?.length}><Download className="w-4 h-4" />CSV</button>
      </div>

      {list.error ? <ErrorBox message={list.error} onRetry={list.reload} /> : !list.data ? <Loading /> : (
        <div className="card overflow-hidden">
          {list.data.length === 0 ? <Empty title="Nenhuma cobrança" >Use “Gerar mensalidades” para criar as cobranças do mês.</Empty> : (
            <ul className="divide-y divide-white/5">
              {list.data.map((inv) => {
                const st = INVOICE_STATUS[inv.status];
                const isOpen = inv.status === "PENDENTE" || inv.status === "VENCIDO";
                const pixMsg = inv.pix_copy_paste ? `\n\n🔑 Código PIX Copia e Cola:\n${inv.pix_copy_paste}` : "";
                const wa = whatsappLink(inv.student?.user?.phone, `Oi ${inv.student?.user?.name?.split(" ")[0] || ""}! Lembrete da sua ${inv.description} no valor de ${brl(inv.amount - inv.discount)}, com vencimento em ${dateBR(inv.due_date)}.${pixMsg}\n\nQualquer dúvida, conte com a equipe Unika! 💜`);
                return (
                  <li key={inv.id} className="p-4 flex flex-wrap items-center gap-3">
                    <Avatar name={inv.student?.user?.name} size={40} tone="dark" />
                    <div className="flex-1 min-w-[180px]">
                      <div className="text-sm font-medium text-white">{inv.student?.user?.name}</div>
                      <div className="text-xs text-ink-300">{inv.description} · vence {dateBR(inv.due_date)}{inv.paid_at ? ` · pago ${dateBR(inv.paid_at)} (${methodLabel(inv.payment_method)})` : ""}{inv.cancel_reason ? ` · ${inv.cancel_reason}` : ""}</div>
                    </div>
                    <div className="text-sm font-semibold text-white">{brl(inv.status === "PAGO" ? inv.paid_amount : inv.amount - inv.discount)}</div>
                    <Tag tone={st.tone}>{st.label}</Tag>
                    <div className="flex gap-2">
                      {isOpen && wa && <a className="btn-dark px-3 py-2 text-xs hover:text-emerald-400" href={wa} target="_blank" rel="noreferrer">Cobrar WhatsApp</a>}
                      {isOpen && <button className="btn-dark px-3 py-2 text-xs" onClick={() => openBoleto(inv)}>Boleto</button>}
                      {isOpen && manage && <button className="btn-lime px-3 py-2 text-xs" onClick={() => setPay(inv)}>Receber</button>}
                      {isOpen && manage && <button className="btn-ghost px-3 py-2 text-xs" onClick={() => setCancel(inv)}>Cancelar</button>}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}

      {pay && <PayModal invoice={pay} onClose={() => setPay(null)} onDone={refresh} />}
      {cancel && (
        <ConfirmDialog title="Cancelar cobrança" danger confirmLabel="Cancelar cobrança" askReason="Motivo (fica no histórico)" reasonRequired busy={busy}
          message={`${cancel.student?.user?.name} — ${cancel.description} (${brl(cancel.amount)}). Cobrança cancelada não conta como receita nem como dívida.`}
          onClose={() => setCancel(null)}
          onConfirm={async (reason) => { const r = await run(() => api.cancelInvoice(cancel.id, reason), "Cobrança cancelada"); if (r) { setCancel(null); refresh(); } }} />
      )}
      {showNew && <NewInvoiceModal onClose={() => setShowNew(false)} onDone={refresh} />}
      {boleto && (
        <Modal title="Boleto (simulado)" onClose={() => setBoleto(null)} width="max-w-3xl">
          {/* sandbox: sem scripts da página e sem acesso ao app */}
          <iframe title="Boleto" srcDoc={boleto} sandbox="allow-modals" className="w-full h-[60vh] rounded-2xl bg-white" />
        </Modal>
      )}
    </div>
  );
}

function NewInvoiceModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [search, setSearch] = useState("");
  const found = useAsync(() => (search.trim().length >= 2 ? api.listStudents(search, "", 1, 6) : Promise.resolve(null)), [search]);
  const [student, setStudent] = useState<{ id: string; name: string } | null>(null);
  const [f, setF] = useState({ amount: 0, due_date: isoDate(), description: "" });

  return (
    <Modal title="Cobrança avulsa" onClose={onClose} width="max-w-md"
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-lime" disabled={busy || !student || f.amount <= 0}
        onClick={async () => { const r = await run(() => api.createInvoice({ student_id: student!.id, ...f }), "Cobrança lançada"); if (r) { onDone(); onClose(); } }}>Lançar</button></>}>
      {student ? (
        <div className="card-soft p-3 flex items-center justify-between text-sm"><span className="text-white">{student.name}</span><button className="btn-ghost px-3 py-1.5" onClick={() => setStudent(null)}>Trocar</button></div>
      ) : (
        <Field label="Aluno">
          <input className="field" placeholder="Digite o nome" value={search} onChange={(e) => setSearch(e.target.value)} autoFocus />
          {found.data && (
            <ul className="mt-2 space-y-1">
              {found.data.data.map((s) => <li key={s.id}><button className="w-full text-left p-2.5 rounded-xl hover:bg-white/5 text-sm text-white" onClick={() => setStudent({ id: s.id, name: s.user.name })}>{s.user.name}</button></li>)}
            </ul>
          )}
        </Field>
      )}
      <div className="grid grid-cols-2 gap-3">
        <Field label="Valor (R$)"><NumberInput value={f.amount} min={0} step={0.01} onChange={(n) => setF({ ...f, amount: n })} /></Field>
        <Field label="Vencimento"><input type="date" className="field" value={f.due_date} onChange={(e) => setF({ ...f, due_date: e.target.value })} /></Field>
      </div>
      <Field label="Descrição"><input className="field" placeholder="Ex.: Avaliação física, taxa de matrícula" value={f.description} onChange={(e) => setF({ ...f, description: e.target.value })} /></Field>
    </Modal>
  );
}
