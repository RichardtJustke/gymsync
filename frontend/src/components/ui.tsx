"use client";

import React from "react";

/* ------------------------------------------------------------------ */
/* Ring de progresso circular                                          */
/* ------------------------------------------------------------------ */
export function Ring({
  value,
  size = 64,
  stroke = 7,
  color = "#D7FF3A",
  track = "rgba(255,255,255,0.08)",
  children,
}: {
  value: number; // 0..100
  size?: number;
  stroke?: number;
  color?: string;
  track?: string;
  children?: React.ReactNode;
}) {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const v = Math.max(0, Math.min(100, value));
  return (
    <div className="relative inline-flex items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} stroke={track} strokeWidth={stroke} fill="none" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={color}
          strokeWidth={stroke}
          fill="none"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c - (v / 100) * c}
          style={{ transition: "stroke-dashoffset .6s cubic-bezier(.2,.8,.2,1)" }}
        />
      </svg>
      {children && <div className="absolute inset-0 flex items-center justify-center">{children}</div>}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Mini gráfico de barras                                              */
/* ------------------------------------------------------------------ */
export function Bars({
  values,
  labels,
  highlight,
  height = 72,
  dim = false,
}: {
  values: number[];
  labels?: string[];
  highlight?: number;
  height?: number;
  dim?: boolean;
}) {
  const max = Math.max(...values, 1);
  return (
    <div className="flex items-end gap-1.5">
      {values.map((v, i) => {
        const on = highlight === undefined ? !dim : i === highlight;
        return (
          <div key={i} className="flex flex-col items-center gap-1.5">
            <div
              className={`w-3.5 rounded-[5px] transition-all ${on ? "bg-lime" : dim ? "bg-white/10" : "bg-lime/40"}`}
              style={{ height: Math.max(6, (v / max) * height) }}
            />
            {labels && <span className="text-[10px] text-ink-300">{labels[i]}</span>}
          </div>
        );
      })}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Faixa semanal (seg..dom) com dia atual em destaque                  */
/* ------------------------------------------------------------------ */
const WEEK = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

export function WeekStrip() {
  const today = new Date();
  const monday = new Date(today);
  const dow = (today.getDay() + 6) % 7; // 0 = seg
  monday.setDate(today.getDate() - dow);
  const days = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    return d;
  });
  return (
    <div className="flex items-center justify-between gap-1">
      {days.map((d) => {
        const isToday = d.toDateString() === today.toDateString();
        return (
          <div
            key={d.toISOString()}
            className={`flex flex-col items-center gap-1 w-11 py-2.5 rounded-2xl transition-all ${isToday ? "bg-lime text-black shadow-lime" : "text-ink-300"}`}
          >
            <span className={`text-[11px] ${isToday ? "font-semibold" : ""}`}>{WEEK[d.getDay()]}</span>
            <span className={`text-sm font-semibold ${isToday ? "" : "text-white"}`}>{d.getDate()}</span>
          </div>
        );
      })}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Avatar com iniciais                                                 */
/* ------------------------------------------------------------------ */
export function Avatar({ name, size = 44, tone = "lime" }: { name?: string; size?: number; tone?: "lime" | "dark" }) {
  const initials =
    (name || "?")
      .split(" ")
      .filter(Boolean)
      .slice(0, 2)
      .map((p) => p[0])
      .join("")
      .toUpperCase() || "?";
  return (
    <div
      className={`rounded-full flex items-center justify-center font-semibold shrink-0 ${tone === "lime" ? "bg-lime text-black" : "bg-ink-700 text-white"}`}
      style={{ width: size, height: size, fontSize: size * 0.36 }}
    >
      {initials}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Abas em formato pill                                                */
/* ------------------------------------------------------------------ */
export function PillTabs<T extends string>({
  tabs,
  value,
  onChange,
}: {
  tabs: { id: T; label: React.ReactNode; icon?: React.ReactNode }[];
  value: T;
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex items-center gap-2 overflow-x-auto no-scrollbar">
      {tabs.map((t) => (
        <button key={t.id} onClick={() => onChange(t.id)} className={value === t.id ? "pill-on" : "pill-off"}>
          {t.icon}
          {t.label}
        </button>
      ))}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Cabeçalho de seção (título + ação)                                  */
/* ------------------------------------------------------------------ */
export function SectionHeader({ title, action }: { title: React.ReactNode; action?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between mb-3">
      <h3 className="section-title">{title}</h3>
      {action && <div className="text-xs text-ink-300">{action}</div>}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Card de métrica                                                     */
/* ------------------------------------------------------------------ */
export function StatCard({
  icon,
  label,
  value,
  hint,
  accent = "#D7FF3A",
  progress,
  variant = "dark",
}: {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
  hint?: React.ReactNode;
  accent?: string;
  progress?: number;
  variant?: "dark" | "lime";
}) {
  const lime = variant === "lime";
  return (
    <div className={`${lime ? "card-lime" : "card"} p-5 flex flex-col justify-between gap-4 min-h-[148px] animate-fade-up`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span
            className={`w-8 h-8 rounded-full flex items-center justify-center ${lime ? "bg-black/10" : "bg-white/5"}`}
            style={{ color: lime ? "#000" : accent }}
          >
            {icon}
          </span>
          <span className={`text-sm font-medium ${lime ? "text-black/70" : "text-ink-200"}`}>{label}</span>
        </div>
      </div>
      <div className="flex items-end justify-between gap-3">
        <div>
          <div className={`text-3xl font-semibold tracking-tight ${lime ? "text-black" : "text-white"}`}>{value}</div>
          {hint && <div className={`text-xs mt-1 ${lime ? "text-black/60" : "text-ink-300"}`}>{hint}</div>}
        </div>
        {progress !== undefined && (
          <Ring value={progress} size={52} stroke={6} color={lime ? "#000" : accent} track={lime ? "rgba(0,0,0,0.12)" : undefined} />
        )}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Alerta inline                                                       */
/* ------------------------------------------------------------------ */
export function Notice({ tone, children, icon }: { tone: "ok" | "bad"; children: React.ReactNode; icon?: React.ReactNode }) {
  return (
    <div
      className={`p-4 rounded-2xl text-sm flex items-center gap-3 animate-fade-up ${tone === "ok" ? "bg-lime/10 text-lime" : "bg-[#FF5A36]/10 text-[#FF7A5C]"}`}
    >
      {icon}
      <span>{children}</span>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Estado vazio                                                        */
/* ------------------------------------------------------------------ */
export function Empty({ icon, title, children }: { icon?: React.ReactNode; title?: string; children?: React.ReactNode }) {
  return (
    <div className="py-12 px-6 text-center space-y-2">
      {icon && <div className="w-14 h-14 rounded-full bg-white/5 text-ink-300 flex items-center justify-center mx-auto mb-3">{icon}</div>}
      {title && <div className="text-sm font-semibold text-white">{title}</div>}
      {children && <div className="text-xs text-ink-300 max-w-xs mx-auto">{children}</div>}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Modal e gaveta lateral                                              */
/* ------------------------------------------------------------------ */
function useEscape(onClose: () => void) {
  React.useEffect(() => {
    const h = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [onClose]);
}

export function Modal({
  title,
  onClose,
  children,
  footer,
  width = "max-w-lg",
  hideClose,
}: {
  title: React.ReactNode;
  onClose: () => void;
  children: React.ReactNode;
  footer?: React.ReactNode;
  width?: string;
  hideClose?: boolean;
}) {
  useEscape(onClose);
  return (
    <div className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center p-0 sm:p-4" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={hideClose ? undefined : onClose} />
      <div className={`relative w-full ${width} max-h-[92vh] flex flex-col card rounded-b-none sm:rounded-b-3xl animate-fade-up`}>
        <div className="flex items-center justify-between gap-3 p-5 pb-3">
          <h3 className="section-title">{title}</h3>
          {!hideClose && <button onClick={onClose} className="btn-icon w-9 h-9 bg-ink-750 text-ink-300 hover:text-white" aria-label="Fechar">✕</button>}
        </div>
        <div className="px-5 pb-5 overflow-y-auto space-y-4">{children}</div>
        {footer && <div className="p-5 pt-3 border-t border-white/5 flex flex-wrap justify-end gap-2">{footer}</div>}
      </div>
    </div>
  );
}

export function Drawer({ title, onClose, children, width = "max-w-xl" }: { title: React.ReactNode; onClose: () => void; children: React.ReactNode; width?: string }) {
  useEscape(onClose);
  return (
    <div className="fixed inset-0 z-[100] flex justify-end" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={onClose} />
      <div className={`relative w-full ${width} h-full bg-ink-850 border-l border-white/5 flex flex-col animate-fade-up`}>
        <div className="flex items-center justify-between gap-3 p-5 border-b border-white/5">
          <h3 className="section-title">{title}</h3>
          <button onClick={onClose} className="btn-icon w-9 h-9 bg-ink-750 text-ink-300 hover:text-white" aria-label="Fechar">✕</button>
        </div>
        <div className="flex-1 overflow-y-auto p-5 space-y-5">{children}</div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Campo de formulário com rótulo                                      */
/* ------------------------------------------------------------------ */
export function Field({ label, hint, children, className = "" }: { label: string; hint?: string; children: React.ReactNode; className?: string }) {
  return (
    <label className={`block ${className}`}>
      <span className="label">{label}</span>
      {children}
      {hint && <span className="block text-[11px] text-ink-400 mt-1 ml-1">{hint}</span>}
    </label>
  );
}

/* ------------------------------------------------------------------ */
/* Badge por tom                                                       */
/* ------------------------------------------------------------------ */
export function Tag({ tone = "mute", children }: { tone?: "ok" | "bad" | "warn" | "mute"; children: React.ReactNode }) {
  const cls = {
    ok: "badge-ok",
    bad: "badge-bad",
    mute: "badge-mute",
    warn: "badge bg-amber-400/15 text-amber-300",
  }[tone];
  return <span className={cls}>{children}</span>;
}

/* ------------------------------------------------------------------ */
/* Seletor de mês                                                      */
/* ------------------------------------------------------------------ */
export function MonthPicker({ value, onChange, label }: { value: string; onChange: (v: string) => void; label: (v: string) => string }) {
  const shift = (d: number) => {
    const [y, m] = value.split("-").map(Number);
    const n = new Date(y, m - 1 + d, 1);
    onChange(`${n.getFullYear()}-${String(n.getMonth() + 1).padStart(2, "0")}`);
  };
  return (
    <div className="inline-flex items-center gap-1 rounded-full bg-ink-750 p-1">
      <button onClick={() => shift(-1)} className="btn-icon w-8 h-8 text-ink-300 hover:text-white" aria-label="Mês anterior">‹</button>
      <span className="px-2 text-sm font-semibold text-white min-w-[120px] text-center">{label(value)}</span>
      <button onClick={() => shift(1)} className="btn-icon w-8 h-8 text-ink-300 hover:text-white" aria-label="Próximo mês">›</button>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Barra de progresso                                                  */
/* ------------------------------------------------------------------ */
export function Progress({ value, color = "#D7FF3A", height = 8 }: { value: number; color?: string; height?: number }) {
  const v = Math.max(0, Math.min(100, value));
  return (
    <div className="w-full rounded-full bg-white/10 overflow-hidden" style={{ height }}>
      <div className="h-full rounded-full transition-all duration-500" style={{ width: `${v}%`, background: color }} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Gráfico de linha/área simples (SVG)                                 */
/* ------------------------------------------------------------------ */
export function LineChart({
  values,
  labels,
  height = 120,
  color = "#D7FF3A",
  format,
}: {
  values: number[];
  labels?: string[];
  height?: number;
  color?: string;
  format?: (n: number) => string;
}) {
  const w = 100;
  const max = Math.max(...values, 1);
  const step = values.length > 1 ? w / (values.length - 1) : w;
  const pts = values.map((v, i) => [i * step, height - 6 - (v / max) * (height - 18)] as const);
  const line = pts.map(([x, y], i) => `${i ? "L" : "M"}${x},${y}`).join(" ");
  const area = `${line} L${(values.length - 1) * step},${height} L0,${height} Z`;
  return (
    <div>
      <svg viewBox={`0 0 ${w} ${height}`} preserveAspectRatio="none" className="w-full" style={{ height }} role="img">
        <defs>
          <linearGradient id="lc-fill" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.35" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={area} fill="url(#lc-fill)" />
        <path d={line} fill="none" stroke={color} strokeWidth="1.6" vectorEffect="non-scaling-stroke" strokeLinejoin="round" />
      </svg>
      {labels && (
        <div className="flex justify-between mt-1.5 text-[10px] text-ink-300">
          {labels.map((l, i) => (
            <span key={i} title={format ? format(values[i]) : String(values[i])}>{l}</span>
          ))}
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Confirmação com motivo opcional/obrigatório                         */
/* ------------------------------------------------------------------ */
export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Confirmar",
  danger,
  askReason,
  reasonRequired,
  busy,
  onConfirm,
  onClose,
}: {
  title: string;
  message: React.ReactNode;
  confirmLabel?: string;
  danger?: boolean;
  askReason?: string;
  reasonRequired?: boolean;
  busy?: boolean;
  onConfirm: (reason: string) => void;
  onClose: () => void;
}) {
  const [reason, setReason] = React.useState("");
  const blocked = busy || (!!askReason && !!reasonRequired && reason.trim().length < 3);
  return (
    <Modal
      title={title}
      onClose={onClose}
      width="max-w-md"
      footer={
        <>
          <button className="btn-ghost" onClick={onClose}>Voltar</button>
          <button className={danger ? "btn bg-[#FF5A36] text-white px-5 py-3 hover:brightness-110" : "btn-lime"} disabled={blocked} onClick={() => onConfirm(reason.trim())}>
            {busy ? "Aguarde…" : confirmLabel}
          </button>
        </>
      }
    >
      <div className="text-sm text-ink-200 leading-relaxed">{message}</div>
      {askReason && (
        <Field label={askReason}>
          <textarea className="field" rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
        </Field>
      )}
    </Modal>
  );
}
