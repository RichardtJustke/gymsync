"use client";

import React, { useMemo, useState } from "react";
import { CalendarDays, Users } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, Empty, Modal, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { isoDate, timeBR, WEEKDAYS } from "@/lib/format";
import type { Booking, ClassSession } from "@/types";

/** Aulas da professora nos próximos dias + chamada */
export default function Classes() {
  const [days, setDays] = useState(7);
  const { data, loading, error, reload } = useAsync(() => api.listSessions({ from: isoDate(), days, teacher_id: "me" }), [days]);
  const [open, setOpen] = useState<ClassSession | null>(null);

  const grouped = useMemo(() => {
    const g: Record<string, ClassSession[]> = {};
    (data || []).forEach((s) => (g[isoDate(new Date(s.starts_at))] ||= []).push(s));
    return Object.entries(g);
  }, [data]);

  return (
    <div className="max-w-3xl w-full mx-auto px-4 py-8 space-y-6">
      <div className="flex items-end justify-between gap-3">
        <div><div className="eyebrow mb-1">Minha agenda</div><h1 className="text-3xl font-semibold text-white tracking-tight">Aulas</h1></div>
        <select className="field w-auto" value={days} onChange={(e) => setDays(Number(e.target.value))} aria-label="Período">
          <option value={1}>Hoje</option><option value={7}>7 dias</option><option value={14}>14 dias</option>
        </select>
      </div>

      {error ? <div className="card p-6"><Empty title="Não foi possível carregar">{error}<div className="mt-3"><button className="btn-dark" onClick={reload}>Tentar de novo</button></div></Empty></div>
        : loading && !data ? <div className="space-y-3"><div className="h-20 rounded-3xl bg-ink-800 animate-pulse" /><div className="h-20 rounded-3xl bg-ink-800 animate-pulse" /></div>
        : grouped.length === 0 ? <div className="card"><Empty icon={<CalendarDays className="w-6 h-6" />} title="Você não tem aulas neste período" /></div>
        : grouped.map(([day, list]) => (
          <section key={day} className="space-y-2">
            <div className="eyebrow">{WEEKDAYS[new Date(day + "T12:00:00").getDay()]}, {new Date(day + "T12:00:00").toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" })}{day === isoDate() ? " · hoje" : ""}</div>
            {list.map((s) => {
              const past = new Date(s.ends_at) < new Date();
              return (
                <button key={s.id} onClick={() => setOpen(s)} className="card w-full p-4 flex items-center gap-3 text-left hover:bg-ink-750 transition-colors">
                  <span className="w-1.5 self-stretch rounded-full" style={{ background: s.modality?.color || "#D7FF3A" }} />
                  <div className="w-24 text-sm font-semibold text-white">{timeBR(s.starts_at)}–{timeBR(s.ends_at)}</div>
                  <div className="flex-1 min-w-0"><div className="text-sm text-white truncate">{s.title || s.modality?.name}</div><div className="text-xs text-ink-300">{s.room || "Sem sala"}</div></div>
                  <span className="text-xs text-ink-300 flex items-center gap-1"><Users className="w-3.5 h-3.5" />{s.booked ?? 0}/{s.capacity}</span>
                  {s.status === "CANCELADA" ? <Tag tone="mute">Cancelada</Tag> : <Tag tone={past ? "mute" : "ok"}>{past ? "Chamada" : "Lista"}</Tag>}
                </button>
              );
            })}
          </section>
        ))}
      {open && <Attendance session={open} onClose={() => { setOpen(null); reload(); }} />}
    </div>
  );
}

function Attendance({ session, onClose }: { session: ClassSession; onClose: () => void }) {
  const { data, loading, error, reload } = useAsync(() => api.sessionAttendees(session.id), [session.id]);
  const { busy, run } = useAction();
  const mark = async (b: Booking, status: "PRESENTE" | "FALTA") => {
    const r = await run(() => api.markAttendance(b.id, status));
    if (r !== undefined) reload();
  };
  return (
    <Modal title={`${session.title || session.modality?.name} · ${timeBR(session.starts_at)}`} onClose={onClose}>
      {error ? <p className="text-sm text-[#FF7A5C]">{error}</p> : loading && !data ? <div className="h-16 rounded-2xl bg-ink-750 animate-pulse" /> : data && data.length === 0 ? <Empty title="Ninguém reservou ainda" /> : (
        <ul className="space-y-2">
          {data?.map((b) => {
            const name = b.student?.user?.name || b.lead?.name || "—";
            return (
              <li key={b.id} className="card-soft p-3 flex items-center gap-3">
                <Avatar name={name} size={36} tone="dark" />
                <div className="flex-1 min-w-0"><div className="text-sm text-white truncate">{name}</div><div className="text-[11px] text-ink-300">{b.lead ? "Aula experimental" : b.status === "ESPERA" ? "Lista de espera" : ""}</div></div>
                {b.status === "ESPERA" ? <Tag tone="warn">espera</Tag> : (
                  <div className="flex gap-1.5">
                    <button disabled={busy} onClick={() => mark(b, "PRESENTE")} className={b.status === "PRESENTE" ? "pill-on px-3 py-1.5 text-xs" : "pill-off px-3 py-1.5 text-xs"}>Presente</button>
                    <button disabled={busy} onClick={() => mark(b, "FALTA")} className={b.status === "FALTA" ? "pill bg-[#FF5A36] text-white px-3 py-1.5 text-xs" : "pill-off px-3 py-1.5 text-xs"}>Faltou</button>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </Modal>
  );
}
