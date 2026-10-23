"use client";

import React, { useState } from "react";
import { Download } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, Bars, Empty, LineChart, PillTabs, Progress, SectionHeader, StatCard } from "@/components/ui";
import { useAsync } from "@/lib/hooks";
import { brl, dateBR, downloadCSV, monthLabel, num, whatsappLink, WEEKDAYS_SHORT } from "@/lib/format";
import { ErrorBox, Loading, PageHead } from "./shared";
import { TrendingDown, Users, Wallet, Receipt } from "lucide-react";
import { BRAND_RGB } from "@/lib/brand";

export default function Reports() {
  const [tab, setTab] = useState<"dono" | "frequencia">("dono");
  return (
    <div className="space-y-5">
      <PageHead title="Relatórios" subtitle="Como a academia está indo" actions={<PillTabs value={tab} onChange={setTab} tabs={[{ id: "dono", label: "Visão do dono" }, { id: "frequencia", label: "Frequência" }]} />} />
      {tab === "dono" ? <Owner /> : <Frequency />}
    </div>
  );
}

function Owner() {
  const [months, setMonths] = useState(6);
  const { data, loading, error, reload } = useAsync(() => api.getOverview(months), [months]);
  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading rows={4} />;
  if (!data) return null;
  const last = data.months[data.months.length - 1];
  const hours = Array.from(new Set(data.by_slot.map((s) => s.hour))).sort((a, b) => a - b);
  const slot = (d: number, h: number) => data.by_slot.find((s) => s.weekday === d && s.hour === h);
  const maxMix = Math.max(...data.plan_mix.map((p) => p.students), 1);

  return (
    <div className="space-y-6">
      <div className="flex justify-end"><PillTabs value={String(months)} onChange={(v) => setMonths(Number(v))} tabs={[{ id: "6", label: "6 meses" }, { id: "12", label: "12 meses" }]} /></div>
      <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <StatCard icon={<Users className="w-4 h-4" />} label="Alunas ativas" value={data.active} hint={`${last.new_students} novos · ${last.canceled} cancelados no mês`} />
        <StatCard icon={<TrendingDown className="w-4 h-4" />} label="Cancelamento (mês)" value={`${num(data.churn_rate, 1)}%`} hint="cancelados ÷ base do mês" accent="#FF7A5C" />
        <StatCard variant="brand" icon={<Wallet className="w-4 h-4" />} label="Receita do mês" value={brl(last.revenue)} hint={`previsto ${brl(last.expected)}`} />
        <StatCard icon={<Receipt className="w-4 h-4" />} label="Ticket médio" value={brl(data.ticket)} hint={`${data.payers} aluna(s) pagaram`} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="card p-5"><SectionHeader title="Receita por mês" />
          <Bars values={data.months.map((m) => m.revenue)} labels={data.months.map((m) => monthLabel(m.month, true).slice(0, 3))} height={110} highlight={data.months.length - 1} /></section>
        <section className="card p-5"><SectionHeader title="Entradas na academia" />
          <LineChart values={data.months.map((m) => m.checkins)} labels={data.months.map((m) => monthLabel(m.month, true).slice(0, 3))} /></section>
      </div>

      <section className="card p-5 overflow-x-auto"><SectionHeader title="Mês a mês" action={<button className="hover:text-white inline-flex items-center gap-1" onClick={() => downloadCSV("evolucao-mensal.csv", ["Mês", "Novos alunas", "Cancelados", "Receita", "Previsto", "Vendas", "Valor vendido", "Leads", "Entradas"], data.months.map((m) => [monthLabel(m.month), m.new_students, m.canceled, m.revenue, m.expected, m.sales, m.sales_amount, m.leads, m.checkins]))}><Download className="w-3.5 h-3.5" />CSV</button>} />
        <table className="w-full text-sm">
          <thead className="text-xs text-ink-300"><tr>{["Mês", "Novos", "Cancel.", "Receita", "Vendas", "Leads", "Entradas"].map((h, i) => <th key={h} className={`py-2 font-medium ${i ? "text-right" : "text-left"}`}>{h}</th>)}</tr></thead>
          <tbody>{[...data.months].reverse().map((m) => (
            <tr key={m.month} className="border-t border-white/5"><td className="py-2.5 text-white">{monthLabel(m.month)}</td><td className="text-right">{m.new_students}</td><td className="text-right">{m.canceled}</td><td className="text-right text-white">{brl(m.revenue)}</td><td className="text-right">{m.sales}</td><td className="text-right">{m.leads}</td><td className="text-right">{m.checkins}</td></tr>))}
          </tbody></table></section>

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="card p-5"><SectionHeader title="Ocupação por professor(a) — 30 dias" />
          {data.by_teacher.length === 0 ? <Empty title="Sem aulas realizadas" /> : <ul className="space-y-3">{data.by_teacher.map((t) => (
            <li key={t.name}><div className="flex justify-between text-sm mb-1"><span className="text-white">{t.name}</span><span className="text-ink-300">{Math.round(t.occupancy)}% · {t.sessions} aulas</span></div><Progress value={t.occupancy} /></li>))}</ul>}</section>
        <section className="card p-5"><SectionHeader title="Alunas por plano" />
          {data.plan_mix.length === 0 ? <Empty title="Sem alunas" /> : <ul className="space-y-3">{data.plan_mix.map((p) => (
            <li key={p.plan}><div className="flex justify-between text-sm mb-1"><span className="text-white">{p.plan}</span><span className="text-ink-300">{p.students} aluna(s) · {brl(p.monthly)}/mês</span></div><Progress value={(p.students / maxMix) * 100} /></li>))}</ul>}</section>
      </div>

      <section className="card p-5 overflow-x-auto"><SectionHeader title="Lotação por dia e horário — 30 dias" />
        {data.by_slot.length === 0 ? <Empty title="Sem aulas realizadas" /> : (
          <table className="text-xs border-separate border-spacing-1"><thead><tr><th />{WEEKDAYS_SHORT.map((d) => <th key={d} className="text-ink-300 font-medium px-2">{d}</th>)}</tr></thead>
            <tbody>{hours.map((h) => (<tr key={h}><td className="text-ink-300 pr-2 text-right">{String(h).padStart(2, "0")}h</td>{WEEKDAYS_SHORT.map((_, d) => { const s = slot(d, h); return (
              <td key={d} className="w-12 h-9 text-center rounded-lg" title={s ? `${s.sessions} aulas · ${s.booked}/${s.capacity} vagas` : ""} style={{ background: s ? `rgba(${BRAND_RGB},${0.08 + Math.min(1, s.occupancy / 100) * 0.7})` : "transparent", color: s && s.occupancy > 55 ? "#000" : undefined }}>{s ? `${Math.round(s.occupancy)}%` : ""}</td>); })}</tr>))}</tbody></table>)}</section>
    </div>
  );
}

