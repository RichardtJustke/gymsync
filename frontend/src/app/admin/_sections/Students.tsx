"use client";

import React, { useRef, useState } from "react";
import { Download, FileSpreadsheet, Plus, Search, UploadCloud } from "lucide-react";
import { api, loadSession } from "@/services/api";
import { Avatar, ConfirmDialog, Drawer, Empty, Field, Modal, PillTabs, SectionHeader, Tag } from "@/components/ui";
import { useAction, useAsync } from "@/lib/hooks";
import { brl, dateBR, dateTimeBR, downloadCSV, INVOICE_STATUS, STUDENT_STATUS, whatsappLink } from "@/lib/format";
import { Credentials, ErrorBox, Loading, NumberInput, PageHead } from "./shared";
import PayModal from "./PayModal";
import type { ImportPreviewResult, Invoice, Permission, Student, StudentStatus } from "@/types";

const FILTERS: { id: "" | StudentStatus; label: string }[] = [
  { id: "", label: "Todos" },
  { id: "ATIVO", label: "Ativos" },
  { id: "INADIMPLENTE", label: "Inadimplentes" },
  { id: "CANCELADO", label: "Cancelados" },
];

export default function Students({ can }: { can: (...p: Permission[]) => boolean }) {
  const [q, setQ] = useState("");
  const [status, setStatus] = useState<"" | StudentStatus>("");
  const [page, setPage] = useState(1);
  const limit = 12;
  const { data, loading, error, reload } = useAsync(() => api.listStudents(q, status, page, limit), [q, status, page]);
  const [openId, setOpenId] = useState<string | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const pages = data ? Math.max(1, Math.ceil(data.total / limit)) : 1;

  return (
    <div className="space-y-5">
      <PageHead
        title="Alunas"
        subtitle={data ? `${data.total} aluna(s) encontrada(s)` : undefined}
        actions={can("students.manage") && (
          <>
            <button className="btn-dark" onClick={() => setShowImport(true)}><UploadCloud className="w-4 h-4" />Importar planilha</button>
            <button className="btn-brand" onClick={() => setShowNew(true)}><Plus className="w-4 h-4" />Nova aluna</button>
          </>
        )}
      />

      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[220px]">
          <Search className="w-4 h-4 absolute left-4 top-1/2 -translate-y-1/2 text-ink-400" />
          <input className="field pl-11" placeholder="Buscar por nome, CPF, telefone ou e-mail" value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} />
        </div>
        <PillTabs tabs={FILTERS} value={status} onChange={(v) => { setStatus(v); setPage(1); }} />
      </div>

      {error ? <ErrorBox message={error} onRetry={reload} /> : loading && !data ? <Loading /> : (
        <div className="card overflow-hidden">
          {data && data.data.length === 0 ? (
            <Empty title="Nenhuma aluna encontrada">Ajuste a busca ou o filtro.</Empty>
          ) : (
            <ul className="divide-y divide-white/5">
              {data?.data.map((s) => {
                const st = STUDENT_STATUS[s.status];
                return (
                  <li key={s.id}>
                    <button onClick={() => setOpenId(s.id)} className="w-full text-left p-4 flex items-center gap-3 hover:bg-white/[0.03] transition-colors">
                      <Avatar name={s.user.name} size={42} tone="dark" />
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-white truncate">{s.user.name}</div>
                        <div className="text-xs text-ink-300 truncate">{s.user.email || s.user.document} · {s.plan_name || "Sem plano"}{s.plan_price ? ` · ${brl(s.plan_price)}` : ""}</div>
                      </div>
                      <div className="hidden sm:block text-xs text-ink-300">vence dia {s.due_day}</div>
                      <Tag tone={st.tone}>{st.label}</Tag>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
          {pages > 1 && (
            <div className="flex items-center justify-between p-3 border-t border-white/5 text-xs text-ink-300">
              <span>Página {page} de {pages}</span>
              <div className="flex gap-2">
                <button className="btn-dark px-4 py-2" disabled={page <= 1} onClick={() => setPage(page - 1)}>Anterior</button>
                <button className="btn-dark px-4 py-2" disabled={page >= pages} onClick={() => setPage(page + 1)}>Próxima</button>
              </div>
            </div>
          )}
        </div>
      )}

      {openId && <StudentDrawer id={openId} can={can} onClose={() => setOpenId(null)} onChanged={reload} />}
      {showNew && <NewStudentModal can={can} onClose={() => setShowNew(false)} onDone={reload} />}
      {showImport && <ImportModal onClose={() => setShowImport(false)} onDone={reload} />}
    </div>
  );
}

/* ----------------------------- Nova aluna ----------------------------- */
function NewStudentModal({ can, onClose, onDone }: { can: (...p: Permission[]) => boolean; onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const plans = useAsync(() => api.listPlans(), []);
  const sellers = useAsync(async () => (can("goals.manage") ? (await api.ranking()).rows.map((r) => ({ id: r.user_id, name: r.name })) : []), []);
  const [f, setF] = useState({ name: "", email: "", document: "", phone: "", plan_id: "", due_day: 10, seller_id: "" });
  const [creds, setCreds] = useState<{ login: string; pass: string } | null>(null);
  const set = (k: string, v: string | number) => setF((p) => ({ ...p, [k]: v }));

  const submit = async () => {
    const r = await run(() => api.createStudent({ ...f, plan_id: f.plan_id || null, seller_id: f.seller_id || null }));
    if (r) {
      setCreds({ login: f.email || f.document, pass: r.temp_password });
      onDone();
    }
  };

  if (creds) {
    return (
      <Modal title="Aluna cadastrada" onClose={onClose} width="max-w-md" footer={<button className="btn-brand" onClick={onClose}>Concluir</button>}>
        <p className="text-sm text-ink-200">Passe estes dados para a aluna entrar no app. No primeiro acesso ela define uma senha nova.</p>
        <Credentials login={creds.login} password={creds.pass} />
      </Modal>
    );
  }
  const plan = plans.data?.find((p) => p.plan.id === f.plan_id)?.plan;
  return (
    <Modal
      title="Novo aluno"
      onClose={onClose}
      footer={<><button className="btn-ghost" onClick={onClose}>Cancelar</button><button className="btn-brand" disabled={busy || f.name.trim().length < 2} onClick={submit}>{busy ? "Salvando…" : "Matricular"}</button></>}
    >
      <Field label="Nome completo"><input className="field" value={f.name} onChange={(e) => set("name", e.target.value)} autoFocus /></Field>
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="E-mail"><input className="field" type="email" value={f.email} onChange={(e) => set("email", e.target.value)} /></Field>
        <Field label="CPF"><input className="field" value={f.document} onChange={(e) => set("document", e.target.value)} /></Field>
        <Field label="Telefone / WhatsApp"><input className="field" value={f.phone} onChange={(e) => set("phone", e.target.value)} /></Field>
        <Field label="Dia do vencimento"><NumberInput value={f.due_day} min={1} max={31} onChange={(n) => set("due_day", n)} /></Field>
      </div>
      <Field label="Plano" hint={plan ? `Gera a 1ª mensalidade de ${brl(plan.price)} e registra a venda` : "Sem plano, nenhuma cobrança é gerada"}>
        <select className="field" value={f.plan_id} onChange={(e) => set("plan_id", e.target.value)}>
          <option value="">Sem plano</option>
          {plans.data?.map(({ plan: p }) => <option key={p.id} value={p.id}>{p.name} — {brl(p.price)}</option>)}
        </select>
      </Field>
      {(sellers.data?.length ?? 0) > 0 && f.plan_id && (
        <Field label="Vendedor(a)" hint="Quem recebe a comissão. Padrão: você.">
          <select className="field" value={f.seller_id} onChange={(e) => set("seller_id", e.target.value)}>
            <option value="">Eu mesmo(a)</option>
            {sellers.data?.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        </Field>
      )}
      <p className="text-xs text-ink-400">É preciso e-mail ou CPF para a aluna entrar no app.</p>
    </Modal>
  );
}

/* ------------------------------ Importação ------------------------------ */
function ImportModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const { busy, run } = useAction();
  const ref = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<ImportPreviewResult | null>(null);
  const [result, setResult] = useState<Awaited<ReturnType<typeof api.executeImport>> | null>(null);

  const pick = async (file?: File) => {
    if (!file) return;
    const r = await run(() => api.uploadImportPreview(file));
    if (r) setPreview(r);
  };
  const confirm = async () => {
    if (!preview) return;
    const r = await run(() => api.executeImport(preview.rows.filter((x) => x.is_valid)), "Importação concluída");
    if (r) {
      setResult(r);
      onDone();
    }
  };

  if (result) {
    return (
      <Modal title="Importação concluída" onClose={onClose} footer={<button className="btn-brand" onClick={onClose}>Fechar</button>}>
        <div className="grid grid-cols-3 gap-3 text-center">
          <div className="card-soft p-3"><div className="text-xl font-semibold text-white">{result.created_count}</div><div className="text-xs text-ink-300">criados</div></div>
          <div className="card-soft p-3"><div className="text-xl font-semibold text-white">{result.updated_count}</div><div className="text-xs text-ink-300">atualizados</div></div>
          <div className="card-soft p-3"><div className="text-xl font-semibold text-white">{result.skipped_count}</div><div className="text-xs text-ink-300">ignorados</div></div>
        </div>
        {result.credentials && result.credentials.length > 0 && (
          <div className="space-y-2">
            <p className="text-sm text-ink-200">As senhas provisórias aparecem só agora. Baixe a lista para entregar às alunas.</p>
            <button className="btn-dark w-full" onClick={() => downloadCSV("acessos-importados.csv", ["Nome", "Login", "Senha provisória"], result.credentials!.map((c) => [c.name, c.login, c.temp_password]))}>
              <Download className="w-4 h-4" />Baixar acessos (CSV)
            </button>
          </div>
        )}
        {result.errors.length > 0 && <ul className="text-xs text-[#FF7A5C] space-y-1 max-h-32 overflow-y-auto">{result.errors.map((e, i) => <li key={i}>{e}</li>)}</ul>}
      </Modal>
    );
  }

  return (
    <Modal
      title="Importar alunas de planilha"
      onClose={onClose}
      width="max-w-3xl"
      footer={preview && <><button className="btn-ghost" onClick={() => setPreview(null)}>Trocar arquivo</button><button className="btn-brand" disabled={busy || preview.valid_rows === 0} onClick={confirm}>{busy ? "Importando…" : `Importar ${preview.valid_rows} aluna(s)`}</button></>}
    >
      {!preview ? (
        <button onClick={() => ref.current?.click()} className="w-full card-soft p-10 text-center hover:bg-ink-700 transition-colors">
          <FileSpreadsheet className="w-10 h-10 mx-auto text-brand mb-3" />
          <div className="text-white font-medium">Escolha a planilha (.xlsx ou .csv)</div>
          <div className="text-xs text-ink-300 mt-1">Pode ser a exportação do EVO. As colunas são reconhecidas automaticamente.</div>
          <input ref={ref} type="file" accept=".xlsx,.xls,.csv" className="hidden" onChange={(e) => pick(e.target.files?.[0])} />
        </button>
      ) : (
        <>
          <div className="flex flex-wrap gap-2 text-xs">
            <Tag tone="ok">{preview.valid_rows} válidas</Tag>
            {preview.invalid_rows > 0 && <Tag tone="bad">{preview.invalid_rows} com problema</Tag>}
            <Tag tone="mute">{preview.total_rows} linhas</Tag>
          </div>
          <div className="overflow-x-auto rounded-2xl border border-white/5 max-h-[45vh]">
            <table className="w-full text-xs">
              <thead className="sticky top-0 bg-ink-800 text-ink-300"><tr>{["#", "Nome", "CPF", "E-mail", "Plano", "Valor", "Venc.", ""].map((h) => <th key={h} className="text-left p-2.5 font-medium">{h}</th>)}</tr></thead>
              <tbody>
                {preview.rows.map((r) => (
                  <tr key={r.row_index} className={`border-t border-white/5 ${r.is_valid ? "" : "bg-[#FF5A36]/5"}`}>
                    <td className="p-2.5 text-ink-400">{r.row_index}</td><td className="p-2.5 text-white">{r.name}</td><td className="p-2.5">{r.document}</td>
                    <td className="p-2.5">{r.email}</td><td className="p-2.5">{r.plan_name}</td><td className="p-2.5">{brl(r.plan_price)}</td><td className="p-2.5">{r.due_day}</td>
                    <td className="p-2.5 text-[#FF7A5C]">{r.error}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </Modal>
  );
}

/* ----------------------------- Ficha da aluna ----------------------------- */
function StudentDrawer({ id, can, onClose, onChanged }: { id: string; can: (...p: Permission[]) => boolean; onClose: () => void; onChanged: () => void }) {
  const { data, loading, error, reload } = useAsync(() => api.getStudent(id), [id]);
  const { busy, run } = useAction();
  const [edit, setEdit] = useState(false);
  const [form, setForm] = useState({ name: "", email: "", phone: "", plan_id: "", due_day: 10, notes: "" });
  const [cancel, setCancel] = useState(false);
  const [pay, setPay] = useState<Invoice | null>(null);
  const [creds, setCreds] = useState<{ login: string; pass: string } | null>(null);
  const plans = useAsync(() => (can("students.manage") ? api.listPlans() : Promise.resolve([])), []);

  const refresh = () => { reload(); onChanged(); };
  if (loading && !data) return <Drawer title="Aluna" onClose={onClose}><Loading /></Drawer>;
  if (error || !data) return <Drawer title="Aluna" onClose={onClose}><ErrorBox message={error || "Sem dados"} onRetry={reload} /></Drawer>;

  const { student: s } = data;
  const st = STUDENT_STATUS[s.status];
  const startEdit = () => {
    setForm({ name: s.user.name, email: s.user.email, phone: s.user.phone, plan_id: s.plan_id || "", due_day: s.due_day, notes: s.notes || "" });
    setEdit(true);
  };
  const save = async () => {
    const payload: Record<string, unknown> = { name: form.name, email: form.email, phone: form.phone, due_day: form.due_day, notes: form.notes };
    if (form.plan_id && form.plan_id !== s.plan_id) payload.plan_id = form.plan_id;
    const r = await run(() => api.updateStudent(id, payload), "Cadastro atualizado");
    if (r) { setEdit(false); refresh(); }
  };
  const setStatus = async (status: StudentStatus, cancel_reason?: string) => {
    const r = await run(() => api.updateStudent(id, { status, cancel_reason }), status === "CANCELADO" ? "Matrícula cancelada" : "Situação atualizada");
    if (r) { setCancel(false); refresh(); }
  };
  const reset = async () => {
    const r = await run(() => api.resetStudentPassword(id));
    if (r) setCreds({ login: r.login, pass: r.temp_password });
  };
  const wa = whatsappLink(s.user.phone);

  return (
    <Drawer title="Ficha do aluno" onClose={onClose}>
      <div className="flex items-center gap-4">
        <Avatar name={s.user.name} size={60} />
        <div className="flex-1 min-w-0">
          <div className="text-lg font-semibold text-white truncate">{s.user.name}</div>
          <div className="text-xs text-ink-300 truncate">{s.user.email || "sem e-mail"} · {s.user.phone || "sem telefone"}</div>
          <div className="mt-1.5 flex flex-wrap gap-1.5"><Tag tone={st.tone}>{st.label}</Tag><Tag tone="mute">{s.plan_name || "Sem plano"}{s.plan_price ? ` · ${brl(s.plan_price)}` : ""}</Tag></div>
        </div>
      </div>

      {s.status === "CANCELADO" && (
        <div className="card-soft p-3 text-xs text-ink-200">Cancelada em {dateBR(s.canceled_at)} · motivo: {s.cancel_reason || "não informado"}</div>
      )}

      {s.status === "ATIVO" && data.checkins_last30 === 0 && (
        <div className="p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-between gap-3 text-xs">
          <div className="text-amber-300">
            ⚠️ <strong>Risco de Evasão:</strong> Aluna sem check-in nos últimos 30 dias.
          </div>
          {wa && (
            <a
              href={whatsappLink(s.user.phone, `Olá ${s.user.name.split(" ")[0]}! Sentimos sua falta nos treinos da Unika! Está tudo bem com você? Como podemos te ajudar na sua rotina? 💪`)}
              target="_blank"
              rel="noreferrer"
              className="btn-dark !py-1.5 !px-3 text-xs shrink-0 text-amber-300 hover:text-white"
            >
              Reengajar
            </a>
          )}
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {wa && <a href={wa} target="_blank" rel="noreferrer" className="btn-dark">WhatsApp</a>}
        {can("students.manage") && !edit && <button className="btn-dark" onClick={startEdit}>Editar</button>}
        {can("students.manage") && <button className="btn-dark" disabled={busy} onClick={reset}>Nova senha</button>}
        {can("students.manage") && s.status === "CANCELADO" && <button className="btn-brand" disabled={busy} onClick={() => setStatus("ATIVO")}>Reativar matrícula</button>}
        {can("students.manage") && s.status !== "CANCELADO" && <button className="btn bg-[#FF5A36]/15 text-[#FF7A5C] px-5 py-3" onClick={() => setCancel(true)}>Cancelar matrícula</button>}
        {can("students.manage") && s.status === "INADIMPLENTE" && <button className="btn-dark" disabled={busy} onClick={() => setStatus("ATIVO")}>Liberar acesso</button>}
      </div>

      {creds && <Credentials login={creds.login} password={creds.pass} />}

      {edit && (
        <div className="card-soft p-4 space-y-3">
          <Field label="Nome"><input className="field" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <div className="grid sm:grid-cols-2 gap-3">
            <Field label="E-mail"><input className="field" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
            <Field label="Telefone"><input className="field" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
            <Field label="Plano" hint="Vale para as próximas mensalidades">
              <select className="field" value={form.plan_id} onChange={(e) => setForm({ ...form, plan_id: e.target.value })}>
                <option value="">{s.plan_name || "Sem plano"} (atual)</option>
                {plans.data?.map(({ plan: p }) => <option key={p.id} value={p.id}>{p.name} — {brl(p.price)}</option>)}
              </select>
            </Field>
            <Field label="Dia do vencimento"><NumberInput value={form.due_day} min={1} max={31} onChange={(n) => setForm({ ...form, due_day: n })} /></Field>
          </div>
          <Field label="Observações"><textarea className="field" rows={3} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
          <div className="flex justify-end gap-2"><button className="btn-ghost" onClick={() => setEdit(false)}>Cancelar</button><button className="btn-brand" disabled={busy} onClick={save}>Salvar</button></div>
        </div>
      )}

      <div className="grid grid-cols-3 gap-3 text-center">
        <div className="card-soft p-3"><div className="text-xl font-semibold text-white">{data.checkins_last30}</div><div className="text-[11px] text-ink-300">entradas em 30 dias</div></div>
        <div className="card-soft p-3"><div className="text-xl font-semibold text-white">{s.due_day}</div><div className="text-[11px] text-ink-300">dia do vencimento</div></div>
        <div className="card-soft p-3"><div className="text-xl font-semibold text-white">{dateBR(s.created_at)}</div><div className="text-[11px] text-ink-300">matrícula</div></div>
      </div>

      <section>
        <SectionHeader title="Mensalidades" />
        {data.invoices.length === 0 ? <p className="text-xs text-ink-400">Nenhuma cobrança ainda.</p> : (
          <ul className="space-y-2">
            {data.invoices.map((inv) => {
              const t = INVOICE_STATUS[inv.status];
              return (
                <li key={inv.id} className="card-soft p-3 flex items-center gap-3 text-sm">
                  <div className="flex-1 min-w-0"><div className="text-white truncate">{inv.description}</div><div className="text-[11px] text-ink-300">vence {dateBR(inv.due_date)}{inv.paid_at ? ` · pago ${dateBR(inv.paid_at)}` : ""}</div></div>
                  <div className="text-white font-medium">{brl(inv.amount - inv.discount)}</div>
                  <Tag tone={t.tone}>{t.label}</Tag>
                  {can("finance.manage") && (inv.status === "PENDENTE" || inv.status === "VENCIDO") && (
                    <button className="btn-brand px-3 py-2 text-xs" onClick={() => setPay({ ...inv, student: s })}>Receber</button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section>
        <SectionHeader title="Últimas aulas reservadas" />
        {data.bookings.length === 0 ? <p className="text-xs text-ink-400">Sem reservas.</p> : (
          <ul className="space-y-1.5 text-sm">
            {data.bookings.map((b) => (
              <li key={b.id} className="flex items-center justify-between gap-3">
                <span className="text-ink-100 truncate">{b.session?.title || b.session?.modality?.name || "Aula"} · {dateTimeBR(b.session?.starts_at)}</span>
                <Tag tone={b.status === "PRESENTE" ? "ok" : b.status === "FALTA" ? "bad" : "mute"}>{b.status.toLowerCase()}</Tag>
              </li>
            ))}
          </ul>
        )}
      </section>

      {data.sales.length > 0 && (
        <section>
          <SectionHeader title="Vendas deste aluno" />
          <ul className="space-y-1.5 text-sm">
            {data.sales.map((v) => <li key={v.id} className="flex justify-between gap-3"><span className="text-ink-100 truncate">{v.description} · {v.seller?.name}</span><span className="text-white">{brl(v.amount)}</span></li>)}
          </ul>
        </section>
      )}

      {cancel && (
        <ConfirmDialog title="Cancelar matrícula" danger confirmLabel="Cancelar matrícula" askReason="Motivo do cancelamento (aparece nos relatórios)" reasonRequired busy={busy}
          message="As reservas futuras serão liberadas e as cobranças em aberto a vencer serão canceladas. Dá para reativar depois."
          onClose={() => setCancel(false)} onConfirm={(reason) => setStatus("CANCELADO", reason)} />
      )}
      {pay && <PayModal invoice={pay} onClose={() => setPay(null)} onDone={refresh} />}
    </Drawer>
  );
}
