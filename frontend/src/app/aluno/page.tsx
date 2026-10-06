"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { 
  Dumbbell, CreditCard, QrCode, CheckCircle2, Clock, 
  Play, Pause, RotateCcw, Copy, ExternalLink, ShieldCheck, 
  AlertCircle, Sparkles, ChevronRight, Check, ArrowRight
} from "lucide-react";
import { api } from "@/services/api";
import { Invoice, Workout, WorkoutPlan } from "@/types";

export default function AlunoPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"workout" | "finance" | "checkin">("workout");

  // Dados do Aluno
  const [planData, setPlanData] = useState<{ has_plan: boolean; student: any; plan: WorkoutPlan | null } | null>(null);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [selectedWorkoutIndex, setSelectedWorkoutIndex] = useState(0);

  // Execução do Treino & Cargas
  const [completedItems, setCompletedItems] = useState<Record<string, boolean>>({});
  const [itemLoads, setItemLoads] = useState<Record<string, number>>({});

  // Cronômetro de Descanso
  const [timerSeconds, setTimerSeconds] = useState(60);
  const [timerTotal, setTimerTotal] = useState(60);
  const [timerActive, setTimerActive] = useState(false);

  // Ações Financeiras & Feedback
  const [copiedPix, setCopiedPix] = useState(false);
  const [payingInvoiceId, setPayingInvoiceId] = useState<string | null>(null);
  const [paySuccessMsg, setPaySuccessMsg] = useState<string | null>(null);

  // Check-in
  const [checkInStatus, setCheckInStatus] = useState<string | null>(null);
  const [checkInError, setCheckInError] = useState<string | null>(null);

  const loadStudentData = async () => {
    try {
      setLoading(true);
      const [workoutsRes, invoicesRes] = await Promise.all([
        api.getMyWorkouts(),
        api.getMyInvoices(),
      ]);

      setPlanData(workoutsRes);
      setInvoices(invoicesRes);

      // Preenche cargas iniciais dos exercícios
      if (workoutsRes.plan?.workouts) {
        const loads: Record<string, number> = {};
        workoutsRes.plan.workouts.forEach((w) => {
          w.items.forEach((it) => {
            loads[it.id] = it.load_kg;
          });
        });
        setItemLoads(loads);
      }
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
    loadStudentData();
  }, []);

  // Efeito do Cronômetro de Descanso
  useEffect(() => {
    let interval: any = null;
    if (timerActive && timerSeconds > 0) {
      interval = setInterval(() => {
        setTimerSeconds((prev) => prev - 1);
      }, 1000);
    } else if (timerSeconds === 0 && timerActive) {
      setTimerActive(false);
      // Feedback sonoro simples usando Web Audio API
      if (typeof window !== "undefined" && window.AudioContext) {
        try {
          const ctx = new AudioContext();
          const osc = ctx.createOscillator();
          osc.type = "sine";
          osc.frequency.setValueAtTime(800, ctx.currentTime);
          osc.connect(ctx.destination);
          osc.start();
          osc.stop(ctx.currentTime + 0.3);
        } catch {}
      }
    }
    return () => clearInterval(interval);
  }, [timerActive, timerSeconds]);

  const startRestTimer = (seconds: number) => {
    setTimerTotal(seconds);
    setTimerSeconds(seconds);
    setTimerActive(true);
  };

  const handleUpdateLoad = async (itemId: string, newLoad: number) => {
    setItemLoads((prev) => ({ ...prev, [itemId]: newLoad }));
    try {
      await api.updateExerciseLoad(itemId, newLoad);
    } catch (err) {
      console.error("Falha ao salvar carga:", err);
    }
  };

  const toggleItemComplete = (itemId: string, restSecs: number) => {
    const isNowDone = !completedItems[itemId];
    setCompletedItems((prev) => ({ ...prev, [itemId]: isNowDone }));
    if (isNowDone) {
      startRestTimer(restSecs || 60);
    }
  };

  const handleCopyPix = (pixCode: string) => {
    navigator.clipboard.writeText(pixCode);
    setCopiedPix(true);
    setTimeout(() => setCopiedPix(false), 3000);
  };

  const handleSimulatePayment = async (invId: string) => {
    setPayingInvoiceId(invId);
    setPaySuccessMsg(null);
    try {
      const res = await api.mockPay(invId);
      setPaySuccessMsg(res.message);
      loadStudentData();
    } catch (err: any) {
      alert("Falha na simulação: " + err.message);
    } finally {
      setPayingInvoiceId(null);
    }
  };

  const handleCheckIn = async () => {
    setCheckInStatus(null);
    setCheckInError(null);
    try {
      const res = await api.doCheckIn();
      setCheckInStatus(res.message);
    } catch (err: any) {
      setCheckInError(err.message || "Acesso não liberado");
    }
  };

  const activeWorkout: Workout | undefined = planData?.plan?.workouts?.[selectedWorkoutIndex];

  return (
    <div className="flex-1 max-w-xl w-full mx-auto px-4 py-6 space-y-6">
      {/* Perfil Compacto do Aluno */}
      {planData?.student && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-full bg-gradient-to-tr from-blue-600 to-indigo-600 flex items-center justify-center font-bold text-base text-white shadow-lg shadow-blue-500/20">
              {planData.student.user?.name?.slice(0, 2).toUpperCase() || "AL"}
            </div>
            <div>
              <div className="font-bold text-white text-sm">{planData.student.user?.name}</div>
              <div className="text-xs text-slate-400">
                {planData.student.plan_name} • Venc. todo dia {planData.student.due_day}
              </div>
            </div>
          </div>

          <span
            className={`px-2.5 py-1 rounded-full text-xs font-bold ${planData.student.status === "ATIVO" ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30" : "bg-red-500/20 text-red-400 border border-red-500/30"}`}
          >
            {planData.student.status}
          </span>
        </div>
      )}

      {/* Navegação entre Abas do App Aluno */}
      <div className="grid grid-cols-3 gap-2 bg-slate-900 border border-slate-800 p-1.5 rounded-2xl">
        <button
          onClick={() => setActiveTab("workout")}
          className={`py-2 text-xs font-bold rounded-xl transition-all flex flex-col sm:flex-row items-center justify-center gap-1.5 ${activeTab === "workout" ? "bg-blue-600 text-white shadow-md shadow-blue-600/30" : "text-slate-400 hover:text-white"}`}
        >
          <Dumbbell className="w-4 h-4" />
          <span>Treino</span>
        </button>

        <button
          onClick={() => setActiveTab("finance")}
          className={`py-2 text-xs font-bold rounded-xl transition-all flex flex-col sm:flex-row items-center justify-center gap-1.5 ${activeTab === "finance" ? "bg-blue-600 text-white shadow-md shadow-blue-600/30" : "text-slate-400 hover:text-white"}`}
        >
          <CreditCard className="w-4 h-4" />
          <span>Faturas</span>
        </button>

        <button
          onClick={() => setActiveTab("checkin")}
          className={`py-2 text-xs font-bold rounded-xl transition-all flex flex-col sm:flex-row items-center justify-center gap-1.5 ${activeTab === "checkin" ? "bg-blue-600 text-white shadow-md shadow-blue-600/30" : "text-slate-400 hover:text-white"}`}
        >
          <QrCode className="w-4 h-4" />
          <span>Check-in</span>
        </button>
      </div>

      {/* ABA: FICHA DE TREINO */}
      {activeTab === "workout" && (
        <div className="space-y-4">
          {planData?.has_plan && planData.plan ? (
            <>
              {/* Seletor de Divisões (A, B, C...) */}
              <div className="flex items-center gap-2 overflow-x-auto pb-1">
                {planData.plan.workouts.map((w, idx) => (
                  <button
                    key={w.id}
                    onClick={() => setSelectedWorkoutIndex(idx)}
                    className={`px-4 py-2 rounded-xl text-xs font-bold whitespace-nowrap transition-all ${selectedWorkoutIndex === idx ? "bg-blue-600 text-white shadow-lg shadow-blue-600/20" : "bg-slate-900 border border-slate-800 text-slate-400 hover:text-white"}`}
                  >
                    Treino {w.letter}: {w.name}
                  </button>
                ))}
              </div>

              {/* Banner do Cronômetro de Descanso */}
              {timerActive && (
                <div className="bg-gradient-to-r from-amber-500/20 via-orange-500/20 to-amber-500/20 border border-amber-500/40 rounded-2xl p-4 flex items-center justify-between animate-pulse">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-amber-500/30 text-amber-400 flex items-center justify-center font-bold text-sm">
                      <Clock className="w-5 h-5 animate-spin" />
                    </div>
                    <div>
                      <div className="text-xs text-amber-300 font-semibold">Descanso entre séries</div>
                      <div className="text-2xl font-black text-amber-400 font-mono">
                        {Math.floor(timerSeconds / 60)}:{(timerSeconds % 60).toString().padStart(2, "0")}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => setTimerActive(!timerActive)}
                      className="p-2 rounded-lg bg-amber-500/20 text-amber-300 hover:bg-amber-500/30"
                    >
                      {timerActive ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                    </button>
                    <button
                      onClick={() => setTimerSeconds(timerTotal)}
                      className="p-2 rounded-lg bg-amber-500/20 text-amber-300 hover:bg-amber-500/30"
                    >
                      <RotateCcw className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              )}

              {/* Lista de Exercícios */}
              {activeWorkout ? (
                <div className="space-y-3">
                  {activeWorkout.items.map((it, idx) => {
                    const isDone = completedItems[it.id];
                    const currentLoad = itemLoads[it.id] ?? it.load_kg;

                    return (
                      <div
                        key={it.id}
                        className={`bg-slate-900 border rounded-2xl p-4 transition-all ${isDone ? "border-emerald-500/40 bg-emerald-950/10" : "border-slate-800"}`}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <button
                            onClick={() => toggleItemComplete(it.id, it.rest_seconds)}
                            className={`w-6 h-6 rounded-lg border flex items-center justify-center transition-colors shrink-0 mt-0.5 ${isDone ? "bg-emerald-600 border-emerald-500 text-white" : "border-slate-700 hover:border-blue-500"}`}
                          >
                            {isDone && <Check className="w-4 h-4" />}
                          </button>

                          <div className="flex-1">
                            <div className={`text-sm font-bold ${isDone ? "line-through text-slate-400" : "text-white"}`}>
                              {it.exercise?.name}
                            </div>
                            <div className="text-xs text-slate-400 mt-0.5">
                              {it.exercise?.muscle_group} • {it.exercise?.equipment}
                            </div>

                            <div className="flex flex-wrap items-center gap-2 mt-3 text-xs">
                              <span className="px-2 py-0.5 rounded-md bg-slate-800 text-slate-300 font-semibold">
                                {it.sets} séries × {it.reps}
                              </span>

                              {/* Ajuste de Carga */}
                              <div className="flex items-center gap-1 px-2 py-0.5 rounded-md bg-slate-800 border border-slate-700 text-slate-300">
                                <span className="text-[10px] text-slate-400">Carga:</span>
                                <input
                                  type="number"
                                  value={currentLoad}
                                  onChange={(e) => handleUpdateLoad(it.id, parseFloat(e.target.value) || 0)}
                                  className="w-12 bg-transparent text-emerald-400 font-bold text-center focus:outline-none"
                                />
                                <span className="text-[10px] text-slate-400">kg</span>
                              </div>

                              {/* Iniciar timer manual */}
                              <button
                                onClick={() => startRestTimer(it.rest_seconds || 60)}
                                className="px-2 py-0.5 rounded-md bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 border border-amber-500/20 flex items-center gap-1 font-semibold"
                              >
                                <Clock className="w-3 h-3" /> {it.rest_seconds}s
                              </button>
                            </div>

                            {it.notes && (
                              <div className="text-[11px] text-slate-400 italic mt-2 bg-slate-800/40 p-2 rounded-lg">
                                💡 {it.notes}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="p-8 bg-slate-900 border border-slate-800 rounded-2xl text-center text-xs text-slate-500">
                  Nenhuma divisão selecionada.
                </div>
              )}
            </>
          ) : (
            <div className="p-12 bg-slate-900 border border-slate-800 rounded-2xl text-center space-y-3">
              <Dumbbell className="w-10 h-10 text-slate-600 mx-auto" />
              <div className="text-sm font-bold text-white">Nenhum treino prescrito ainda</div>
              <p className="text-xs text-slate-400 max-w-xs mx-auto">
                Seu professor ou personal ainda não configurou sua ficha. Fale com ele na recepção ou aguarde a prescrição!
              </p>
            </div>
          )}
        </div>
      )}

      {/* ABA: FATURAS E MENSALIDADES */}
      {activeTab === "finance" && (
        <div className="space-y-4">
          <div className="text-xs text-slate-400">
            Acompanhe suas mensalidades. Baixe o boleto em PDF ou pague instantaneamente via PIX.
          </div>

          {paySuccessMsg && (
            <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>{paySuccessMsg}</span>
            </div>
          )}

          {invoices.length > 0 ? (
            <div className="space-y-3">
              {invoices.map((inv) => (
                <div key={inv.id} className="bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-4">
                  <div className="flex items-center justify-between border-b border-slate-800 pb-3">
                    <div>
                      <div className="text-xs font-bold text-white">{inv.description}</div>
                      <div className="text-[11px] text-slate-400">
                        Vencimento: {new Date(inv.due_date).toLocaleDateString("pt-BR")}
                      </div>
                    </div>
                    <span
                      className={`px-2 py-0.5 rounded-full text-xs font-bold ${inv.status === "PAGO" ? "bg-emerald-500/20 text-emerald-400" : "bg-red-500/20 text-red-400"}`}
                    >
                      {inv.status}
                    </span>
                  </div>

                  <div className="flex items-baseline justify-between">
                    <span className="text-xs text-slate-400">Valor Total</span>
                    <span className="text-xl font-black text-white">
                      R$ {inv.amount.toFixed(2)}
                    </span>
                  </div>

                  {inv.status !== "PAGO" && (
                    <div className="space-y-2 pt-2">
                      <div className="grid grid-cols-2 gap-2">
                        <a
                          href={api.getBoletoUrl(inv.id)}
                          target="_blank"
                          className="py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold flex items-center justify-center gap-1.5 border border-slate-700 transition-colors"
                        >
                          <ExternalLink className="w-3.5 h-3.5 text-blue-400" />
                          <span>Baixar Boleto</span>
                        </a>

                        <button
                          onClick={() => handleCopyPix(inv.pix_copy_paste)}
                          className="py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold flex items-center justify-center gap-1.5 border border-slate-700 transition-colors"
                        >
                          <Copy className="w-3.5 h-3.5 text-emerald-400" />
                          <span>{copiedPix ? "Copiado!" : "Copiar PIX"}</span>
                        </button>
                      </div>

                      {/* Botão de Simulação de Pagamento */}
                      <button
                        onClick={() => handleSimulatePayment(inv.id)}
                        disabled={payingInvoiceId === inv.id}
                        className="w-full py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition-colors shadow-lg shadow-emerald-600/30 flex items-center justify-center gap-2"
                      >
                        <Sparkles className="w-4 h-4 text-amber-300" />
                        {payingInvoiceId === inv.id ? "Confirmando..." : "Simular Pagamento Instantâneo"}
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="p-8 bg-slate-900 border border-slate-800 rounded-2xl text-center text-xs text-slate-500">
              Nenhuma fatura encontrada.
            </div>
          )}
        </div>
      )}

      {/* ABA: CHECK-IN / ENTRADA NA ACADEMIA */}
      {activeTab === "checkin" && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 text-center space-y-6">
          <div className="space-y-1">
            <h2 className="text-base font-bold text-white">Controle de Entrada</h2>
            <p className="text-xs text-slate-400">
              Apresente o QR Code na catraca ou confirme seu check-in pelo botão.
            </p>
          </div>

          <div className="w-48 h-48 mx-auto bg-white p-3 rounded-2xl flex items-center justify-center shadow-xl">
            {/* Visual de QR Code simulado */}
            <div className="w-full h-full bg-slate-950 rounded-xl flex flex-col items-center justify-center p-3 text-center">
              <QrCode className="w-20 h-20 text-white" />
              <span className="text-[9px] text-slate-400 font-mono mt-1">GYMSYNC-PASS-OK</span>
            </div>
          </div>

          {checkInStatus && (
            <div className="p-3 bg-emerald-500/10 border border-emerald-500/30 rounded-xl text-emerald-400 text-xs font-semibold flex items-center justify-center gap-2">
              <CheckCircle2 className="w-4 h-4" />
              <span>{checkInStatus}</span>
            </div>
          )}

          {checkInError && (
            <div className="p-3 bg-red-500/10 border border-red-500/30 rounded-xl text-red-400 text-xs font-semibold flex items-center justify-center gap-2">
              <AlertCircle className="w-4 h-4" />
              <span>{checkInError}</span>
            </div>
          )}

          <button
            onClick={handleCheckIn}
            className="w-full py-3.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold transition-colors shadow-lg shadow-blue-600/30 flex items-center justify-center gap-2"
          >
            <CheckCircle2 className="w-4 h-4" />
            <span>Confirmar Check-in de Hoje</span>
          </button>
        </div>
      )}
    </div>
  );
}
