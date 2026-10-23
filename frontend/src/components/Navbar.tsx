"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { LogOut, Bell, Sun, Moon } from "lucide-react";
import { api, clearToken, getToken } from "@/services/api";
import { Avatar } from "@/components/ui";
import { useTheme } from "@/context/ThemeContext";
import { useSession } from "@/lib/session";
import { dateTimeBR, ROLE_LABEL } from "@/lib/format";
import { asset } from "@/lib/paths";
import type { Notification } from "@/types";

export default function Navbar() {
  const router = useRouter();
  const pathname = usePathname();
  const { theme, toggleTheme } = useTheme();
  const { user, can } = useSession();
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<Notification[]>([]);
  const [unread, setUnread] = useState(0);
  const box = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    if (!getToken()) return;
    try {
      const r = await api.myNotifications();
      setItems(r.data);
      setUnread(r.unread);
    } catch {
      /* sem rede ou sessão encerrada: o request já trata o 401 */
    }
  }, []);

  useEffect(() => {
    if (!user) return;
    load();
    const t = setInterval(load, 60000);
    return () => clearInterval(t);
  }, [user?.id, pathname, load]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const close = (e: MouseEvent) => { if (box.current && !box.current.contains(e.target as Node)) setOpen(false); };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  const handleLogout = () => {
    clearToken();
    router.push("/");
  };

  const openItem = async (n: Notification) => {
    setOpen(false);
    if (!n.read_at) {
      setItems((l) => l.map((x) => (x.id === n.id ? { ...x, read_at: new Date().toISOString() } : x)));
      setUnread((u) => Math.max(0, u - 1));
      api.markNotificationsRead([n.id]).catch(() => {});
    }
    if (n.link) router.push(n.link);
  };
  const readAll = async () => {
    await api.markNotificationsRead([]).catch(() => {});
    setItems((l) => l.map((x) => ({ ...x, read_at: x.read_at || new Date().toISOString() })));
    setUnread(0);
  };

  // A tela de login tem seu próprio cabeçalho
  if (pathname === "/") return null;

  const links: { href: string; label: string }[] = [];
  if (user?.role === "ALUNO") links.push({ href: "/aluno", label: "Meu app" });
  else if (user?.role === "PROFESSOR") links.push({ href: "/professor", label: "Aulas e treinos" });
  else if (user) {
    links.push({ href: "/admin", label: "Gestão" });
    if (can("workouts.manage")) links.push({ href: "/professor", label: "Treinos" });
  }

  return (
    <header className="sticky top-0 z-50 px-3 sm:px-6 pt-safe">
      <div className="max-w-7xl mx-auto h-14 sm:h-16 pl-2 sm:pl-3 pr-1.5 sm:pr-2 flex items-center justify-between rounded-full bg-ink-850/80 backdrop-blur-xl border border-white/[0.05]">
        <div className="flex items-center gap-6">
          <Link href={user ? links[0]?.href || "/" : "/"} className="flex items-center gap-2.5 group">
            <img src={asset("/logo-192.png")} alt="" className="w-9 h-9 sm:w-10 sm:h-10 rounded-full group-hover:shadow-glow transition-shadow" />
            <span className="font-bold text-base sm:text-lg tracking-tight text-white">
              {user?.gym?.name?.split(" ")[0] || "Unika"}<span className="text-brand">.</span>
            </span>
          </Link>

          {links.length > 1 && (
            <nav className="hidden md:flex items-center gap-1 p-1 rounded-full bg-ink-750">
              {links.map((l) => (
                <Link key={l.href} href={l.href} className={`px-4 py-2 rounded-full text-sm font-medium transition-all ${pathname.startsWith(l.href) ? "bg-brand text-black" : "text-ink-300 hover:text-white"}`}>
                  {l.label}
                </Link>
              ))}
            </nav>
          )}
        </div>

        <div className="flex items-center gap-1.5 sm:gap-2">
          <button onClick={toggleTheme} className="btn-icon bg-ink-750 text-ink-200 hover:text-white transition-transform active:scale-90" title={theme === "dark" ? "Mudar para Modo Claro" : "Mudar para Modo Escuro"} aria-label="Alternar tema">
            {theme === "dark" ? <Sun className="w-[18px] h-[18px] text-amber-300" /> : <Moon className="w-[18px] h-[18px] text-indigo-500" />}
          </button>

          {user ? (
            <>
              <div className="relative" ref={box}>
                <button onClick={() => setOpen(!open)} className="btn-icon bg-ink-750 text-ink-200 hover:text-white relative" title="Notificações" aria-label={`Notificações${unread ? `, ${unread} não lidas` : ""}`}>
                  <Bell className="w-[18px] h-[18px]" />
                  {unread > 0 && <span className="min-w-[18px] h-[18px] px-1 rounded-full bg-brand text-black text-[10px] font-bold flex items-center justify-center absolute -top-0.5 -right-0.5">{unread > 9 ? "9+" : unread}</span>}
                </button>
                {open && (
                  <div className="absolute right-0 mt-2 w-[min(22rem,calc(100vw-1.5rem))] rounded-3xl bg-ink-850 border border-white/10 shadow-card p-4 z-50 animate-fade-up">
                    <div className="flex items-center justify-between pb-3 border-b border-white/5">
                      <span className="text-sm font-semibold text-white">Notificações</span>
                      {unread > 0 && <button onClick={readAll} className="text-[11px] text-brand font-semibold hover:underline">Marcar todas como lidas</button>}
                    </div>
                    <div className="space-y-2 pt-3 max-h-[60vh] overflow-y-auto">
                      {items.length === 0 && <div className="text-xs text-ink-300 py-6 text-center">Nada por aqui ainda.</div>}
                      {items.map((n) => (
                        <button key={n.id} onClick={() => openItem(n)} className={`w-full text-left p-2.5 rounded-2xl text-xs transition-colors hover:bg-ink-700 ${n.read_at ? "bg-ink-800" : "bg-ink-750"}`}>
                          <div className="flex items-start gap-2">
                            {!n.read_at && <span className="w-2 h-2 rounded-full bg-brand mt-1 shrink-0" />}
                            <div className="min-w-0">
                              <div className="font-semibold text-white">{n.title}</div>
                              <div className="text-ink-300 text-[11px] mt-0.5">{n.body}</div>
                              <div className="text-ink-400 text-[10px] mt-1">{dateTimeBR(n.created_at)}</div>
                            </div>
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2.5 pl-1 pr-1 sm:pr-3 py-1 rounded-full bg-ink-750">
                <Avatar name={user.name} size={36} />
                <div className="hidden sm:block leading-tight">
                  <div className="text-sm font-semibold text-white">{user.name?.split(" ")[0]}</div>
                  <div className="text-[11px] text-brand font-medium">{user.job_title || ROLE_LABEL[user.role] || user.role}</div>
                </div>
              </div>
              <button onClick={handleLogout} title="Sair" aria-label="Sair" className="btn-icon bg-ink-750 text-ink-300 hover:text-[#FF7A5C] hover:bg-[#FF5A36]/10">
                <LogOut className="w-[18px] h-[18px]" />
              </button>
            </>
          ) : (
            <Link href="/" className="btn-brand !py-2.5">Entrar</Link>
          )}
        </div>
      </div>

      {links.length > 1 && (
        <nav className="md:hidden mt-2 flex items-center gap-1 p-1 rounded-full bg-ink-850/80 backdrop-blur-xl border border-white/[0.05] overflow-x-auto no-scrollbar">
          {links.map((l) => (
            <Link key={l.href} href={l.href} className={`flex-1 text-center px-3 py-2 rounded-full text-xs font-semibold whitespace-nowrap transition-all ${pathname.startsWith(l.href) ? "bg-brand text-black" : "text-ink-300"}`}>{l.label}</Link>
          ))}
        </nav>
      )}
    </header>
  );
}
