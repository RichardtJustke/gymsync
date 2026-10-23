"use client";

import React, { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { Search } from "lucide-react";
import { api } from "@/services/api";
import { Avatar, Empty, SectionHeader, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { STUDENT_STATUS, timeBR } from "@/lib/format";
import { ErrorBox, PageHead } from "./shared";
import { useToast } from "@/context/ToastContext";

export default function Reception() {
  const toast = useToast();
  const today = useAsync(() => api.getTodayCheckIns(), []);
  const [qr, setQr] = useState<{ payload: string; code: string; left: number } | null>(null);
  const [qrError, setQrError] = useState("");
  const [q, setQ] = useState("");
  const found = useAsync(() => (q.trim().length >= 2 ? api.listStudents(q, "", 1, 6) : Promise.resolve(null)), [q]);
  const { busy, run } = useAction();

  // O código muda sozinho; busca o próximo pouco antes de expirar
  useEffect(() => {
    let alive = true;
    let timer: ReturnType<typeof setTimeout>;
    let tick: ReturnType<typeof setInterval>;
    const load = async () => {
      try {
        const r = await api.getQRCode();
        if (!alive) return;
        setQr({ payload: r.qr_payload, code: r.code, left: r.expires_in });
        setQrError("");
        timer = setTimeout(load, Math.max(2, r.expires_in - 1) * 1000);
      } catch (e) {
        if (alive) { setQrError(e instanceof Error ? e.message : "Erro"); timer = setTimeout(load, 8000); }
      }
    };
    load();
    tick = setInterval(() => setQr((p) => (p ? { ...p, left: Math.max(0, p.left - 1) } : p)), 1000);
    return () => { alive = false; clearTimeout(timer); clearInterval(tick); };
  }, []);

  const checkIn = async (id: string, name: string) => {
    const r = await run(() => api.manualCheckIn(id));
    if (r) {
      toast[r.already ? "info" : "success"](r.already ? `${name} já tem check-in registrado hoje.` : `Entrada de ${name} registrada.`);
      setQ("");
      today.reload();
    }
  };

  return (
    <div className="space-y-6">
      <PageHead title="Recepção" subtitle="Mostre o QR para a aluna escanear no app, ou registre a entrada à mão" />
      <div className="grid gap-6 lg:grid-cols-2">
        <section className="card p-6 flex flex-col items-center text-center">
          <SectionHeader title="QR de entrada" />
          {qrError ? <ErrorBox message={qrError} /> : !qr ? <div className="w-64 h-64 rounded-3xl bg-ink-750 animate-pulse" /> : (
            <>
              <div className="bg-white p-4 rounded-3xl"><QRCodeSVG value={qr.payload} size={232} level="M" /></div>
              <div className="mt-4 font-mono text-3xl tracking-[0.3em] text-white">{qr.code}</div>
              <div className="mt-2 text-xs text-ink-300">Renova em {qr.left}s · a aluna também pode digitar o código no app</div>
            </>
          )}
        </section>

        <section className="card p-6">
          <SectionHeader title="Check-in manual" />
          <div className="relative">
            <Search className="w-4 h-4 absolute left-4 top-1/2 -translate-y-1/2 text-ink-400" />
            <input className="field pl-11" placeholder="Nome, CPF ou telefone do aluno" value={q} onChange={(e) => setQ(e.target.value)} />
          </div>
          {found.data && (
            <ul className="mt-3 space-y-2">
              {found.data.data.length === 0 && <li className="text-xs text-ink-400 p-2">Ninguém encontrado.</li>}
              {found.data.data.map((s) => {
                const st = STUDENT_STATUS[s.status];
                return (
                  <li key={s.id} className="card-soft p-3 flex items-center gap-3">
                    <Avatar name={s.user.name} size={38} tone="dark" />
                    <div className="flex-1 min-w-0"><div className="text-sm text-white truncate">{s.user.name}</div><div className="text-[11px] text-ink-300">{s.plan_name || "Sem plano"}</div></div>
                    <Tag tone={st.tone}>{st.label}</Tag>
                    <button className="btn-brand px-4 py-2 text-xs" disabled={busy} onClick={() => checkIn(s.id, s.user.name)}>Liberar</button>
                  </li>
                );
              })}
            </ul>
          )}
          <p className="text-[11px] text-ink-400 mt-3">Alunas inadimplentes ou canceladas não têm a entrada liberada.</p>
        </section>
      </div>

      <section className="card p-5">
        <SectionHeader title={`Entradas de hoje${today.data ? ` (${today.data.length})` : ""}`} />
        {today.error ? <ErrorBox message={today.error} onRetry={today.reload} /> : !today.data || today.data.length === 0 ? <Empty title="Ninguém entrou ainda" /> : (
          <ul className="grid gap-2 sm:grid-cols-2">
            {today.data.map((c) => (
              <li key={c.id} className="card-soft p-3 flex items-center gap-3">
                <Avatar name={c.student?.user?.name} size={36} tone="dark" />
                <div className="flex-1 min-w-0 text-sm text-white truncate">{c.student?.user?.name}</div>
                <span className="text-xs text-ink-300">{timeBR(c.created_at)}</span>
                <Tag tone="mute">{c.method === "MANUAL" ? "Manual" : "QR"}</Tag>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
