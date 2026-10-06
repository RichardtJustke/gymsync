"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { Dumbbell, Users, CreditCard, LogOut, CheckCircle2, ShieldCheck, UserCheck, Calendar } from "lucide-react";
import { clearToken } from "@/services/api";

export default function Navbar() {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    const raw = localStorage.getItem("gymsync_user");
    if (raw) {
      try {
        setUser(JSON.parse(raw));
      } catch {}
    }
  }, [pathname]);

  const handleLogout = () => {
    clearToken();
    router.push("/");
  };

  const getRoleBadge = (role: string) => {
    switch (role) {
      case "ADMIN":
        return <span className="px-2 py-0.5 text-xs font-semibold bg-blue-500/20 text-blue-400 border border-blue-500/30 rounded-full flex items-center gap-1"><ShieldCheck className="w-3 h-3" /> Gestor</span>;
      case "PROFESSOR":
        return <span className="px-2 py-0.5 text-xs font-semibold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 rounded-full flex items-center gap-1"><UserCheck className="w-3 h-3" /> Professor</span>;
      case "ALUNO":
        return <span className="px-2 py-0.5 text-xs font-semibold bg-amber-500/20 text-amber-400 border border-amber-500/30 rounded-full flex items-center gap-1"><Dumbbell className="w-3 h-3" /> Aluno</span>;
      default:
        return null;
    }
  };

  return (
    <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur-md sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href="/" className="flex items-center gap-2 group">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center shadow-lg shadow-blue-500/20 group-hover:scale-105 transition-transform">
              <Dumbbell className="w-5 h-5 text-white" />
            </div>
            <div>
              <span className="font-bold text-lg bg-gradient-to-r from-white via-slate-200 to-slate-400 bg-clip-text text-transparent">
                GymSync
              </span>
              <span className="hidden sm:inline-block text-[10px] text-blue-400 font-medium ml-2 px-1.5 py-0.2 bg-blue-950/60 border border-blue-800/40 rounded">
                SaaS
              </span>
            </div>
          </Link>

          {user && (
            <nav className="hidden md:flex items-center gap-1 text-sm font-medium">
              {user.role === "ADMIN" && (
                <>
                  <Link
                    href="/admin"
                    className={`px-3 py-1.5 rounded-lg transition-colors ${pathname === "/admin" ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"}`}
                  >
                    Painel do Gestor
                  </Link>
                  <Link
                    href="/professor"
                    className={`px-3 py-1.5 rounded-lg transition-colors ${pathname === "/professor" ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"}`}
                  >
                    Visão Professor
                  </Link>
                  <Link
                    href="/aluno"
                    className={`px-3 py-1.5 rounded-lg transition-colors ${pathname === "/aluno" ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"}`}
                  >
                    Visão Aluno
                  </Link>
                </>
              )}

              {user.role === "PROFESSOR" && (
                <>
                  <Link
                    href="/professor"
                    className={`px-3 py-1.5 rounded-lg transition-colors ${pathname === "/professor" ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"}`}
                  >
                    Prescrição de Treinos
                  </Link>
                  <Link
                    href="/aluno"
                    className={`px-3 py-1.5 rounded-lg transition-colors ${pathname === "/aluno" ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"}`}
                  >
                    Visualizar App Aluno
                  </Link>
                </>
              )}

              {user.role === "ALUNO" && (
                <Link
                  href="/aluno"
                  className={`px-3 py-1.5 rounded-lg transition-colors ${pathname === "/aluno" ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"}`}
                >
                  Meu Treino & Faturas
                </Link>
              )}
            </nav>
          )}
        </div>

        <div className="flex items-center gap-3">
          {user ? (
            <>
              <div className="flex items-center gap-2">
                <div className="text-right hidden sm:block">
                  <div className="text-sm font-medium text-slate-200">{user.name}</div>
                  <div className="text-xs text-slate-400">{user.email}</div>
                </div>
                {getRoleBadge(user.role)}
              </div>
              <button
                onClick={handleLogout}
                title="Sair"
                className="p-2 text-slate-400 hover:text-red-400 hover:bg-slate-800/60 rounded-lg transition-colors"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </>
          ) : (
            <Link
              href="/"
              className="text-sm font-medium text-slate-300 hover:text-white px-3 py-1.5 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors"
            >
              Entrar
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
