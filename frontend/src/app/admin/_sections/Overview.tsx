"use client";

import React from "react";
import { AlertTriangle, CalendarClock, CheckCircle2, Users, UserPlus, Wallet, Wrench, Dumbbell, Handshake } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, Empty, Notice, Progress, SectionHeader, StatCard, Tag } from "@/components/ui";
import { useAsync } from "@/lib/hooks";
import { brl, dateBR, timeBR, INVOICE_STATUS, whatsappLink } from "@/lib/format";
import { ErrorBox, Loading, PageHead } from "./shared";
import type { Permission } from "@/types";
import { BRAND } from "@/lib/brand";

export default function Overview({ can, go }: { can: (...p: Permission[]) => boolean; go: (tab: string) => void }) {
  const { data: s, loading, error, reload } = useAsync(() => api.getDashboardStats(), []);
  if (loading && !s) return <Loading rows={4} />;
  if (error || !s) return <ErrorBox message={error || "Sem dados"} onRetry={reload} />;

  const occupancy = (cap: number, booked: number) => (cap ? Math.round((booked / cap) * 100) : 0);
  const receivedPct = s.expected_revenue ? Math.round((s.monthly_revenue / s.expected_revenue) * 100) : 0;

  const alerts: { tone: "bad" | "warn"; text: string; tab?: string }[] = [];
  if (s.uncovered_sessions > 0) alerts.push({ tone: "bad", text: `${s.uncovered_sessions} aula(s) dos próximos 7 dias estão sem professor(a)`, tab: "agenda" });
  if (s.show_finance && s.delinquent_students > 0) alerts.push({ tone: "bad", text: `${s.delinquent_students} aluna(s) com mensalidade vencida (${brl(s.overdue_total)})`, tab: "financeiro" });
  if (s.equipment_issues > 0) alerts.push({ tone: "warn", text: `${s.equipment_issues} equipamento(s) em atenção ou manutenção`, tab: "config" });
  if (s.new_leads > 0) alerts.push({ tone: "warn", text: `${s.new_leads} lead(s) novo(s) esperando contato`, tab: "leads" });
  if (s.without_workout_count > 0) alerts.push({ tone: "warn", text: `${s.without_workout_count} aluna(s) ativa(s) sem ficha de treino`, tab: "alunos" });

  return (
    <div className="space-y-6">
      <PageHead title="Visão geral" subtitle="O que está acontecendo na academia hoje" />

      {alerts.length > 0 && (
        <div className="grid gap-2 md:grid-cols-2">
          {alerts.map((a) => (
            <button key={a.text} onClick={() => a.tab && go(a.tab)} className="text-left">
              <Notice tone={a.tone === "bad" ? "bad" : "ok"} icon={<AlertTriangle className="w-4 h-4 shrink-0" />}>{a.text}</Notice>
            </button>
          ))}
        </div>
      )}

      <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
        <StatCard icon={<Users className="w-4 h-4" />} label="Alunas ativas" value={s.active_students} hint={`${s.total_students} matrículas no total`}
          progress={s.total_students ? (s.active_students / s.total_students) * 100 : 0} />
        <StatCard icon={<CheckCircle2 className="w-4 h-4" />} label="Check-ins hoje" value={s.today_checkins}
          hint={`${occupancy(s.active_students, s.today_checkins)}% dos ativos`} progress={occupancy(s.active_students, s.today_checkins)} />
        {s.show_finance ? (
          <StatCard variant="brand" icon={<Wallet className="w-4 h-4" />} label="Recebido no mês" value={brl(s.monthly_revenue)}
            hint={`de ${brl(s.expected_revenue)} previstos`} progress={receivedPct} />
        ) : (
          <StatCard icon={<Handshake className="w-4 h-4" />} label="Vendas no mês" value={s.sales_month} hint="matrículas e renovações" />
        )}
        <StatCard icon={<UserPlus className="w-4 h-4" />} label="Leads (30 dias)" value={s.leads_last30} hint={`${s.enrolled_last30} viraram matrícula`}
          progress={s.leads_last30 ? (s.enrolled_last30 / s.leads_last30) * 100 : 0} />
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <section className="card p-5 lg:col-span-2">
          <SectionHeader title="Aulas de hoje" action={can("agenda.manage") ? <button className="hover:text-white" onClick={() => go("agenda")}>Abrir agenda →</button> : undefined} />
          {s.today_sessions.length === 0 ? (
            <Empty icon={<CalendarClock className="w-6 h-6" />} title="Nenhuma aula hoje">Cadastre turmas na Agenda para elas aparecerem aqui.</Empty>
          ) : (
            <ul className="space-y-2">
              {s.today_sessions.map((t) => (
                <li key={t.id} className="card-soft p-3.5 flex items-center gap-3">
                  <span className="w-1.5 self-stretch rounded-full" style={{ background: t.color || BRAND }} />
                  <div className="min-w-[52px] text-sm font-semibold text-white">{timeBR(t.starts_at)}</div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-white truncate">{t.title}</div>
                    <div className="text-xs text-ink-300 truncate">{t.teacher || "Sem professor(a)"}</div>
                  </div>
                  {t.status === "CANCELADA" ? <Tag tone="mute">Cancelada</Tag> : (
                    <div className="w-20 sm:w-28 shrink-0">
                      <div className="text-[11px] text-ink-300 text-right mb-1">{t.booked}/{t.capacity} vagas</div>
                      <Progress value={occupancy(t.capacity, t.booked)} height={6} />
                    </div>
                  )}
                  {!t.teacher && t.status !== "CANCELADA" && <Tag tone="bad">Sem prof.</Tag>}
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="card p-5">
          <SectionHeader title="Últimas entradas" />
          {s.recent_checkins.length === 0 ? (
            <Empty icon={<Dumbbell className="w-6 h-6" />} title="Ninguém entrou ainda" />
          ) : (
            <ul className="space-y-2.5">
              {s.recent_checkins.map((c) => (
                <li key={c.id} className="flex items-center gap-3">
                  <Avatar name={c.student?.user?.name} size={36} tone="dark" />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm text-white truncate">{c.student?.user?.name || "Aluna"}</div>
                    <div className="text-[11px] text-ink-300">{dateBR(c.created_at)} · {timeBR(c.created_at)}</div>
                  </div>
                  <Tag tone="mute">{c.method === "MANUAL" ? "Manual" : "QR"}</Tag>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>

      {s.show_finance && s.pending_invoices.length > 0 && (
        <section className="card p-5">
          <SectionHeader title="Cobranças para acompanhar" action={<button className="hover:text-white" onClick={() => go("financeiro")}>Ver financeiro →</button>} />
          <ul className="divide-y divide-white/5">
            {s.pending_invoices.map((inv) => {
              const st = INVOICE_STATUS[inv.status];
              const wa = whatsappLink(inv.student?.user?.phone, `Oi ${inv.student?.user?.name?.split(" ")[0] || ""}! Passando para lembrar da mensalidade (${brl(inv.amount - inv.discount)}) com vencimento em ${dateBR(inv.due_date)}. Qualquer dúvida estamos à disposição 💜`);
              return (
                <li key={inv.id} className="py-3 flex items-center gap-3">
                  <Avatar name={inv.student?.user?.name} size={36} tone="dark" />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm text-white truncate">{inv.student?.user?.name}</div>
                    <div className="text-[11px] text-ink-300">Vence {dateBR(inv.due_date)} · {brl(inv.amount - inv.discount)}</div>
                  </div>
                  <Tag tone={st.tone}>{st.label}</Tag>
                  {wa && <a href={wa} target="_blank" rel="noreferrer" className="btn-dark px-3 py-2 text-xs">WhatsApp</a>}
                </li>
              );
            })}
          </ul>
        </section>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <div className="card-soft p-4 flex items-center gap-3"><Users className="w-5 h-5 text-brand" /><div><div className="text-lg font-semibold text-white">{s.staff_active}</div><div className="text-xs text-ink-300">pessoas na equipe</div></div></div>
        <div className="card-soft p-4 flex items-center gap-3"><Handshake className="w-5 h-5 text-brand" /><div><div className="text-lg font-semibold text-white">{s.sales_month}</div><div className="text-xs text-ink-300">vendas no mês</div></div></div>
        <div className="card-soft p-4 flex items-center gap-3"><Wrench className="w-5 h-5 text-brand" /><div><div className="text-lg font-semibold text-white">{s.equipment_issues}</div><div className="text-xs text-ink-300">equipamentos com problema</div></div></div>
      </div>
    </div>
  );
}
