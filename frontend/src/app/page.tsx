"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  AlertCircle, ArrowRight, BatteryFull, ChevronRight, ClipboardList, Download,
  Dumbbell, Eye, EyeOff, Loader2, Lock, Mail, MessageCircle, Phone, QrCode,
  ShieldCheck, Signal, Sparkles, TrendingUp, UserCheck, Users, Wifi, X,
} from "lucide-react";
import { api, homeFor, loadSession } from "@/services/api";
import { IMG } from "@/lib/images";
import { asset } from "@/lib/paths";

const HEROES = [
  { src: IMG.kettlebell, alt: "Professora orientando aluna no treino" },
  { src: IMG.gym, alt: "Espaço de musculação da Unika" },
  { src: IMG.strength, alt: "Aluna treinando levantamento terra" },
];

type DemoAccount = {
  name: string;
  role: string;
  description: string;
  ident: string;
  icon: React.ElementType;
};

const DEMO_ACCOUNTS: DemoAccount[] = [
  {
    name: "Ana",
    role: "Aluna",
    description: "Ficha, cargas e check-in",
    ident: "aluna@unika.app",
    icon: Dumbbell,
  },
  {
    name: "Camila",
    role: "Professora",
    description: "Aulas, prescrição e chamada",
    ident: "professora@unika.app",
    icon: UserCheck,
  },
  {
    name: "Gestão Unika",
    role: "Admin",
    description: "Financeiro, relatórios e equipe",
    ident: "admin@unika.app",
    icon: ShieldCheck,
  },
];

type Sheet = "demo" | "help" | "install" | null;

type InstallPromptEvent = Event & {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
};

function BottomSheet({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center md:items-center md:p-6" role="dialog" aria-modal="true">
      <button aria-label="Fechar" onClick={onClose} className="absolute inset-0 cursor-default bg-black/70 backdrop-blur-sm" />
      <div className="login-sheet relative z-10 max-h-[86dvh] w-full max-w-[430px] overflow-y-auto rounded-t-[2rem] border border-white/10 bg-[#101014] px-5 pb-[max(1.5rem,env(safe-area-inset-bottom))] pt-3 shadow-[0_-30px_90px_rgba(0,0,0,0.85)] no-scrollbar md:rounded-[2rem] md:pb-6">
        <div className="mx-auto mb-4 h-1 w-10 rounded-full bg-white/15 md:hidden" />
        <button
          onClick={onClose}
          aria-label="Fechar"
          className="absolute right-4 top-4 flex h-8 w-8 items-center justify-center rounded-full bg-white/5 text-white/60 transition-colors hover:bg-white/10 hover:text-white"
        >
          <X className="h-4 w-4" />
        </button>
        {children}
      </div>
    </div>
  );
}

