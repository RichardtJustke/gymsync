"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  Dumbbell, ShieldCheck, UserCheck, ArrowRight, Download, Smartphone,
  CheckCircle2, Sparkles, Lock, ArrowUpRight, ChevronRight, HelpCircle,
  ExternalLink, Layers, Eye, Laptop
} from "lucide-react";
import { api, homeFor, loadSession } from "@/services/api";

type RoleMode = "aluno" | "professor" | "gestor";

export default function HomePage() {
  const router = useRouter();
  const [roleMode, setRoleMode] = useState<RoleMode>("aluno");
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Detecção e gatilho de instalação PWA
  const [deferredPrompt, setDeferredPrompt] = useState<any>(null);
  const [isStandalone, setIsStandalone] = useState(false);
  const [showIosTip, setShowIosTip] = useState(false);
  const [viewMobileFrame, setViewMobileFrame] = useState(true);

  // Redirecionamento automático se já estiver logado
  useEffect(() => {
    const session = loadSession();
    if (session?.user) {
      router.replace(homeFor(session.user.role));
    }
  }, [router]);

  // Captura do evento de instalação PWA
  useEffect(() => {
    const handleBeforeInstall = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e);
    };

    window.addEventListener("beforeinstallprompt", handleBeforeInstall);

    if (
      window.matchMedia("(display-mode: standalone)").matches ||
      (window.navigator as any).standalone === true
    ) {
      setIsStandalone(true);
    }

    return () => {
      window.removeEventListener("beforeinstallprompt", handleBeforeInstall);
    };
  }, []);

  const handleInstallPWA = async () => {
    if (deferredPrompt) {
      deferredPrompt.prompt();
      const { outcome } = await deferredPrompt.userChoice;
      if (outcome === "accepted") {
        setDeferredPrompt(null);
      }
    } else {
      setShowIosTip(true);
    }
  };

  const handleLogin = async (e?: React.FormEvent, customUser?: { ident: string; pass: string }) => {
    if (e) e.preventDefault();
    setError(null);
    setLoading(true);

    const loginIdent = customUser ? customUser.ident : identifier;
    const loginPass = customUser ? customUser.pass : password;

    try {
      const res = await api.login(loginIdent, loginPass);
      router.push(homeFor(res.user.role));
    } catch (err: any) {
      setError(err.message || "Falha no login. Verifique seus dados.");
    } finally {
      setLoading(false);
    }
  };

  const demoLogin = (ident: string, pass: string, targetRole: RoleMode) => {
    setRoleMode(targetRole);
    setIdentifier(ident);
    setPassword(pass);
    handleLogin(undefined, { ident, pass });
  };

  return (
    <div className="min-h-screen bg-ink-950 flex flex-col justify-between p-3 sm:p-6 lg:p-8 select-none">
      {/* Barra superior de utilidades para demonstração e preview */}
      <header className="max-w-md md:max-w-3xl mx-auto w-full flex items-center justify-between pb-3 text-xs text-ink-300">
        <div className="flex items-center gap-2">
          <div className="w-2.5 h-2.5 rounded-full bg-lime animate-pulse" />
          <span className="font-semibold text-white">Unika GymSync</span>
          <span className="hidden sm:inline text-ink-400">· 3 Unidades Online</span>
        </div>

        <div className="flex items-center gap-2">
          {/* Botão de alternar moldura mobile no desktop */}
          <button
            onClick={() => setViewMobileFrame(!viewMobileFrame)}
            className="hidden md:inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-ink-850 hover:bg-ink-800 text-ink-200 border border-white/5 transition-colors"
          >
            {viewMobileFrame ? <Laptop className="w-3.5 h-3.5" /> : <Smartphone className="w-3.5 h-3.5" />}
            <span>{viewMobileFrame ? "Expandir Tela" : "Modo Celular"}</span>
          </button>

          {!isStandalone && (
            <button
              onClick={handleInstallPWA}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-lime/10 text-lime hover:bg-lime/20 border border-lime/30 font-semibold transition-all"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Instalar App (PWA)</span>
            </button>
          )}
        </div>
      </header>

      {/* Container Principal — Seletor de Modo Mobile / Tablet */}
      <main className="flex-1 flex items-center justify-center py-2">
        <div
          className={`w-full transition-all duration-300 ${
            viewMobileFrame
              ? "max-w-[430px] rounded-[42px] border border-white/10 bg-ink-900/95 shadow-2xl p-5 sm:p-7 relative overflow-hidden backdrop-blur-xl"
              : "max-w-xl rounded-4xl border border-white/10 bg-ink-900/95 shadow-2xl p-6 sm:p-8 backdrop-blur-xl"
          }`}
        >
          {/* Top Notch decorativo em modo mobile */}
          {viewMobileFrame && (
            <div className="w-28 h-4 bg-ink-950 rounded-b-xl mx-auto -mt-5 mb-4 flex items-center justify-center opacity-60">
              <div className="w-10 h-1 bg-ink-800 rounded-full" />
            </div>
          )}

          {/* Cabeçalho do App */}
          <div className="text-center space-y-2 mb-6">
            <div className="w-14 h-14 rounded-2xl bg-gradient-to-tr from-lime to-lime-400 text-black flex items-center justify-center mx-auto shadow-glow">
              <Dumbbell className="w-7 h-7" strokeWidth={2.4} />
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-white">
                Gym<span className="text-lime">Sync</span>
              </h1>
              <p className="text-xs text-ink-300 mt-0.5">
                Experiência mobile para alunos e professores
              </p>
            </div>
          </div>

          {/* Dica para iOS se clicado em Instalar */}
          {showIosTip && (
            <div className="p-3.5 mb-4 rounded-2xl bg-lime/10 border border-lime/20 text-xs text-lime space-y-1 animate-fade-up">
              <div className="font-bold flex items-center justify-between">
                <span>Como instalar no iPhone (Safari):</span>
                <button onClick={() => setShowIosTip(false)} className="text-ink-300 hover:text-white">✕</button>
              </div>
              <p className="text-[11px] text-ink-200">
                1. Toque no botão de <b>Compartilhar</b> (ícone do quadrado com seta para cima).
                <br />
                2. Selecione <b>"Adicionar à Tela de Início"</b>.
              </p>
            </div>
          )}

          {/* Seletor de Perfil do App (Abas estilo App Nativo) */}
          <div className="grid grid-cols-3 p-1 rounded-2xl bg-ink-800 border border-white/[0.04] mb-5">
            <button
              onClick={() => { setRoleMode("aluno"); setError(null); }}
              className={`py-2 text-xs font-semibold rounded-xl transition-all ${
                roleMode === "aluno"
                  ? "bg-lime text-black shadow-sm"
                  : "text-ink-300 hover:text-white"
              }`}
            >
              🏋️ Aluno
            </button>
            <button
              onClick={() => { setRoleMode("professor"); setError(null); }}
              className={`py-2 text-xs font-semibold rounded-xl transition-all ${
                roleMode === "professor"
                  ? "bg-lime text-black shadow-sm"
                  : "text-ink-300 hover:text-white"
              }`}
            >
              📋 Professor
            </button>
            <button
              onClick={() => { setRoleMode("gestor"); setError(null); }}
              className={`py-2 text-xs font-semibold rounded-xl transition-all ${
                roleMode === "gestor"
                  ? "bg-lime text-black shadow-sm"
                  : "text-ink-300 hover:text-white"
              }`}
            >
              💼 Gestão
            </button>
          </div>

          {/* Banner contextual de cada perfil */}
          <div className="mb-4 text-xs">
            {roleMode === "aluno" && (
              <div className="p-3 rounded-2xl bg-ink-800/80 border border-white/5 text-ink-200 flex items-center gap-2.5">
                <Dumbbell className="w-4 h-4 text-lime shrink-0" />
                <span>Acesse sua ficha de treino, registre cargas e faça check-in por QR.</span>
              </div>
            )}
            {roleMode === "professor" && (
              <div className="p-3 rounded-2xl bg-ink-800/80 border border-white/5 text-ink-200 flex items-center gap-2.5">
                <UserCheck className="w-4 h-4 text-lime shrink-0" />
                <span>Prescreva fichas, use a Biblioteca de Modelos e controle chamadas.</span>
              </div>
            )}
            {roleMode === "gestor" && (
              <div className="p-3 rounded-2xl bg-amber-500/10 border border-amber-500/20 text-amber-300 flex items-center gap-2.5">
                <ShieldCheck className="w-4 h-4 shrink-0" />
                <span>Painel administrativo para PC / Recepção (relatórios, financeiro e equipe).</span>
              </div>
            )}
          </div>

          {error && (
            <div className="p-3 rounded-2xl bg-[#FF5A36]/15 border border-[#FF5A36]/30 text-[#FF7A5C] text-xs mb-4 animate-fade-up">
              {error}
            </div>
          )}

          {/* Formulário de Login */}
          <form onSubmit={handleLogin} className="space-y-3.5">
            <div>
              <label className="text-[11px] font-semibold text-ink-300 uppercase tracking-wider block mb-1">
                {roleMode === "aluno" ? "CPF ou E-mail cadastrado" : "E-mail de acesso"}
              </label>
              <input
                type="text"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                placeholder={roleMode === "aluno" ? "123.456.789-00 ou seu@email.com" : "seu@unika.app"}
                required
                className="field !py-3.5 text-sm"
              />
            </div>

            <div>
              <label className="text-[11px] font-semibold text-ink-300 uppercase tracking-wider block mb-1">
                Senha
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                className="field !py-3.5 text-sm"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="btn-lime w-full !py-3.5 text-sm font-bold flex items-center justify-center gap-2 shadow-glow active:scale-[0.99] transition-transform"
            >
              {loading ? "Entrando no App..." : "Entrar no GymSync"}
              <ArrowRight className="w-4 h-4" />
            </button>
          </form>

          {/* Atalhos de Demonstração Rápida com 1 Toque */}
          <div className="mt-6 pt-5 border-t border-white/5 space-y-2.5">
            <div className="flex items-center justify-between text-[11px] text-ink-400">
              <span className="font-semibold uppercase tracking-wider">Acesso Rápido de Teste</span>
              <span>senha: unika2026</span>
            </div>

            {roleMode === "aluno" && (
              <button
                onClick={() => demoLogin("aluna@unika.app", "unika2026", "aluno")}
                disabled={loading}
                className="w-full p-2.5 rounded-2xl bg-ink-800 hover:bg-ink-750 border border-white/5 hover:border-lime/30 text-left flex items-center justify-between transition-colors group"
              >
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-xl bg-lime/10 text-lime flex items-center justify-center">
                    <Dumbbell className="w-4 h-4" />
                  </div>
                  <div>
                    <div className="text-xs font-bold text-white group-hover:text-lime transition-colors">
                      Entrar como Aluna Ana
                    </div>
                    <div className="text-[10px] text-ink-300">Treino ativo, faturas PIX e check-in</div>
                  </div>
                </div>
                <ChevronRight className="w-4 h-4 text-ink-400 group-hover:text-lime" />
              </button>
            )}

            {roleMode === "professor" && (
              <button
                onClick={() => demoLogin("professora@unika.app", "unika2026", "professor")}
                disabled={loading}
                className="w-full p-2.5 rounded-2xl bg-ink-800 hover:bg-ink-750 border border-white/5 hover:border-lime/30 text-left flex items-center justify-between transition-colors group"
              >
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-xl bg-lime/10 text-lime flex items-center justify-center">
                    <UserCheck className="w-4 h-4" />
                  </div>
                  <div>
                    <div className="text-xs font-bold text-white group-hover:text-lime transition-colors">
                      Entrar como Profa Camila
                    </div>
                    <div className="text-[10px] text-ink-300">Prescrição rápida e modelos de treino</div>
                  </div>
                </div>
                <ChevronRight className="w-4 h-4 text-ink-400 group-hover:text-lime" />
              </button>
            )}

            {roleMode === "gestor" && (
              <div className="space-y-1.5">
                <button
                  onClick={() => demoLogin("admin@unika.app", "unika2026", "gestor")}
                  disabled={loading}
                  className="w-full p-2.5 rounded-2xl bg-ink-800 hover:bg-ink-750 border border-white/5 hover:border-lime/30 text-left flex items-center justify-between transition-colors group"
                >
                  <div className="flex items-center gap-2.5">
                    <div className="w-8 h-8 rounded-xl bg-lime/10 text-lime flex items-center justify-center">
                      <ShieldCheck className="w-4 h-4" />
                    </div>
                    <div>
                      <div className="text-xs font-bold text-white group-hover:text-lime transition-colors">
                        Gestão Unika (Dono / Matriz)
                      </div>
                      <div className="text-[10px] text-ink-300">Visão consolidada das 3 academias</div>
                    </div>
                  </div>
                  <ChevronRight className="w-4 h-4 text-ink-400 group-hover:text-lime" />
                </button>
              </div>
            )}
          </div>

          {/* Link para o Portal do Gestor no Desktop */}
          {roleMode !== "gestor" && (
            <div className="mt-4 text-center">
              <button
                onClick={() => setRoleMode("gestor")}
                className="text-xs text-ink-300 hover:text-white transition-colors inline-flex items-center gap-1.5"
              >
                <Laptop className="w-3.5 h-3.5" />
                <span>É da administração? Acesse o <b>Portal do Gestor</b></span>
              </button>
            </div>
          )}
        </div>
      </main>

      {/* Rodapé com conformidade LGPD e Termos */}
      <footer className="max-w-md md:max-w-3xl mx-auto w-full pt-3 pb-1 flex flex-col sm:flex-row items-center justify-between gap-2 text-[11px] text-ink-400">
        <div className="flex items-center gap-1.5">
          <Lock className="w-3.5 h-3.5 text-lime" />
          <span>LGPD (Lei 13.709/2018) Ativa · Multi-tenant Seguro</span>
        </div>

        <div className="flex items-center gap-4">
          <Link href="/privacidade" className="hover:text-white transition-colors underline">
            Política de Privacidade & Termos
          </Link>
          <span>Capacitor Mobile v2.1</span>
        </div>
      </footer>
    </div>
  );
}
