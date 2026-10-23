"use client";

import React, { useState } from "react";
import { api, clearToken } from "@/services/api";
import { useSession } from "@/lib/session";
import { Field, Modal } from "@/components/ui";
import { useAction } from "@/lib/hooks";

/** Quem entra com senha provisória precisa criar a própria senha antes de usar o sistema */
export default function FirstAccessGate() {
  const { user, ready } = useSession();
  const { busy, run } = useAction();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [again, setAgain] = useState("");
  if (!ready || !user?.first_access) return null;

  const tooShort = next.length < 8;
  const mismatch = next !== again;
  const submit = async () => {
    const r = await run(() => api.changePassword(current, next), "Senha criada! Bem-vinda.");
    if (r) await api.getMe();
  };
  const logout = () => { clearToken(); window.location.href = "/"; };

  return (
    <Modal title="Crie sua senha" onClose={() => {}} hideClose width="max-w-md"
      footer={<><button className="btn-ghost" onClick={logout}>Sair</button><button className="btn-brand" disabled={busy || tooShort || mismatch || !current} onClick={submit}>{busy ? "Salvando…" : "Salvar senha"}</button></>}>
      <p className="text-sm text-ink-200">Você entrou com uma senha provisória. Escolha uma senha só sua para continuar.</p>
      <Field label="Senha provisória (a que você recebeu)"><input className="field" type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} /></Field>
      <Field label="Nova senha" hint="Mínimo de 8 caracteres"><input className="field" type="password" autoComplete="new-password" value={next} onChange={(e) => setNext(e.target.value)} /></Field>
      <Field label="Repita a nova senha"><input className="field" type="password" autoComplete="new-password" value={again} onChange={(e) => setAgain(e.target.value)} /></Field>
      {again && mismatch && <p className="text-xs text-[#FF7A5C]">As senhas não conferem.</p>}
    </Modal>
  );
}