export default function HomePage() {
  const router = useRouter();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Experiência mobile
  const [slide, setSlide] = useState(0);
  const [sheet, setSheet] = useState<Sheet>(null);
  const [clock, setClock] = useState("");
  const [gym, setGym] = useState<{ name: string; phone: string } | null>(null);
  const [deferredPrompt, setDeferredPrompt] = useState<InstallPromptEvent | null>(null);
  const [isStandalone, setIsStandalone] = useState(false);

  const gymName = gym?.name || "Unika";
  const phoneDigits = (gym?.phone || "").replace(/\D/g, "");
  const intlPhone = phoneDigits.length >= 12 ? phoneDigits : phoneDigits ? `55${phoneDigits}` : "";
  const whatsappLink = intlPhone ? `https://wa.me/${intlPhone}` : null;

  // Redireciona quem já tem sessão ativa
  useEffect(() => {
    const session = loadSession();
    if (session?.user) {
      router.replace(homeFor(session.user.role));
    }
  }, [router]);

  // Lembra o último acesso no aparelho
  useEffect(() => {
    const saved = localStorage.getItem("gymsync_last_identifier");
    if (saved) setIdentifier(saved);
  }, []);

  // Relógio da barra de status (apenas na moldura desktop)
  useEffect(() => {
    const tick = () =>
      setClock(new Date().toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" }));
    tick();
    const id = window.setInterval(tick, 30_000);
    return () => window.clearInterval(id);
  }, []);

  // Dados públicos da academia (nome/telefone reais)
  useEffect(() => {
    let alive = true;
    api
      .publicGym()
      .then((g) => {
        if (alive) setGym({ name: g.name, phone: g.phone });
      })
      .catch(() => {
        /* segue com o fallback local */
      });
    return () => {
      alive = false;
    };
  }, []);

  // Slideshow do hero com pausa quando uma folha está aberta
  useEffect(() => {
    if (sheet) return;
    const reduce = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    if (reduce) return;
    const id = window.setInterval(() => setSlide((s) => (s + 1) % HEROES.length), 6500);
    return () => window.clearInterval(id);
  }, [sheet]);

  // Captura do evento de instalação PWA
  useEffect(() => {
    const handleBeforeInstall = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e as InstallPromptEvent);
    };

    window.addEventListener("beforeinstallprompt", handleBeforeInstall);

    if (
      window.matchMedia("(display-mode: standalone)").matches ||
      (window.navigator as unknown as { standalone?: boolean }).standalone === true
    ) {
      setIsStandalone(true);
    }

    return () => window.removeEventListener("beforeinstallprompt", handleBeforeInstall);
  }, []);

  const haptic = (ms = 8) => {
    try {
      navigator.vibrate?.(ms);
    } catch {
      /* sem suporte */
    }
  };

  const handleInstall = async () => {
    haptic(6);
    if (deferredPrompt) {
      await deferredPrompt.prompt();
      const { outcome } = await deferredPrompt.userChoice;
      if (outcome === "accepted") setDeferredPrompt(null);
    } else {
      setSheet("install");
    }
  };

  const doLogin = async (ident: string, pass: string) => {
    if (loading) return;
    setError(null);
    setLoading(true);
    try {
      const res = await api.login(ident, pass);
      localStorage.setItem("gymsync_last_identifier", ident);
      router.push(homeFor(res.user.role));
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Falha no login. Verifique seus dados.";
      setError(message);
      haptic(20);
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    doLogin(identifier, password);
  };

  const demoLogin = (account: DemoAccount) => {
    setIdentifier(account.ident);
    setPassword("unika2026");
    setSheet(null);
    haptic(10);
    doLogin(account.ident, "unika2026");
  };

  const openContact = () => {
    haptic(6);
    if (whatsappLink) {
      window.open(whatsappLink, "_blank", "noopener,noreferrer");
    } else {
      setSheet("help");
    }
  };

  return (
    <div className="login-screen relative flex min-h-[100dvh] flex-col overflow-hidden bg-[#040406]">
      {/* Fundo vivo: aurora + grão */}
      <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
        <div className="login-aurora absolute -left-[18%] -top-[12%] h-[70vmin] w-[70vmin] rounded-full bg-[#2A57FF]/45 blur-[90px]" />
        <div className="login-aurora-slow absolute -bottom-[18%] -right-[20%] h-[80vmin] w-[80vmin] rounded-full bg-[#FF8A00]/40 blur-[100px]" />
        <div className="login-aurora-reverse absolute left-[30%] top-[28%] h-[52vmin] w-[52vmin] rounded-full bg-brand/30 blur-[90px]" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,rgba(255,255,255,0.07),transparent_55%)]" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_30%,rgba(0,0,0,0.55))]" />
      </div>
      <div aria-hidden className="login-grain pointer-events-none absolute inset-0" />

      <div className="relative z-10 flex flex-1 flex-col p-0 md:items-center md:justify-center md:py-8">
        <main className="relative mx-auto flex w-full flex-1 flex-col overflow-hidden bg-[#08080a] md:w-[424px] md:flex-none md:rounded-[3rem] md:border-[7px] md:border-[#151518] md:shadow-[0_60px_140px_-30px_rgba(0,0,0,0.95),0_0_0_1px_rgba(255,255,255,0.06)]">
          {/* Barra de status simulada (desktop) */}
          <div className="pointer-events-none absolute inset-x-0 top-0 z-40 hidden items-center justify-between px-6 pt-3 text-[11px] font-bold tabular-nums text-white/90 md:flex">
            <span>{clock || "9:41"}</span>
            <div className="flex items-center gap-1.5">
              <Signal className="h-3 w-3" />
              <Wifi className="h-3 w-3" />
              <BatteryFull className="h-4 w-4" />
            </div>
          </div>
          <div className="pointer-events-none absolute left-1/2 top-2 z-40 hidden h-6 w-24 -translate-x-1/2 rounded-full bg-black md:block" />

          {/* Hero com fotos reais da academia */}
          <section className="relative h-[36dvh] max-h-[380px] min-h-[290px] w-full shrink-0 overflow-hidden md:h-[364px] md:max-h-none md:min-h-0">
            {HEROES.map((hero, i) => (
              <div
                key={hero.src}
                className={`absolute inset-0 transition-opacity duration-[1400ms] ease-out ${
                  i === slide ? "opacity-100" : "opacity-0"
                }`}
                aria-hidden={i !== slide}
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={hero.src}
                  alt={hero.alt}
                  draggable={false}
                  className={`h-full w-full object-cover transition-transform duration-[7000ms] ease-out ${
                    i === slide ? "scale-110" : "scale-100"
                  }`}
                />
              </div>
            ))}
            <div className="absolute inset-0 bg-gradient-to-b from-black/70 via-black/10 to-[#08080a]" />
            <div className="absolute inset-0 bg-gradient-to-t from-[#08080a] via-transparent to-transparent" />
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_30%,rgba(0,0,0,0.5))]" />

            {/* Marca + instalação */}
            <div className="absolute inset-x-0 top-0 z-20 flex items-center justify-between px-5 pt-[max(1rem,env(safe-area-inset-top))] md:pt-11">
              <div className="flex items-center gap-2.5">
                <div className="relative">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={asset("/logo-192.png")}
                    alt={`Logo ${gymName}`}
                    className="h-10 w-10 rounded-[14px] shadow-[0_8px_24px_rgba(0,0,0,0.5)] ring-1 ring-white/25"
                  />
                  <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-black bg-emerald-400" />
                </div>
                <div className="leading-tight">
                  <div className="text-[13px] font-extrabold tracking-tight text-white">{gymName}</div>
                  <div className="text-[9px] font-bold uppercase tracking-[0.22em] text-white/55">GymSync</div>
                </div>
              </div>

              {!isStandalone && (
                <button
                  type="button"
                  onClick={handleInstall}
                  className="flex h-9 items-center gap-1.5 rounded-full border border-white/15 bg-black/30 px-3 text-[11px] font-semibold text-white/90 backdrop-blur-md transition-all hover:bg-black/45 active:scale-95"
                >
                  <Download className="h-3.5 w-3.5" />
                  <span>Instalar</span>
                </button>
              )}
            </div>

            {/* Chamada de boas-vindas */}
            <div className="absolute inset-x-0 bottom-0 z-20 px-6 pb-14 text-center md:pb-16">
              <div className="mb-3 inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-3 py-1 text-[9px] font-bold uppercase tracking-[0.18em] text-white/85 backdrop-blur-md">
                <Sparkles className="h-3 w-3 text-brand" />
                Sua evolução começa aqui
              </div>
              <h1 className="text-[clamp(1.9rem,8vw,2.45rem)] font-extrabold leading-[1.03] tracking-tight text-white drop-shadow-[0_6px_30px_rgba(0,0,0,0.6)]">
                Bem-vindo
                <br />
                <span className="text-brand">de volta.</span>
              </h1>
              <p className="mx-auto mt-2.5 max-w-[17rem] text-[12px] leading-relaxed text-white/75">
                Sua ficha, sua evolução e seu check-in na palma da mão.
              </p>
            </div>
          </section>

          {/* Painel de acesso */}
          <section className="relative z-30 -mt-7 flex flex-1 flex-col rounded-t-[2rem] bg-gradient-to-b from-[#0b0b0e] via-[#09090b] to-[#08080a] px-5 pb-[max(1.5rem,env(safe-area-inset-bottom))] pt-6 md:flex-none md:px-7 md:pb-7">
            {/* Indicador das fotos do hero */}
            <div className="mx-auto mb-5 flex w-24 gap-1.5" role="tablist" aria-label="Fotos da academia">
              {HEROES.map((hero, i) => (
                <button
                  key={hero.src}
                  type="button"
                  role="tab"
                  aria-selected={i === slide}
                  aria-label={`Foto ${i + 1} de ${HEROES.length}`}
                  onClick={() => setSlide(i)}
                  className="group relative h-1 flex-1 overflow-hidden rounded-full bg-white/15"
                >
                  <span
                    className={`absolute inset-y-0 left-0 rounded-full bg-brand transition-all duration-500 ${
                      i === slide ? "w-full" : "w-0 group-hover:w-1/3"
                    }`}
                  />
                </button>
              ))}
            </div>

            <p className="mb-3 text-[10px] font-bold uppercase tracking-[0.2em] text-white/35">Acesse sua conta</p>

            {error && (
              <div
                role="alert"
                className="mb-3 flex items-start gap-2 rounded-2xl border border-brand/25 bg-brand/10 px-3.5 py-3 text-xs text-[#FFB1A6] animate-fade-up"
              >
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <form onSubmit={handleLogin} className="space-y-3">
              <div>
                <label htmlFor="login-identifier" className="sr-only">
                  E-mail ou CPF
                </label>
                <div className="relative">
                  <Mail className="pointer-events-none absolute left-4 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-white/35" />
                  <input
                    id="login-identifier"
                    name="identifier"
                    type="text"
                    inputMode="text"
                    autoComplete="username"
                    autoCapitalize="none"
                    spellCheck={false}
                    value={identifier}
                    onChange={(e) => {
                      setIdentifier(e.target.value);
                      setError(null);
                    }}
                    placeholder="E-mail ou CPF"
                    required
                    className="h-14 w-full rounded-2xl border border-white/10 bg-white/[0.045] pl-11 pr-4 text-base text-white outline-none transition-all placeholder:text-white/30 focus:border-brand/60 focus:bg-white/[0.07] focus:ring-4 focus:ring-brand/10"
                  />
                </div>
              </div>

              <div>
                <label htmlFor="login-password" className="sr-only">
                  Senha
                </label>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-4 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-white/35" />
                  <input
                    id="login-password"
                    name="password"
                    type={showPassword ? "text" : "password"}
                    autoComplete="current-password"
                    enterKeyHint="go"
                    value={password}
                    onChange={(e) => {
                      setPassword(e.target.value);
                      setError(null);
                    }}
                    placeholder="Sua senha"
                    required
                    className="h-14 w-full rounded-2xl border border-white/10 bg-white/[0.045] pl-11 pr-12 text-base text-white outline-none transition-all placeholder:text-white/30 focus:border-brand/60 focus:bg-white/[0.07] focus:ring-4 focus:ring-brand/10"
                  />
                  <button
                    type="button"
                    onClick={() => {
                      setShowPassword((v) => !v);
                      haptic(5);
                    }}
                    aria-label={showPassword ? "Ocultar senha" : "Mostrar senha"}
                    className="absolute right-2 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full text-white/40 transition-colors hover:bg-white/5 hover:text-white/80"
                  >
                    {showPassword ? <EyeOff className="h-[18px] w-[18px]" /> : <Eye className="h-[18px] w-[18px]" />}
                  </button>
                </div>
              </div>

              <div className="flex justify-end pt-0.5">
                <button
                  type="button"
                  onClick={() => setSheet("help")}
                  className="text-[11px] font-medium text-white/40 transition-colors hover:text-white/80"
                >
                  Esqueci minha senha
                </button>
              </div>

              <button
                type="submit"
                disabled={loading}
                aria-busy={loading}
                className="group mt-1 flex h-14 w-full items-center justify-center gap-2 rounded-full bg-brand text-[15px] font-bold text-black shadow-[0_18px_45px_-14px_rgba(254,111,95,0.8)] transition-all hover:bg-brand-300 active:scale-[0.985] disabled:pointer-events-none disabled:opacity-60"
              >
                {loading ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Entrando…
                  </>
                ) : (
                  <>
                    Entrar
                    <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
                  </>
                )}
              </button>
            </form>

            <button
              type="button"
              onClick={() => {
                setError(null);
                setSheet("demo");
                haptic(6);
              }}
              className="mt-3 flex h-[52px] w-full items-center justify-center gap-2.5 rounded-full border border-white/[0.12] bg-white/[0.035] text-[13px] font-semibold text-white/90 transition-all hover:bg-white/[0.07] active:scale-[0.985]"
            >
              <Users className="h-[18px] w-[18px] text-white/55" />
              Acesso de demonstração
              <ChevronRight className="h-4 w-4 text-white/30" />
            </button>

            <div className="mt-auto flex items-center justify-center gap-3.5 pt-6 text-[9px] font-bold uppercase tracking-[0.14em] text-white/25">
              <span className="flex items-center gap-1">
                <QrCode className="h-3 w-3" />
                Check-in
              </span>
              <span className="h-0.5 w-0.5 rounded-full bg-white/20" />
              <span className="flex items-center gap-1">
                <ClipboardList className="h-3 w-3" />
                Fichas
              </span>
              <span className="h-0.5 w-0.5 rounded-full bg-white/20" />
              <span className="flex items-center gap-1">
                <TrendingUp className="h-3 w-3" />
                Evolução
              </span>
            </div>

            <div className="pt-4 text-center">
              <p className="text-xs text-white/40">
                Ainda não tem acesso?{" "}
                <button
                  type="button"
                  onClick={openContact}
                  className="font-bold text-brand transition-colors hover:text-brand-300"
                >
                  Fale com a recepção
                </button>
              </p>
              <p className="mt-3 text-[10px] leading-relaxed text-white/25">
                Ao entrar, você concorda com os{" "}
                <Link
                  href="/privacidade"
                  className="underline decoration-white/20 underline-offset-2 transition-colors hover:text-white/60"
                >
                  termos e a Política de Privacidade
                </Link>
                .
              </p>
            </div>
          </section>

          {/* Folhas de apoio */}
          {sheet === "demo" && (
            <BottomSheet onClose={() => setSheet(null)}>
              <div className="pr-8">
                <h2 className="text-base font-bold text-white">Acesso de demonstração</h2>
                <p className="mt-1 text-xs leading-relaxed text-white/45">
                  Escolha um perfil para conhecer o app. Os dados são fictícios.
                </p>
              </div>

              <div className="mt-4 space-y-2">
                {DEMO_ACCOUNTS.map((account) => {
                  const Icon = account.icon;
                  return (
                    <button
                      key={account.ident}
                      type="button"
                      onClick={() => demoLogin(account)}
                      disabled={loading}
                      className="group flex w-full items-center gap-3 rounded-2xl border border-white/[0.08] bg-white/[0.035] p-3.5 text-left transition-all hover:border-brand/30 hover:bg-white/[0.06] active:scale-[0.99] disabled:opacity-50"
                    >
                      <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-brand/10 text-brand">
                        <Icon className="h-5 w-5" />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="flex items-center gap-1.5 text-sm font-bold text-white">
                          {account.name}
                          <span className="rounded-full bg-white/[0.08] px-2 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-white/50">
                            {account.role}
                          </span>
                        </span>
                        <span className="mt-0.5 block truncate text-[11px] text-white/40">
                          {account.description} · senha unika2026
                        </span>
                      </span>
                      <ArrowRight className="h-4 w-4 shrink-0 text-white/25 transition-all group-hover:translate-x-0.5 group-hover:text-brand" />
                    </button>
                  );
                })}
              </div>

              <p className="mt-4 text-center text-[10px] text-white/25">Ambiente de testes · dados fictícios</p>
            </BottomSheet>
          )}

          {sheet === "help" && (
            <BottomSheet onClose={() => setSheet(null)}>
              <div className="pr-8">
                <h2 className="text-base font-bold text-white">Precisa de ajuda para entrar?</h2>
                <p className="mt-1 text-xs leading-relaxed text-white/45">
                  Fale com a recepção da {gymName} para criar seu acesso ou redefinir sua senha.
                </p>
              </div>

              {gym?.phone && (
                <div className="mt-4 flex items-center gap-3 rounded-2xl border border-white/[0.08] bg-white/[0.035] p-3.5">
                  <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand/10 text-brand">
                    <Phone className="h-[18px] w-[18px]" />
                  </span>
                  <div>
                    <div className="text-[10px] font-semibold uppercase tracking-wide text-white/35">Recepção</div>
                    <div className="text-sm font-bold text-white">{gym.phone}</div>
                  </div>
                </div>
              )}

              <div className="mt-4 space-y-2">
                {whatsappLink && (
                  <a
                    href={whatsappLink}
                    target="_blank"
                    rel="noreferrer"
                    className="flex h-12 w-full items-center justify-center gap-2 rounded-full bg-brand text-[13px] font-bold text-black transition-all hover:bg-brand-300 active:scale-[0.985]"
                  >
                    <MessageCircle className="h-4 w-4" />
                    Chamar no WhatsApp
                  </a>
                )}
                {phoneDigits && (
                  <a
                    href={`tel:+${intlPhone}`}
                    className="flex h-12 w-full items-center justify-center gap-2 rounded-full border border-white/[0.12] bg-white/[0.035] text-[13px] font-semibold text-white transition-all hover:bg-white/[0.07] active:scale-[0.985]"
                  >
                    <Phone className="h-4 w-4 text-white/55" />
                    Ligar para a recepção
                  </a>
                )}
                <button
                  type="button"
                  onClick={() => setSheet(null)}
                  className="flex h-12 w-full items-center justify-center rounded-full text-[13px] font-semibold text-white/50 transition-colors hover:text-white"
                >
                  Voltar para o login
                </button>
              </div>
            </BottomSheet>
          )}

          {sheet === "install" && (
            <BottomSheet onClose={() => setSheet(null)}>
              <div className="pr-8">
                <h2 className="text-base font-bold text-white">Instalar o GymSync</h2>
                <p className="mt-1 text-xs leading-relaxed text-white/45">
                  Tenha o app na tela de início, sem precisar de loja de aplicativos.
                </p>
              </div>

              <div className="mt-4 space-y-2.5">
                <div className="rounded-2xl border border-white/[0.08] bg-white/[0.035] p-3.5">
                  <div className="flex items-center gap-2 text-xs font-bold text-white">
                    <span className="rounded-md bg-white/10 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-white/60">iPhone</span>
                    Safari
                  </div>
                  <p className="mt-1.5 text-[11px] leading-relaxed text-white/50">
                    Toque em <b className="text-white/80">Compartilhar</b> e depois em{" "}
                    <b className="text-white/80">Adicionar à Tela de Início</b>.
                  </p>
                </div>
                <div className="rounded-2xl border border-white/[0.08] bg-white/[0.035] p-3.5">
                  <div className="flex items-center gap-2 text-xs font-bold text-white">
                    <span className="rounded-md bg-white/10 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-white/60">Android</span>
                    Chrome
                  </div>
                  <p className="mt-1.5 text-[11px] leading-relaxed text-white/50">
                    Abra o menu <b className="text-white/80">⋮</b> e toque em{" "}
                    <b className="text-white/80">Instalar aplicativo</b>.
                  </p>
                </div>
              </div>

              <button
                type="button"
                onClick={() => setSheet(null)}
                className="mt-4 flex h-12 w-full items-center justify-center rounded-full bg-brand text-[13px] font-bold text-black transition-all hover:bg-brand-300 active:scale-[0.985]"
              >
                Entendi
              </button>
            </BottomSheet>
          )}
        </main>
      </div>
    </div>
  );
}
