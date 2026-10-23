"use client";

import React, { useState } from "react";
import { api } from "@/services/api";
import { Field, Modal } from "@/components/ui";
import { useAction } from "@/lib/hooks";
import { brl, dateBR, PAYMENT_METHODS } from "@/lib/format";
import { NumberInput } from "./shared";
import type { Invoice, PaymentMethod } from "@/types";

/** Baixa manual de uma cobrança (recepção recebeu o pagamento) */
export default function PayModal({ invoice, onClose, onDone }: { invoice: Invoice; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const [method, setMethod] = useState<PaymentMethod>("PIX");
  const [discount, setDiscount] = useState(0);
  const total = Math.max(0, invoice.amount - discount);

  const submit = async () => {
    const r = await run(() => api.payInvoice(invoice.id, { method, discount }), "Pagamento registrado");
    if (r) {
      onDone();
      onClose();
    }
  };

  return (
    <Modal
      title="Receber pagamento"
      onClose={onClose}
      width="max-w-md"
      footer={
        <>
          <button className="btn-ghost" onClick={onClose}>Cancelar</button>
          <button className="btn-brand" disabled={busy} onClick={submit}>{busy ? "Registrando…" : `Receber ${brl(total)}`}</button>
        </>
      }
    >
      <div className="card-soft p-4 text-sm space-y-1">
        <div className="text-white font-medium">{invoice.student?.user?.name}</div>
        <div className="text-ink-300">{invoice.description}</div>
        <div className="text-ink-300">Vencimento {dateBR(invoice.due_date)} · valor {brl(invoice.amount)}</div>
      </div>
      <Field label="Forma de pagamento">
        <div className="flex flex-wrap gap-2">
          {PAYMENT_METHODS.map((m) => (
            <button key={m.id} type="button" onClick={() => setMethod(m.id)} className={method === m.id ? "pill-on" : "pill-off"}>{m.label}</button>
          ))}
        </div>
      </Field>
      <Field label="Desconto (R$)" hint="Fica registrado no histórico de ações">
        <NumberInput value={discount} min={0} max={invoice.amount} step={0.01} onChange={setDiscount} />
      </Field>
    </Modal>
  );
}
