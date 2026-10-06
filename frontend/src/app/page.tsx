"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { Dumbbell, ShieldCheck, UserCheck, ArrowRight, Sparkles, CheckCircle2, FileSpreadsheet, CreditCard, Clock } from "lucide-react";
import { api } from "@/services/api";

export default function HomePage() {
  const router = useRouter();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async (e?: React.FormEvent, customUser?: { ident: string; pass: string }) => {
    if (e) e.preventDefault();
    setError(null);
    setLoading(true);

    const loginIdent = customUser ? customUser.ident : identifier;
    const loginPass = customUser ? customUser.pass : password;

    try {
      const res = await api.login(loginIdent, loginPass);
      if (res.user.role === "ADMIN") {
        router.push("/admin");
      } else if (res.user.role === "PROFESSOR") {
        router.push("/professor");
      } else {
        router.push("/aluno");
      }
    } catch (err: any) {
      setError(err.message || "Falha no login. Verifique os dados.");
    } finally {
      setLoading(false);
    }
  };

  const demoLogin = (ident: string, pass: string) => {
    setIdentifier(ident);
    setPassword(pass);
    handleLogin(undefined, { ident, pass });
  };

  return (
    <div className="flex-1 flex flex-col justify-center items-center px-4 py-12 bg-gradient-to-b from-slate-950 via-slate-900 to-slate-950">
      <div className="max-w-4xl w-full text-center space-y-4 mb-10">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-500/10 border border-blue-500/20 text-blue-400 text-xs font-semibold">
          <Sparkles className="w-3.5 h-3.5" /> Plataforma SaaS Integrada para Academias
        </div>
        <h1 className="text-4xl sm:text-5xl font-extrabold tracking-tight text-white">
          Gestão completa, prescrição rápida e a melhor experiência para seu aluno.
        </h1>
        <p className="text-base sm:text-lg text-slate-400 max-w-2xl mx-auto">
          Conecte a academia, o professor e o aluno em um ecossistema com importação inteligente de planilhas, cobrança simulada e mais de 110 exercícios.
        </p>
      </div>

      <div className="max-w-4xl w-full grid grid-cols-1 md:grid-cols-2 gap-8 items-start">
        {/* Card de Acesso Rápido de Testes */}
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur-sm space-y-4">
          <div className="flex items-center justify-between border-b border-slate-800 pb-3">
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-amber-400" /> Acesso Rápido (Demonstração)
            </h2>
            <span className="text-xs text-slate-500">Clique para testar</span>
          </div>

          <div className="space-y-3">
            {/* Gestor */}
            <button
              onClick={() => demoLogin("admin@gymsync.com", "123456")}
              disabled={loading}
              className="w-full text-left p-4 rounded-xl bg-slate-800/60 hover:bg-slate-800 border border-slate-700/60 hover:border-blue-500/50 transition-all flex items-center justify-between group"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-blue-500/20 text-blue-400 flex items-center justify-center">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <div>
                  <div className="font-semibold text-white text-sm group-hover:text-blue-400 transition-colors">
                    1. Carlos Gestor (Academia)
                  </div>
                  <div className="text-xs text-slate-400">
                    Dashboard, Importador Excel/CSV, Inadimplência
                  </div>
                </div>
              </div>
              <ArrowRight className="w-4 h-4 text-slate-500 group-hover:text-blue-400 transition-colors" />
            </button>

            {/* Professor */}
            <button
              onClick={() => demoLogin("professor@gymsync.com", "123456")}
              disabled={loading}
              className="w-full text-left p-4 rounded-xl bg-slate-800/60 hover:bg-slate-800 border border-slate-700/60 hover:border-emerald-500/50 transition-all flex items-center justify-between group"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-emerald-500/20 text-emerald-400 flex items-center justify-center">
                  <UserCheck className="w-5 h-5" />
                </div>
                <div>
                  <div className="font-semibold text-white text-sm group-hover:text-emerald-400 transition-colors">
                    2. Lucas Treinador (Professor)
                  </div>
                  <div className="text-xs text-slate-400">
                    Montador de treinos, 110+ exercícios, alunos no salão
                  </div>
                </div>
              </div>
              <ArrowRight className="w-4 h-4 text-slate-500 group-hover:text-emerald-400 transition-colors" />
            </button>

            {/* Aluno */}
            <button
              onClick={() => demoLogin("aluno@gymsync.com", "123456")}
              disabled={loading}
              className="w-full text-left p-4 rounded-xl bg-slate-800/60 hover:bg-slate-800 border border-slate-700/60 hover:border-amber-500/50 transition-all flex items-center justify-between group"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-amber-500/20 text-amber-400 flex items-center justify-center">
                  <Dumbbell className="w-5 h-5" />
                </div>
                <div>
                  <div className="font-semibold text-white text-sm group-hover:text-amber-400 transition-colors">
                    3. Matheus Aluno (Cliente)
                  </div>
                  <div className="text-xs text-slate-400">
                    Ficha interativa, timer de descanso, boletos e PIX
                  </div>
                </div>
              </div>
              <ArrowRight className="w-4 h-4 text-slate-500 group-hover:text-amber-400 transition-colors" />
            </button>
          </div>

          <div className="pt-2 text-xs text-slate-500 flex items-center justify-between">
            <span>Senha padrão de teste: 123456</span>
            <span>Ambiente seguro</span>
          </div>
        </div>

        {/* Formulário de Login Manual */}
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-6 shadow-xl backdrop-blur-sm">
          <h2 className="text-lg font-bold text-white mb-2">Entrar com suas Credenciais</h2>
          <p className="text-xs text-slate-400 mb-6">
            Alunos importados entram com o <b>CPF</b> ou E-mail.
          </p>

          {error && (
            <div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs">
              {error}
            </div>
          )}

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                E-mail ou CPF
              </label>
              <input
                type="text"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                placeholder="ex: admin@gymsync.com ou 123.456.789-00"
                required
                className="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Senha
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                className="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-medium text-sm transition-colors shadow-lg shadow-blue-600/30 disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {loading ? "Entrando..." : "Acessar Plataforma"}
              <ArrowRight className="w-4 h-4" />
            </button>
          </form>

          <div className="mt-6 pt-4 border-t border-slate-800 grid grid-cols-3 gap-2 text-center text-[11px] text-slate-400">
            <div className="flex flex-col items-center gap-1">
              <FileSpreadsheet className="w-4 h-4 text-blue-400" />
              <span>Importação XLSX</span>
            </div>
            <div className="flex flex-col items-center gap-1">
              <CreditCard className="w-4 h-4 text-emerald-400" />
              <span>Boleto & PIX</span>
            </div>
            <div className="flex flex-col items-center gap-1">
              <Clock className="w-4 h-4 text-amber-400" />
              <span>Timer Treino</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
