"use client";

import React, { useMemo, useState } from "react";
import { AlertTriangle, CalendarPlus, ChevronLeft, ChevronRight, Repeat, Users } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, ConfirmDialog, Empty, Field, Modal, PillTabs, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { isoDate, timeBR, WEEKDAYS } from "@/lib/format";
import { ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import type { Booking, ClassSchedule, ClassSession, Permission } from "@/types";
import { BRAND } from "@/lib/brand";

function mondayOf(d: Date) {
  const x = new Date(d.getFullYear(), d.getMonth(), d.getDate());
  x.setDate(x.getDate() - ((x.getDay() + 6) % 7));
  return x;
}

export default function Agenda({ can }: { can: (...p: Permission[]) => boolean }) {
  const manage = can("agenda.manage");
  const [view, setView] = useState<"semana" | "turmas">("semana");
  const [monday, setMonday] = useState(() => mondayOf(new Date()));
  const [teacherFilter, setTeacherFilter] = useState("");
  const from = isoDate(monday);
  const sessions = useAsync(() => api.listSessions({ from, days: 7, teacher_id: teacherFilter }), [from, teacherFilter]);
  const teachers = useAsync(() => api.listTeachers(), []);
  const [attend, setAttend] = useState<ClassSession | null>(null);
  const [swap, setSwap] = useState<ClassSession | null>(null);
  const [cancel, setCancel] = useState<ClassSession | null>(null);
  const [showNew, setShowNew] = useState(false);
  const { busy, run } = useAction();

  const days = useMemo(() => Array.from({ length: 7 }, (_, i) => { const d = new Date(monday); d.setDate(d.getDate() + i); return d; }), [monday]);
  const byDay = useMemo(() => {
    const m: Record<string, ClassSession[]> = {};
    (sessions.data || []).forEach((s) => { const k = isoDate(new Date(s.starts_at)); (m[k] ||= []).push(s); });
    return m;
  }, [sessions.data]);
  const uncovered = (sessions.data || []).filter((s) => !s.teacher_id && s.status === "AGENDADA" && new Date(s.starts_at) > new Date()).length;
  const shift = (w: number) => { const d = new Date(monday); d.setDate(d.getDate() + 7 * w); setMonday(d); };
  const rangeLabel = `${monday.toLocaleDateString("pt-BR", { day: "2-digit", month: "short" })} – ${days[6].toLocaleDateString("pt-BR", { day: "2-digit", month: "short" })}`;

  return (
    <div className="space-y-5">
      <PageHead title="Agenda" subtitle="Aulas, chamada e escala dos professores"
        actions={<>
          <PillTabs tabs={[{ id: "semana", label: "Semana" }, { id: "turmas", label: "Turmas fixas" }]} value={view} onChange={setView} />
          {manage && <button className="btn-brand" onClick={() => setShowNew(true)}><CalendarPlus className="w-4 h-4" />Aula avulsa</button>}
        </>} />

      {view === "turmas" ? <Schedules manage={manage} teachers={teachers.data || []} onChanged={sessions.reload} /> : (
        <>
          <div className="flex flex-wrap items-center gap-3">
            <div className="inline-flex items-center gap-1 rounded-full bg-ink-750 p-1">
              <button className="btn-icon w-8 h-8 text-ink-300 hover:text-white" onClick={() => shift(-1)} aria-label="Semana anterior"><ChevronLeft className="w-4 h-4" /></button>
              <button className="px-3 text-sm font-semibold text-white" onClick={() => setMonday(mondayOf(new Date()))}>{rangeLabel}</button>
              <button className="btn-icon w-8 h-8 text-ink-300 hover:text-white" onClick={() => shift(1)} aria-label="Próxima semana"><ChevronRight className="w-4 h-4" /></button>
            </div>
            <select className="field w-auto" value={teacherFilter} onChange={(e) => setTeacherFilter(e.target.value)} aria-label="Filtrar por professor">
              <option value="">Todos os professores</option>
              <option value="none">Sem professor(a)</option>
              {teachers.data?.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
            {uncovered > 0 && <Tag tone="bad"><AlertTriangle className="w-3 h-3" />{uncovered} aula(s) sem professor(a)</Tag>}
          </div>

          {sessions.error ? <ErrorBox message={sessions.error} onRetry={sessions.reload} /> : !sessions.data ? <Loading /> : (
            <div className="space-y-4">
              {days.map((d) => {
                const list = byDay[isoDate(d)] || [];
                const today = isoDate(d) === isoDate();
                return (
                  <section key={d.toISOString()} className="card p-4">
                    <div className="flex items-center gap-2 mb-3">
                      <h3 className={`text-sm font-semibold ${today ? "text-brand" : "text-white"}`}>{WEEKDAYS[d.getDay()]}, {d.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" })}</h3>
                      {today && <Tag tone="ok">hoje</Tag>}
                    </div>
                    {list.length === 0 ? <p className="text-xs text-ink-400">Sem aulas.</p> : (
                      <ul className="space-y-2">
                        {list.map((s) => {
                          const canceled = s.status === "CANCELADA";
                          const past = new Date(s.ends_at) < new Date();
                          return (
                            <li key={s.id} className={`card-soft p-3 flex flex-wrap items-center gap-3 ${canceled ? "opacity-60" : ""}`}>
                              <span className="w-1.5 self-stretch rounded-full" style={{ background: s.modality?.color || BRAND }} />
                              <div className="w-24 text-sm font-semibold text-white">{timeBR(s.starts_at)}–{timeBR(s.ends_at)}</div>
                              <div className="flex-1 min-w-[160px]">
                                <div className="text-sm text-white">{s.title || s.modality?.name}{s.room ? <span className="text-ink-300"> · {s.room}</span> : null}</div>
                                <div className="text-xs">{s.teacher ? <span className="text-ink-300">{s.teacher.name}</span> : <span className="text-[#FF7A5C]">Sem professor(a)</span>}</div>
                              </div>
                              <div className="text-xs text-ink-300 flex items-center gap-1"><Users className="w-3.5 h-3.5" />{s.booked ?? 0}/{s.capacity}{(s.waitlist ?? 0) > 0 ? ` (+${s.waitlist} espera)` : ""}</div>
                              {canceled ? <Tag tone="mute">Cancelada</Tag> : (
                                <div className="flex gap-2">
                                  <button className="btn-dark px-3 py-2 text-xs" onClick={() => setAttend(s)}>{past ? "Presenças" : "Lista"}</button>
                                  {manage && !past && <button className="btn-dark px-3 py-2 text-xs" onClick={() => setSwap(s)}>Professor(a)</button>}
                                  {manage && !past && <button className="btn-ghost px-3 py-2 text-xs" onClick={() => setCancel(s)}>Cancelar</button>}
                                </div>
                              )}
                            </li>
                          );
                        })}
                      </ul>
                    )}
                  </section>
                );
              })}
            </div>
          )}
        </>
      )}

      {attend && <AttendeesModal session={attend} onClose={() => setAttend(null)} onChanged={sessions.reload} />}
      {swap && <SwapModal session={swap} teachers={teachers.data || []} onClose={() => setSwap(null)} onDone={sessions.reload} />}
      {cancel && (
        <ConfirmDialog title="Cancelar aula" danger confirmLabel="Cancelar aula" askReason="Motivo (as alunas com reserva são avisados)" busy={busy}
          message={`${cancel.title || cancel.modality?.name} — ${timeBR(cancel.starts_at)}. Quem reservou recebe uma notificação.`}
          onClose={() => setCancel(null)}
          onConfirm={async (reason) => { const r = await run(() => api.cancelSession(cancel.id, reason), "Aula cancelada"); if (r) { setCancel(null); sessions.reload(); } }} />
      )}
      {showNew && <NewSessionModal teachers={teachers.data || []} onClose={() => setShowNew(false)} onDone={sessions.reload} />}
    </div>
  );
}

/* ------------------------------ Chamada ------------------------------ */
function AttendeesModal({ session, onClose, onChanged }: { session: ClassSession; onClose: () => void; onChanged: () => void }) {
  const { data, loading, error, reload } = useAsync(() => api.sessionAttendees(session.id), [session.id]);
  const { busy, run } = useAction();
  const mark = async (b: Booking, status: "PRESENTE" | "FALTA") => {
    const r = await run(() => api.markAttendance(b.id, status));
    if (r !== undefined) { reload(); onChanged(); }
  };
  return (
    <Modal title={`${session.title || session.modality?.name} · ${timeBR(session.starts_at)}`} onClose={onClose}>
      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading rows={2} /> : data && data.length === 0 ? <Empty title="Ninguém reservou ainda" /> : (
        <ul className="space-y-2">
          {data?.map((b) => {
            const name = b.student?.user?.name || b.lead?.name || "—";
            return (
              <li key={b.id} className="card-soft p-3 flex items-center gap-3">
                <Avatar name={name} size={36} tone="dark" />
                <div className="flex-1 min-w-0"><div className="text-sm text-white truncate">{name}</div><div className="text-[11px] text-ink-300">{b.lead ? "Aula experimental" : b.status === "ESPERA" ? "Lista de espera" : "Aluna"}</div></div>
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

/* -------------------------- Troca de professor(a) -------------------------- */
function SwapModal({ session, teachers, onClose, onDone }: { session: ClassSession; teachers: { id: string; name: string }[]; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [teacher, setTeacher] = useState(session.teacher_id || "");
  return (
    <Modal title="Professor(a) desta aula" onClose={onClose} width="max-w-md"
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-brand" disabled={busy || teacher === (session.teacher_id || "")}
        onClick={async () => { const r = await run(() => api.updateSession(session.id, { teacher_id: teacher || "00000000-0000-0000-0000-000000000000" }), "Professor(a) atualizado(a)"); if (r) { onDone(); onClose(); } }}>Salvar</button></>}>
      <p className="text-sm text-ink-300">Vale só para esta aula. Quem reservou e a nova pessoa são avisados. O sistema não deixa escalar alguém que já tenha aula no mesmo horário.</p>
      <Field label="Professor(a)">
        <select className="field" value={teacher} onChange={(e) => setTeacher(e.target.value)}>
          <option value="">Sem professor(a)</option>
          {teachers.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </select>
      </Field>
    </Modal>
  );
}

/* ---------------------------- Aula avulsa ---------------------------- */
function NewSessionModal({ teachers, onClose, onDone }: { teachers: { id: string; name: string }[]; onClose: () => void; onDone: () => void }) {
  const mods = useAsync(() => api.listModalities(), []);
  const { busy, run } = useAction();
  const [f, setF] = useState({ modality_id: "", teacher_id: "", title: "", date: isoDate(), start_time: "18:00", duration_min: 60, capacity: 15, room: "", notes: "" });
  return (
    <Modal title="Aula avulsa / aulão" onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-brand" disabled={busy || !f.modality_id}
        onClick={async () => { const r = await run(() => api.createSession({ ...f, teacher_id: f.teacher_id || null }), "Aula criada"); if (r) { onDone(); onClose(); } }}>Criar</button></>}>
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="Modalidade"><select className="field" value={f.modality_id} onChange={(e) => setF({ ...f, modality_id: e.target.value })}><option value="">Escolha…</option>{mods.data?.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}</select></Field>
        <Field label="Professor(a)"><select className="field" value={f.teacher_id} onChange={(e) => setF({ ...f, teacher_id: e.target.value })}><option value="">A definir</option>{teachers.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}</select></Field>
        <Field label="Data"><input type="date" className="field" value={f.date} min={isoDate()} onChange={(e) => setF({ ...f, date: e.target.value })} /></Field>
        <Field label="Horário"><input type="time" className="field" value={f.start_time} onChange={(e) => setF({ ...f, start_time: e.target.value })} /></Field>
        <Field label="Duração (min)"><NumberInput value={f.duration_min} min={15} max={240} onChange={(n) => setF({ ...f, duration_min: n })} /></Field>
        <Field label="Vagas"><NumberInput value={f.capacity} min={1} onChange={(n) => setF({ ...f, capacity: n })} /></Field>
      </div>
      <Field label="Nome do evento (opcional)"><input className="field" placeholder="Ex.: Aulão de Zumba" value={f.title} onChange={(e) => setF({ ...f, title: e.target.value })} /></Field>
      <Field label="Sala"><input className="field" value={f.room} onChange={(e) => setF({ ...f, room: e.target.value })} /></Field>
    </Modal>
  );
}

/* ---------------------------- Turmas fixas ---------------------------- */
function Schedules({ manage, teachers, onChanged }: { manage: boolean; teachers: { id: string; name: string }[]; onChanged: () => void }) {
  const { data, loading, error, reload } = useAsync(() => api.listSchedules(), []);
  const [edit, setEdit] = useState<Partial<ClassSchedule> | null>(null);
  const [del, setDel] = useState<ClassSchedule | null>(null);
  const { busy, run } = useAction();
  const grouped = useMemo(() => {
    const g: Record<number, ClassSchedule[]> = {};
    (data || []).forEach((s) => (g[s.weekday] ||= []).push(s));
    return g;
  }, [data]);
  const refresh = () => { reload(); onChanged(); };

  if (error) return <ErrorBox message={error} onRetry={reload} />;
  if (loading && !data) return <Loading />;
  return (
    <div className="space-y-4">
      {manage && <div className="flex justify-end"><button className="btn-brand" onClick={() => setEdit({ weekday: 1, start_time: "18:00", duration_min: 60, capacity: 15 })}><Repeat className="w-4 h-4" />Nova turma fixa</button></div>}
      {(data || []).length === 0 ? <div className="card"><Empty title="Nenhuma turma fixa">As turmas se repetem toda semana e geram as aulas automaticamente.</Empty></div> : (
        [1, 2, 3, 4, 5, 6, 0].filter((d) => grouped[d]).map((d) => (
          <section key={d} className="card p-4">
            <h3 className="text-sm font-semibold text-white mb-3">{WEEKDAYS[d]}</h3>
            <ul className="space-y-2">
              {grouped[d].map((s) => (
                <li key={s.id} className="card-soft p-3 flex flex-wrap items-center gap-3">
                  <span className="w-1.5 self-stretch rounded-full" style={{ background: s.modality?.color || BRAND }} />
                  <div className="w-16 text-sm font-semibold text-white">{s.start_time}</div>
                  <div className="flex-1 min-w-[150px]"><div className="text-sm text-white">{s.modality?.name} · {s.duration_min}min</div><div className="text-xs text-ink-300">{s.teacher?.name || "Sem professor(a)"} · {s.capacity} vagas{s.room ? ` · ${s.room}` : ""}</div></div>
                  {manage && <div className="flex gap-2"><button className="btn-dark px-3 py-2 text-xs" onClick={() => setEdit(s)}>Editar</button><button className="btn-ghost px-3 py-2 text-xs" onClick={() => setDel(s)}>Encerrar</button></div>}
                </li>
              ))}
            </ul>
          </section>
        ))
      )}
      {edit && <ScheduleModal initial={edit} teachers={teachers} onClose={() => setEdit(null)} onDone={refresh} />}
      {del && <ConfirmDialog title="Encerrar turma" danger confirmLabel="Encerrar turma" busy={busy}
        message={`${WEEKDAYS[del.weekday]} ${del.start_time} — ${del.modality?.name}. As aulas futuras sem reserva são removidas; as que têm reserva são canceladas e as alunas avisados.`}
        onClose={() => setDel(null)} onConfirm={async () => { const r = await run(() => api.deleteSchedule(del.id), "Turma encerrada"); if (r !== undefined) { setDel(null); refresh(); } }} />}
    </div>
  );
}

function ScheduleModal({ initial, teachers, onClose, onDone }: { initial: Partial<ClassSchedule>; teachers: { id: string; name: string }[]; onClose: () => void; onDone: () => void }) {
  const mods = useAsync(() => api.listModalities(), []);
  const { busy, run } = useAction();
  const [f, setF] = useState({ modality_id: initial.modality_id || "", teacher_id: initial.teacher_id || "", weekday: initial.weekday ?? 1, start_time: initial.start_time || "18:00", duration_min: initial.duration_min || 60, capacity: initial.capacity || 15, room: initial.room || "" });
  const save = async () => {
    const payload = { ...f, teacher_id: f.teacher_id || null };
    const r = await run(() => (initial.id ? api.updateSchedule(initial.id, payload) : api.createSchedule(payload)), "Turma salva");
    if (r) { onDone(); onClose(); }
  };
  return (
    <Modal title={initial.id ? "Editar turma fixa" : "Nova turma fixa"} onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-brand" disabled={busy || !f.modality_id} onClick={save}>Salvar</button></>}>
      {initial.id && <p className="text-xs text-ink-300">Mudar dia ou horário recria as aulas futuras e avisa quem tinha reserva.</p>}
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="Modalidade"><select className="field" value={f.modality_id} onChange={(e) => setF({ ...f, modality_id: e.target.value })}><option value="">Escolha…</option>{mods.data?.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}</select></Field>
        <Field label="Professor(a)"><select className="field" value={f.teacher_id} onChange={(e) => setF({ ...f, teacher_id: e.target.value })}><option value="">A definir</option>{teachers.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}</select></Field>
        <Field label="Dia da semana"><select className="field" value={f.weekday} onChange={(e) => setF({ ...f, weekday: Number(e.target.value) })}>{WEEKDAYS.map((w, i) => <option key={i} value={i}>{w}</option>)}</select></Field>
        <Field label="Horário"><input type="time" className="field" value={f.start_time} onChange={(e) => setF({ ...f, start_time: e.target.value })} /></Field>
        <Field label="Duração (min)"><NumberInput value={f.duration_min} min={15} max={240} onChange={(n) => setF({ ...f, duration_min: n })} /></Field>
        <Field label="Vagas"><NumberInput value={f.capacity} min={1} onChange={(n) => setF({ ...f, capacity: n })} /></Field>
      </div>
      <Field label="Sala"><input className="field" value={f.room} onChange={(e) => setF({ ...f, room: e.target.value })} /></Field>
    </Modal>
  );
}