function Frequency() {
  const [days, setDays] = useState(30);
  const { data, loading, error, reload } = useAsync(() => api.getFrequency(days), [days]);
  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading rows={4} />;
  if (!data) return null;
  const maxHeat = Math.max(...data.heatmap.flat(), 1);
  const hours = Array.from({ length: 17 }, (_, i) => i + 6);

  return (
    <div className="space-y-6">
      <div className="flex justify-end"><PillTabs value={String(days)} onChange={(v) => setDays(Number(v))} tabs={[{ id: "14", label: "14 dias" }, { id: "30", label: "30 dias" }, { id: "90", label: "90 dias" }]} /></div>
      <div className="grid gap-6 lg:grid-cols-2">
        <section className="card p-5"><SectionHeader title={`Entradas por dia (${data.total} no período)`} /><LineChart values={data.daily.map((d) => d.count)} labels={data.daily.filter((_, i) => i % Math.ceil(data.daily.length / 6) === 0).map((d) => d.date.slice(8)).concat([])} /></section>
        <section className="card p-5"><SectionHeader title="Presença por modalidade" />
          {data.attendance.length === 0 ? <Empty title="Sem aulas no período" /> : <ul className="space-y-3">{data.attendance.map((a) => (
            <li key={a.modality}><div className="flex justify-between text-sm mb-1"><span className="text-white flex items-center gap-2"><span className="w-2.5 h-2.5 rounded-full" style={{ background: a.color }} />{a.modality}</span><span className="text-ink-300">{a.attended} presenças · {a.no_show} faltas · {a.sessions} aulas</span></div><Progress value={a.capacity ? (a.attended / a.capacity) * 100 : 0} color={a.color} /></li>))}</ul>}</section>
      </div>

      <section className="card p-5 overflow-x-auto"><SectionHeader title="Horários de pico" />
        <table className="text-xs border-separate border-spacing-1"><thead><tr><th />{WEEKDAYS_SHORT.map((d) => <th key={d} className="text-ink-300 font-medium px-2">{d}</th>)}</tr></thead>
          <tbody>{hours.map((h) => (<tr key={h}><td className="text-ink-300 pr-2 text-right">{String(h).padStart(2, "0")}h</td>{WEEKDAYS_SHORT.map((_, d) => { const v = data.heatmap[d]?.[h] || 0; return (
            <td key={d} className="w-12 h-7 text-center rounded-lg" style={{ background: v ? `rgba(${BRAND_RGB},${0.1 + (v / maxHeat) * 0.75})` : "rgba(255,255,255,0.03)", color: v / maxHeat > 0.55 ? "#000" : undefined }}>{v || ""}</td>); })}</tr>))}</tbody></table></section>

      <section className="card p-5"><SectionHeader title={`Alunas sumidas (${data.absent.length})`} action={<span>ativas sem entrar há 7+ dias</span>} />
        {data.absent.length === 0 ? <Empty title="Ninguém sumido 🎉" /> : (
          <ul className="divide-y divide-white/5">{data.absent.map((a) => { const wa = whatsappLink(a.phone, `Oi ${a.name.split(" ")[0]}! Sentimos sua falta por aqui 💜 Está tudo bem? Quer que a gente ajude a encaixar um horário?`); return (
            <li key={a.student_id} className="py-3 flex items-center gap-3"><Avatar name={a.name} size={38} tone="dark" />
              <div className="flex-1 min-w-0"><div className="text-sm text-white truncate">{a.name}</div><div className="text-[11px] text-ink-300">{a.plan_name || "Sem plano"} · {a.last_checkin_at ? `última entrada ${dateBR(a.last_checkin_at)}` : "nunca entrou no último ano"}</div></div>
              <span className="text-sm font-semibold text-[#FF7A5C]">{a.days_absent < 0 ? "—" : `${a.days_absent} dias`}</span>
              {wa && <a className="btn-dark px-3 py-2 text-xs" href={wa} target="_blank" rel="noreferrer">Chamar</a>}</li>); })}</ul>)}</section>
    </div>
  );
}
