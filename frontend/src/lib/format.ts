import type { InvoiceStatus, PaymentMethod, Role, StudentStatus } from "@/types";

export const brl = (n?: number | null) =>
  (n ?? 0).toLocaleString("pt-BR", { style: "currency", currency: "BRL", minimumFractionDigits: 2 });

export const num = (n?: number | null, digits = 0) =>
  (n ?? 0).toLocaleString("pt-BR", { minimumFractionDigits: digits, maximumFractionDigits: digits });

export const pct = (n?: number | null) => `${Math.round(n ?? 0)}%`;

export const dateBR = (s?: string | null) => (s ? new Date(s).toLocaleDateString("pt-BR") : "—");

export const dateTimeBR = (s?: string | null) =>
  s ? new Date(s).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }) : "—";

export const timeBR = (s?: string | null) => (s ? new Date(s).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" }) : "—");

export const WEEKDAYS = ["Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"];
export const WEEKDAYS_SHORT = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

/** AAAA-MM do mês atual (ou deslocado) */
export function monthKey(offset = 0, base = new Date()) {
  const d = new Date(base.getFullYear(), base.getMonth() + offset, 1);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

export function shiftMonth(key: string, delta: number) {
  const [y, m] = key.split("-").map(Number);
  return monthKey(delta, new Date(y, m - 1, 1));
}

export function monthLabel(key: string, short = false) {
  const [y, m] = key.split("-").map(Number);
  const s = new Date(y, m - 1, 1).toLocaleDateString("pt-BR", { month: short ? "short" : "long", year: short ? "2-digit" : "numeric" });
  return s.charAt(0).toUpperCase() + s.slice(1).replace(".", "");
}

/** AAAA-MM-DD no fuso local */
export function isoDate(d = new Date()) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export const ROLE_LABEL: Record<Role, string> = {
  ADMIN: "Dono(a)",
  GERENTE: "Gerente",
  RECEPCAO: "Recepção",
  CONSULTOR: "Consultor(a)",
  PROFESSOR: "Professor(a)",
  ALUNO: "Aluno(a)",
};

export const STUDENT_STATUS: Record<StudentStatus, { label: string; tone: "ok" | "bad" | "warn" | "mute" }> = {
  ATIVO: { label: "Ativo", tone: "ok" },
  INADIMPLENTE: { label: "Inadimplente", tone: "bad" },
  CANCELADO: { label: "Cancelado", tone: "mute" },
  PENDENTE: { label: "Pendente", tone: "warn" },
};

export const INVOICE_STATUS: Record<InvoiceStatus, { label: string; tone: "ok" | "bad" | "warn" | "mute" }> = {
  PAGO: { label: "Pago", tone: "ok" },
  PENDENTE: { label: "Em aberto", tone: "warn" },
  VENCIDO: { label: "Vencido", tone: "bad" },
  CANCELADO: { label: "Cancelado", tone: "mute" },
};

export const PAYMENT_METHODS: { id: PaymentMethod; label: string }[] = [
  { id: "PIX", label: "PIX" },
  { id: "CARTAO", label: "Cartão" },
  { id: "DINHEIRO", label: "Dinheiro" },
  { id: "BOLETO", label: "Boleto" },
  { id: "TRANSFERENCIA", label: "Transferência" },
];

export const methodLabel = (m?: string | null) => PAYMENT_METHODS.find((x) => x.id === m)?.label || m || "—";

export function whatsappLink(phone?: string, text = "") {
  const digits = (phone || "").replace(/\D/g, "");
  if (digits.length < 10) return "";
  const full = digits.length <= 11 ? `55${digits}` : digits;
  return `https://wa.me/${full}${text ? `?text=${encodeURIComponent(text)}` : ""}`;
}

/** Baixa um CSV (separador ; para abrir direto no Excel em pt-BR) */
export function downloadCSV(filename: string, header: string[], rows: (string | number)[][]) {
  const esc = (v: string | number) => {
    const s = typeof v === "number" ? v.toLocaleString("pt-BR") : v ?? "";
    return /[;"\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  const csv = "﻿" + [header, ...rows].map((r) => r.map(esc).join(";")).join("\n");
  const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
