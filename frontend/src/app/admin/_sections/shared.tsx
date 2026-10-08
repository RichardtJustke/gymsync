"use client";

import React from "react";
import { Empty } from "@/components/ui";

export function Loading({ rows = 3 }: { rows?: number }) {
  return (
    <div className="space-y-3" aria-busy="true">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="h-16 rounded-2xl bg-ink-750 animate-pulse" />
      ))}
    </div>
  );
}

export function ErrorBox({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="card p-6">
      <Empty title="Não foi possível carregar">
        {message}
        {onRetry && (
          <div className="mt-3">
            <button className="btn-dark" onClick={onRetry}>Tentar de novo</button>
          </div>
        )}
      </Empty>
    </div>
  );
}

export function PageHead({ title, subtitle, actions }: { title: string; subtitle?: string; actions?: React.ReactNode }) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h2 className="text-2xl font-semibold text-white tracking-tight">{title}</h2>
        {subtitle && <p className="text-sm text-ink-300 mt-0.5">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}

/** Credenciais provisórias exibidas uma única vez */
export function Credentials({ login, password }: { login: string; password: string }) {
  return (
    <div className="card-soft p-4 space-y-2 text-sm">
      <div className="text-ink-300">Login</div>
      <div className="font-mono text-white break-all">{login}</div>
      <div className="text-ink-300 pt-1">Senha provisória (aparece só agora)</div>
      <div className="font-mono text-lime text-lg tracking-wider select-all">{password}</div>
      <button className="btn-dark w-full mt-1" onClick={() => navigator.clipboard?.writeText(`Login: ${login}\nSenha provisória: ${password}`)}>
        Copiar login e senha
      </button>
    </div>
  );
}

export function NumberInput({ value, onChange, ...rest }: { value: number; onChange: (n: number) => void } & Omit<React.InputHTMLAttributes<HTMLInputElement>, "value" | "onChange">) {
  return (
    <input
      type="number"
      inputMode="decimal"
      className="field"
      value={Number.isFinite(value) ? value : ""}
      onChange={(e) => onChange(e.target.value === "" ? 0 : Number(e.target.value))}
      {...rest}
    />
  );
}
