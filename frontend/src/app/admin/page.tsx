"use client";

import React, { useEffect, useState, useRef } from "react";
import { useRouter } from "next/navigation";
import { 
  Users, DollarSign, AlertCircle, CheckCircle, UploadCloud, 
  Search, Plus, FileText, CheckCircle2, RefreshCw, Eye, ArrowUpRight, 
  Clock, ShieldAlert, Sparkles, Filter, ChevronRight
} from "lucide-react";
import { api } from "@/services/api";
import { DashboardStats, ImportPreviewResult, Student } from "@/types";

export default function AdminDashboardPage() {
  const router = useRouter();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [activeTab, setActiveTab] = useState<"overview" | "importer" | "students" | "finance">("overview");
  const [loading, setLoading] = useState(true);

  // Alunos
  const [students, setStudents] = useState<Student[]>([]);
  const [studentSearch, setStudentSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [studentPage, setStudentPage] = useState(1);
  const [studentTotal, setStudentTotal] = useState(0);

  // Importador
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [previewResult, setPreviewResult] = useState<ImportPreviewResult | null>(null);
  const [importLoading, setImportLoading] = useState(false);
  const [importSuccessMessage, setImportSuccessMessage] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | null>(null);

  // Novo Aluno Manual
  const [showNewStudentModal, setShowNewStudentModal] = useState(false);
  const [newStudent, setNewStudent] = useState({
    name: "",
    document: "",
    email: "",
    phone: "",
    plan_name: "Plano Mensal",
    plan_price: 119.90,
    due_day: 10,
  });

  const loadData = async () => {
    try {
      setLoading(true);
      const [dashStats, studentsRes] = await Promise.all([
        api.getDashboardStats(),
        api.listStudents(studentSearch, statusFilter, studentPage, 15),
      ]);
      setStats(dashStats);
      setStudents(studentsRes.data);
      setStudentTotal(studentsRes.total);
    } catch (err: any) {
      console.error(err);
      if (err.message?.includes("401") || err.message?.includes("Acesso negado")) {
        router.push("/");
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [studentSearch, statusFilter, studentPage]);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files || e.target.files.length === 0) return;
    const file = e.target.files[0];
    setImportFile(file);
    setImportError(null);
    setImportSuccessMessage(null);
    setImportLoading(true);

    try {
      const preview = await api.uploadImportPreview(file);
      setPreviewResult(preview);
    } catch (err: any) {
      setImportError(err.message || "Erro ao analisar o arquivo.");
    } finally {
      setImportLoading(false);
    }
  };

  const loadSampleCSV = async () => {
    setImportLoading(true);
    setImportError(null);
    try {
      const sampleText = `Nome Completo;CPF / Documento;E-mail;Telefone / WhatsApp;Plano;Mensalidade;Dia Vencimento;Status
Lucas Gabriel Oliveira;456.789.123-01;lucas.oliveira@email.com;(11) 98765-1122;Plano Mensal;R$ 119,90;10;ATIVO
Beatriz Santos Souza;789.123.456-02;beatriz.souza@email.com;(11) 97654-2233;Plano Anual;R$ 89,90;15;ATIVO
Rafael Mendes Costa;321.654.987-03;rafael.costa@email.com;(11) 96543-3344;Trimestral;R$ 99,90;05;INADIMPLENTE
Juliana Ferreira Lima;654.987.321-04;juliana.lima@email.com;(11) 95432-4455;Semestral;R$ 94,90;20;ATIVO
Thiago Rocha Silveira;147.258.369-05;thiago.silveira@email.com;(11) 94321-5566;Plano Black;R$ 139,90;10;INADIMPLENTE
Camila Duarte Barbosa;258.369.147-06;camila.barbosa@email.com;(11) 93210-6677;Plano Mensal;R$ 119,90;25;ATIVO
Gabriel Henrique Pinto;369.147.258-07;gabriel.pinto@email.com;(11) 92109-7788;Plano Anual;R$ 89,90;10;ATIVO
Mariana Alcantara Cruz;951.753.852-08;mariana.cruz@email.com;(11) 91098-8899;Plano Mensal;R$ 119,90;05;ATIVO`;

      const blob = new Blob([sampleText], { type: "text/csv" });
      const file = new File([blob], "alunos_exemplo.csv", { type: "text/csv" });
      setImportFile(file);
      const preview = await api.uploadImportPreview(file);
      setPreviewResult(preview);
    } catch (err: any) {
      setImportError(err.message || "Erro ao carregar exemplo");
    } finally {
      setImportLoading(false);
    }
  };

  const handleExecuteImport = async () => {
    if (!previewResult || previewResult.rows.length === 0) return;
    setImportLoading(true);
    setImportError(null);
    try {
      const summary = await api.executeImport(previewResult.rows);
      setImportSuccessMessage(
        `Sucesso! ${summary.created_count} novos alunos cadastrados e ${summary.updated_count} atualizados.`
      );
      setPreviewResult(null);
      setImportFile(null);
      loadData();
    } catch (err: any) {
      setImportError(err.message || "Falha ao gravar alunos no banco.");
    } finally {
      setImportLoading(false);
    }
  };

  const handleCreateStudent = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.createStudent(newStudent);
      setShowNewStudentModal(false);
      setNewStudent({
        name: "",
        document: "",
        email: "",
        phone: "",
        plan_name: "Plano Mensal",
        plan_price: 119.90,
        due_day: 10,
      });
      loadData();
    } catch (err: any) {
      alert("Erro ao cadastrar aluno: " + err.message);
    }
  };

  return (
    <div className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      {/* Cabeçalho */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold text-white flex items-center gap-2">
            Painel do Gestor <span className="text-xs px-2 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30">Admin</span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Visão financeira, frequência e gestão centralizada de alunos da academia.
          </p>
        </div>

        {/* Abas */}
        <div className="flex items-center gap-1 bg-slate-900 border border-slate-800 p-1 rounded-xl">
          <button
            onClick={() => setActiveTab("overview")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors ${activeTab === "overview" ? "bg-blue-600 text-white" : "text-slate-400 hover:text-white"}`}
          >
            Visão Geral
          </button>
          <button
            onClick={() => setActiveTab("importer")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors flex items-center gap-1.5 ${activeTab === "importer" ? "bg-blue-600 text-white" : "text-slate-400 hover:text-white"}`}
          >
            <UploadCloud className="w-3.5 h-3.5" /> Importar Planilha
          </button>
          <button
            onClick={() => setActiveTab("students")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors ${activeTab === "students" ? "bg-blue-600 text-white" : "text-slate-400 hover:text-white"}`}
          >
            Alunos ({studentTotal})
          </button>
          <button
            onClick={() => setActiveTab("finance")}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors ${activeTab === "finance" ? "bg-blue-600 text-white" : "text-slate-400 hover:text-white"}`}
          >
            Inadimplência
          </button>
        </div>
      </div>

      {/* Cards de Métricas Principais */}
      {stats && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
              <span>Alunos Ativos</span>
              <Users className="w-4 h-4 text-blue-400" />
            </div>
            <div className="text-2xl font-bold text-white">{stats.active_students}</div>
            <div className="text-[11px] text-slate-500">De {stats.total_students} matrículas cadastradas</div>
          </div>

          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
              <span>Faturamento Mensal (MRR)</span>
              <DollarSign className="w-4 h-4 text-emerald-400" />
            </div>
            <div className="text-2xl font-bold text-emerald-400">
              R$ {stats.monthly_revenue.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
            </div>
            <div className="text-[11px] text-emerald-500/80">Recorrência mensal estimada</div>
          </div>

          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
              <span>Inadimplentes</span>
              <AlertCircle className="w-4 h-4 text-red-400" />
            </div>
            <div className="text-2xl font-bold text-red-400">{stats.delinquent_students}</div>
            <div className="text-[11px] text-red-400/80">Acesso bloqueado na catraca</div>
          </div>

          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-2">
            <div className="flex items-center justify-between text-slate-400 text-xs font-medium">
              <span>Presenças Hoje</span>
              <CheckCircle className="w-4 h-4 text-indigo-400" />
            </div>
            <div className="text-2xl font-bold text-indigo-400">{stats.today_checkins}</div>
            <div className="text-[11px] text-slate-500">Check-ins efetuados hoje</div>
          </div>
        </div>
      )}

      {/* ABA: VISÃO GERAL */}
      {activeTab === "overview" && stats && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Presenças Recentes */}
          <div className="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="font-bold text-white text-base flex items-center gap-2">
                <Clock className="w-4 h-4 text-indigo-400" /> Presenças e Frequência Recentes
              </h3>
              <span className="text-xs text-slate-500">Tempo real</span>
            </div>

            {stats.recent_checkins && stats.recent_checkins.length > 0 ? (
              <div className="divide-y divide-slate-800">
                {stats.recent_checkins.map((chk) => (
                  <div key={chk.id} className="py-3 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-slate-800 flex items-center justify-center text-xs font-bold text-indigo-400">
                        {chk.student?.user?.name?.slice(0, 2).toUpperCase() || "AL"}
                      </div>
                      <div>
                        <div className="text-sm font-semibold text-white">{chk.student?.user?.name}</div>
                        <div className="text-xs text-slate-500">
                          {chk.student?.plan_name} • CPF: {chk.student?.user?.document || "N/A"}
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <span className="text-xs text-indigo-400 font-mono">
                        {new Date(chk.created_at).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" })}
                      </span>
                      <div className="text-[10px] text-slate-500">Hoje</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-8 text-center text-slate-500 text-sm">
                Nenhum check-in registrado hoje ainda.
              </div>
            )}
          </div>

          {/* Destaque Inadimplência / Ações Rápidas */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
            <h3 className="font-bold text-white text-base flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-red-400" /> Cobranças Pendentes
            </h3>

            {stats.pending_invoices && stats.pending_invoices.length > 0 ? (
              <div className="space-y-3">
                {stats.pending_invoices.slice(0, 4).map((inv) => (
                  <div key={inv.id} className="p-3 rounded-xl bg-slate-800/60 border border-slate-700/60 flex items-center justify-between">
                    <div>
                      <div className="text-xs font-semibold text-white">{inv.student?.user?.name}</div>
                      <div className="text-[11px] text-slate-400">
                        Vence: {new Date(inv.due_date).toLocaleDateString("pt-BR")}
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-xs font-bold text-amber-400">
                        R$ {inv.amount.toFixed(2)}
                      </div>
                      <a
                        href={api.getBoletoUrl(inv.id)}
                        target="_blank"
                        className="text-[10px] text-blue-400 hover:underline flex items-center gap-0.5 justify-end"
                      >
                        Boleto <ArrowUpRight className="w-2.5 h-2.5" />
                      </a>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-8 text-center text-slate-500 text-sm">
                Nenhuma mensalidade em atraso!
              </div>
            )}

            <div className="pt-2 border-t border-slate-800">
              <button
                onClick={() => setActiveTab("importer")}
                className="w-full py-2 px-3 rounded-xl bg-blue-600/20 hover:bg-blue-600/30 text-blue-400 border border-blue-500/30 text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
              >
                <UploadCloud className="w-4 h-4" /> Importar Novos Alunos em Massa
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ABA: IMPORTADOR INTELIGENTE DE ALUNOS */}
      {activeTab === "importer" && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h2 className="text-lg font-bold text-white flex items-center gap-2">
                <UploadCloud className="w-5 h-5 text-blue-400" /> Importador Inteligente de Alunos
              </h2>
              <p className="text-xs text-slate-400 mt-1">
                Suba planilhas em <b>.xlsx</b> ou <b>.csv</b>. O sistema detecta automaticamente os cabeçalhos mesmo com nomes diferentes.
              </p>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={loadSampleCSV}
                disabled={importLoading}
                className="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-medium text-slate-300 border border-slate-700 transition-colors flex items-center gap-1.5"
              >
                <FileText className="w-3.5 h-3.5 text-blue-400" /> Carregar Planilha de Exemplo
              </button>
              <button
                onClick={() => fileInputRef.current?.click()}
                disabled={importLoading}
                className="px-4 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-xs font-medium text-white shadow-lg shadow-blue-600/20 transition-colors flex items-center gap-1.5"
              >
                <UploadCloud className="w-3.5 h-3.5" /> Selecionar Arquivo do PC
              </button>
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv, .xlsx, .xls"
                onChange={handleFileSelect}
                className="hidden"
              />
            </div>
          </div>

          {importError && (
            <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{importError}</span>
            </div>
          )}

          {importSuccessMessage && (
            <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>{importSuccessMessage}</span>
            </div>
          )}

          {/* Área de Preview da Importação */}
          {previewResult && (
            <div className="space-y-4 pt-2 border-t border-slate-800">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 bg-slate-800/40 p-4 rounded-xl border border-slate-700/60">
                <div>
                  <div className="text-sm font-bold text-white flex items-center gap-2">
                    <Sparkles className="w-4 h-4 text-amber-400" /> Validação Prévia (Dry Run)
                  </div>
                  <div className="text-xs text-slate-400 mt-0.5">
                    Total: <b>{previewResult.total_rows}</b> | Válidos: <b className="text-emerald-400">{previewResult.valid_rows}</b> | Com aviso: <b className="text-amber-400">{previewResult.invalid_rows}</b>
                  </div>
                </div>

                <button
                  onClick={handleExecuteImport}
                  disabled={importLoading || previewResult.valid_rows === 0}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-colors shadow-lg shadow-emerald-600/30 flex items-center gap-2 disabled:opacity-50"
                >
                  <CheckCircle2 className="w-4 h-4" />
                  {importLoading ? "Gravando no Banco..." : `Confirmar e Importar ${previewResult.valid_rows} Alunos`}
                </button>
              </div>

              {/* Mapeamento de colunas detectadas */}
              <div className="flex flex-wrap items-center gap-2 text-xs">
                <span className="text-slate-400">Colunas mapeadas:</span>
                {Object.entries(previewResult.detected_fields).map(([header, field]) => (
                  <span key={header} className="px-2 py-0.5 rounded bg-blue-950/60 border border-blue-800/50 text-blue-300">
                    "{header}" → <b>{field}</b>
                  </span>
                ))}
              </div>

              {/* Tabela de Preview */}
              <div className="overflow-x-auto rounded-xl border border-slate-800">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-800/80 text-slate-300 uppercase text-[10px] tracking-wider">
                    <tr>
                      <th className="p-3">#</th>
                      <th className="p-3">Nome</th>
                      <th className="p-3">CPF</th>
                      <th className="p-3">E-mail</th>
                      <th className="p-3">Telefone</th>
                      <th className="p-3">Plano</th>
                      <th className="p-3">Valor</th>
                      <th className="p-3">Vencimento</th>
                      <th className="p-3">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800 bg-slate-900/50">
                    {previewResult.rows.map((row) => (
                      <tr key={row.row_index} className="hover:bg-slate-800/40">
                        <td className="p-3 text-slate-500">{row.row_index}</td>
                        <td className="p-3 font-semibold text-white">{row.name}</td>
                        <td className="p-3 text-slate-300 font-mono">{row.document || "—"}</td>
                        <td className="p-3 text-slate-300">{row.email}</td>
                        <td className="p-3 text-slate-300">{row.phone || "—"}</td>
                        <td className="p-3 text-blue-400">{row.plan_name}</td>
                        <td className="p-3 text-emerald-400 font-medium">R$ {row.plan_price.toFixed(2)}</td>
                        <td className="p-3 text-slate-300">Dia {row.due_day}</td>
                        <td className="p-3">
                          <span className={`px-2 py-0.5 rounded-full text-[10px] font-semibold ${row.status === "ATIVO" ? "bg-emerald-500/20 text-emerald-400" : "bg-red-500/20 text-red-400"}`}>
                            {row.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ABA: GESTÃO DE ALUNOS */}
      {activeTab === "students" && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex-1 flex flex-col sm:flex-row gap-3">
              <div className="relative flex-1">
                <Search className="w-4 h-4 text-slate-500 absolute left-3.5 top-3" />
                <input
                  type="text"
                  placeholder="Buscar por nome, CPF ou telefone..."
                  value={studentSearch}
                  onChange={(e) => {
                    setStudentSearch(e.target.value);
                    setStudentPage(1);
                  }}
                  className="w-full pl-10 pr-4 py-2 bg-slate-800/80 border border-slate-700 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <select
                value={statusFilter}
                onChange={(e) => {
                  setStatusFilter(e.target.value);
                  setStudentPage(1);
                }}
                className="px-3 py-2 bg-slate-800/80 border border-slate-700 rounded-xl text-xs text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Todos os Status</option>
                <option value="ATIVO">Ativos</option>
                <option value="INADIMPLENTE">Inadimplentes</option>
              </select>
            </div>

            <button
              onClick={() => setShowNewStudentModal(true)}
              className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-xs font-semibold text-white shadow-lg shadow-blue-600/20 transition-colors flex items-center gap-1.5 shrink-0"
            >
              <Plus className="w-4 h-4" /> Cadastrar Aluno Individual
            </button>
          </div>

          {/* Tabela de Alunos */}
          <div className="overflow-x-auto rounded-xl border border-slate-800">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-800/80 text-slate-300 uppercase text-[10px] tracking-wider">
                <tr>
                  <th className="p-3">Aluno</th>
                  <th className="p-3">CPF</th>
                  <th className="p-3">Telefone</th>
                  <th className="p-3">Plano</th>
                  <th className="p-3">Mensalidade</th>
                  <th className="p-3">Status</th>
                  <th className="p-3 text-right">Ação</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800 bg-slate-900/50">
                {students.map((std) => (
                  <tr key={std.id} className="hover:bg-slate-800/40">
                    <td className="p-3">
                      <div className="font-semibold text-white">{std.user?.name}</div>
                      <div className="text-[11px] text-slate-400">{std.user?.email}</div>
                    </td>
                    <td className="p-3 font-mono text-slate-300">{std.user?.document || "—"}</td>
                    <td className="p-3 text-slate-300">{std.user?.phone || "—"}</td>
                    <td className="p-3 text-blue-400">{std.plan_name}</td>
                    <td className="p-3 text-emerald-400 font-medium">
                      R$ {std.plan_price.toFixed(2)} (dia {std.due_day})
                    </td>
                    <td className="p-3">
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-semibold ${std.status === "ATIVO" ? "bg-emerald-500/20 text-emerald-400" : "bg-red-500/20 text-red-400"}`}>
                        {std.status}
                      </span>
                    </td>
                    <td className="p-3 text-right">
                      <button
                        onClick={() => router.push(`/professor?studentId=${std.id}`)}
                        className="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-blue-400 text-[11px] font-medium transition-colors"
                      >
                        Ver Treino
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ABA: INADIMPLÊNCIA / FINANCEIRO */}
      {activeTab === "finance" && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-bold text-white flex items-center gap-2">
                <AlertCircle className="w-5 h-5 text-red-400" /> Gestão de Inadimplência & Boletos
              </h2>
              <p className="text-xs text-slate-400 mt-1">
                Alunos com faturas atrasadas têm bloqueio automático do check-in no aplicativo.
              </p>
            </div>
          </div>

          <div className="overflow-x-auto rounded-xl border border-slate-800">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-800/80 text-slate-300 uppercase text-[10px] tracking-wider">
                <tr>
                  <th className="p-3">Aluno</th>
                  <th className="p-3">Valor</th>
                  <th className="p-3">Vencimento</th>
                  <th className="p-3">Linha Digitável</th>
                  <th className="p-3">Status</th>
                  <th className="p-3 text-right">Ação</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800 bg-slate-900/50">
                {stats?.pending_invoices?.map((inv) => (
                  <tr key={inv.id} className="hover:bg-slate-800/40">
                    <td className="p-3">
                      <div className="font-semibold text-white">{inv.student?.user?.name}</div>
                      <div className="text-[11px] text-slate-400">CPF: {inv.student?.user?.document}</div>
                    </td>
                    <td className="p-3 text-emerald-400 font-bold">R$ {inv.amount.toFixed(2)}</td>
                    <td className="p-3 text-amber-400">
                      {new Date(inv.due_date).toLocaleDateString("pt-BR")}
                    </td>
                    <td className="p-3 font-mono text-slate-400 text-[11px]">
                      {inv.boleto_digits ? inv.boleto_digits.slice(0, 24) + "..." : "—"}
                    </td>
                    <td className="p-3">
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-red-500/20 text-red-400">
                        {inv.status}
                      </span>
                    </td>
                    <td className="p-3 text-right">
                      <a
                        href={api.getBoletoUrl(inv.id)}
                        target="_blank"
                        className="px-2.5 py-1 rounded-lg bg-blue-600/20 text-blue-400 hover:bg-blue-600/30 text-[11px] font-medium transition-colors inline-flex items-center gap-1"
                      >
                        Boleto PDF <ArrowUpRight className="w-3 h-3" />
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Modal de Cadastro Manual */}
      {showNewStudentModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-bold text-white">Cadastrar Aluno Individual</h3>

            <form onSubmit={handleCreateStudent} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">Nome Completo</label>
                <input
                  type="text"
                  required
                  value={newStudent.name}
                  onChange={(e) => setNewStudent({ ...newStudent, name: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1">CPF</label>
                  <input
                    type="text"
                    value={newStudent.document}
                    onChange={(e) => setNewStudent({ ...newStudent, document: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1">Telefone/WhatsApp</label>
                  <input
                    type="text"
                    value={newStudent.phone}
                    onChange={(e) => setNewStudent({ ...newStudent, phone: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">E-mail</label>
                <input
                  type="email"
                  value={newStudent.email}
                  onChange={(e) => setNewStudent({ ...newStudent, email: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                />
              </div>

              <div className="grid grid-cols-3 gap-2">
                <div className="col-span-2">
                  <label className="block text-xs font-medium text-slate-300 mb-1">Plano</label>
                  <input
                    type="text"
                    value={newStudent.plan_name}
                    onChange={(e) => setNewStudent({ ...newStudent, plan_name: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1">Valor (R$)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={newStudent.plan_price}
                    onChange={(e) => setNewStudent({ ...newStudent, plan_price: parseFloat(e.target.value) || 0 })}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-800">
                <button
                  type="button"
                  onClick={() => setShowNewStudentModal(false)}
                  className="px-4 py-2 rounded-xl text-xs text-slate-400 hover:text-white"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-xs font-semibold text-white shadow-lg shadow-blue-600/30"
                >
                  Salvar Aluno
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
