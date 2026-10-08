"use client";

import React, { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  Dumbbell, CreditCard, QrCode, CheckCircle2, Clock,
  Play, Pause, RotateCcw, Copy, ExternalLink,
  AlertCircle, Sparkles, Check, Flame, Timer, Layers, Wallet, Minus, Plus,
  Video, Trophy, Award, TrendingUp, Calendar, User, ChevronRight, ChevronLeft,
  X, CheckCheck, Heart, Activity, ArrowRight, Share2, Info, Droplets, Shuffle,
  Music, BarChart2, ShieldCheck, Download
} from "lucide-react";
import { api, clearToken } from "@/services/api";
import { Invoice, Workout, WorkoutPlan, Exercise, Booking, ClassSession, PhysicalAssessment } from "@/types";
import QrScanner from "@/components/QrScanner";
import { Modal } from "@/components/ui";
import { brl, dateTimeBR, timeBR, isoDate, monthKey, monthLabel, WEEKDAYS } from "@/lib/format";
import { Avatar, Empty, Notice, Ring, WeekStrip, Bars } from "@/components/ui";
import { IMG } from "@/lib/images";
import { useToast } from "@/context/ToastContext";

export default function AlunoPage() {
  const router = useRouter();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"workout" | "agenda" | "finance" | "checkin" | "progress" | "profile">("workout");

  // Dados do Aluno
  const [planData, setPlanData] = useState<{ has_plan: boolean; student: any; plan: WorkoutPlan | null } | null>(null);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [selectedWorkoutIndex, setSelectedWorkoutIndex] = useState(0);
  const [allExercises, setAllExercises] = useState<Exercise[]>([]);

  // Execução do Treino & Cargas
  const [completedItems, setCompletedItems] = useState<Record<string, boolean>>({});
  const [itemLoads, setItemLoads] = useState<Record<string, number>>({});

  // Cronômetro de Descanso
  const [timerSeconds, setTimerSeconds] = useState(60);
  const [timerTotal, setTimerTotal] = useState(60);
  const [timerActive, setTimerActive] = useState(false);
  const [timerVisible, setTimerVisible] = useState(false);

  // Modo Treino em Andamento (Focus Mode)
  const [inFocusMode, setInFocusMode] = useState(false);
  const [focusItemIndex, setFocusItemIndex] = useState(0);
  const [sessionSeconds, setSessionSeconds] = useState(0);
  const [sessionActive, setSessionActive] = useState(false);
  const [setLogs, setSetLogs] = useState<Record<string, { done: boolean; reps: string; load: number }[]>>({});
  const [showFinishSummary, setShowFinishSummary] = useState(false);
  const [rpeScore, setRpeScore] = useState<number>(8);

  // Modal de Vídeo / Instruções
  const [viewExercise, setViewExercise] = useState<Exercise | null>(null);

  // Modal de Substituição de Aparelho Ocupado
  const [swapModalItem, setSwapModalItem] = useState<{ itemId: string; currentExercise: Exercise } | null>(null);

  // Modal de Calculadora de 1RM & Gráfico de Força
  const [show1RMModal, setShow1RMModal] = useState(false);
  const [oneRmExercise, setOneRmExercise] = useState("Supino Reto com Barra");
  const [oneRmWeight, setOneRmWeight] = useState(80);
  const [oneRmReps, setOneRmReps] = useState(6);
  const [prRecords, setPrRecords] = useState<{ exercise: string; pr: number; date: string }[]>([]);

  // Widget de Hidratação
  const [waterMl, setWaterMl] = useState(0);
  const waterTarget = 2500;

  // Modal de Edição de Medidas no Perfil
  const [editWeight, setEditWeight] = useState<number>(75);
  const [editHeight, setEditHeight] = useState<number>(178);
  const [editNotes, setEditNotes] = useState<string>("");
  const [savingProfile, setSavingProfile] = useState(false);

  // Avaliação Física & Medidas da Aluna
  const [myAssessments, setMyAssessments] = useState<PhysicalAssessment[]>([]);
  const [showMyMeasurementModal, setShowMyMeasurementModal] = useState(false);
  const [myMeasureForm, setMyMeasureForm] = useState({
    weight: 70,
    height: 170,
    body_fat: 0,
    chest: 0,
    waist: 0,
    abdomen: 0,
    hips: 0,
    right_arm: 0,
    right_thigh: 0,
    notes: "",
  });
  const [savingMyMeasure, setSavingMyMeasure] = useState(false);

  // Ações Financeiras
  const [copiedPix, setCopiedPix] = useState(false);
  const [payingInvoiceId, setPayingInvoiceId] = useState<string | null>(null);

  // Check-in
  const [checkInStatus, setCheckInStatus] = useState<string | null>(null);
  const [checkInError, setCheckInError] = useState<string | null>(null);
  const [freq, setFreq] = useState<{ month: string; days: number[]; month_count: number; last30_count: number; week_count: number; streak_weeks: number } | null>(null);
  const [prevMonthDays, setPrevMonthDays] = useState<number[]>([]);
  const [code, setCode] = useState("");
  const [scanning, setScanning] = useState(false);
  const [checkingIn, setCheckingIn] = useState(false);

  // Agenda de aulas
  const [sessions, setSessions] = useState<ClassSession[]>([]);
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [agendaLoading, setAgendaLoading] = useState(false);
  const [busySession, setBusySession] = useState<string | null>(null);

  // Boleto, troca de senha e exclusão de conta
  const [boletoHtml, setBoletoHtml] = useState<string | null>(null);
  const [pw, setPw] = useState({ current: "", next: "" });
  const [showDelete, setShowDelete] = useState(false);
  const [deletePw, setDeletePw] = useState("");

  const loadStudentData = async () => {
    try {
      setLoading(true);
      const [workoutsRes, invoicesRes, exRes] = await Promise.all([
        api.getMyWorkouts(),
        api.getMyInvoices(),
        api.listExercises(""),
      ]);

      setPlanData(workoutsRes);
      setInvoices(invoicesRes);
      setAllExercises(exRes.data);
      api.myCheckIns().then(setFreq).catch(() => {});
      api.myCheckIns(monthKey(-1)).then((r) => setPrevMonthDays(r.days)).catch(() => {});
      api.getMyAssessments().then((r) => {
        if (r.assessments && r.assessments.length > 0) {
          setMyAssessments(r.assessments);
          const latest = r.assessments[0];
          if (latest.weight) setEditWeight(latest.weight);
          if (latest.height) setEditHeight(latest.height);
        }
      }).catch(() => {});

      if (workoutsRes.student) {
        setEditWeight((prev) => workoutsRes.student.weight || prev || 75);
        setEditHeight((prev) => workoutsRes.student.height || prev || 178);
        setEditNotes(workoutsRes.student.notes || "");
      }

      if (workoutsRes.plan?.workouts) {
        const loads: Record<string, number> = {};
        const logs: Record<string, { done: boolean; reps: string; load: number }[]> = {};

        workoutsRes.plan.workouts.forEach((w) => {
          w.items.forEach((it) => {
            loads[it.id] = it.load_kg;
            const totalSets = it.sets || 3;
            logs[it.id] = Array.from({ length: totalSets }, () => ({
              done: false,
              reps: it.reps || "10-12",
              load: it.load_kg || 20,
            }));
          });
        });
        setItemLoads(loads);
        setSetLogs(logs);
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
    // Links das notificações: /aluno?tab=agenda
    const t = new URLSearchParams(window.location.search).get("tab");
    const map: Record<string, typeof activeTab> = { agenda: "agenda", perfil: "profile", inicio: "workout", treino: "workout", faturas: "finance" };
    if (t && map[t]) setActiveTab(map[t]);
    try {
      const w = JSON.parse(localStorage.getItem("unika_water") || "null");
      if (w?.date === isoDate()) setWaterMl(w.ml);
      setPrRecords(JSON.parse(localStorage.getItem("unika_prs") || "[]"));
    } catch {
      /* armazenamento indisponível */
    }
  }, []);

  const loadAgenda = useCallback(async () => {
    setAgendaLoading(true);
    try {
      const [list, mine] = await Promise.all([api.listSessions({ from: isoDate(), days: 8 }), api.myBookings()]);
      setSessions(list);
      setBookings(mine);
    } catch (err: any) {
      toast.error(err.message || "Não foi possível carregar a agenda");
    } finally {
      setAgendaLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    if (activeTab === "agenda") loadAgenda();
  }, [activeTab, loadAgenda]);

  const handleBook = async (s: ClassSession) => {
    setBusySession(s.id);
    try {
      const r = await api.book(s.id);
      toast.success(r.message);
      await loadAgenda();
    } catch (err: any) {
      toast.error(err.message);
    } finally {
      setBusySession(null);
    }
  };
  const handleCancelBooking = async (s: ClassSession) => {
    setBusySession(s.id);
    try {
      const r = await api.cancelBooking(s.id);
      toast.success(r.message || "Reserva cancelada");
      await loadAgenda();
    } catch (err: any) {
      toast.error(err.message);
    } finally {
      setBusySession(null);
    }
  };

  // Cronômetro de Descanso
  useEffect(() => {
    let interval: any = null;
    if (timerActive && timerSeconds > 0) {
      interval = setInterval(() => {
        setTimerSeconds((prev) => prev - 1);
      }, 1000);
    } else if (timerSeconds === 0 && timerActive) {
      setTimerActive(false);
      toast.success("Descanso finalizado! Próxima série!");
      if (typeof window !== "undefined" && window.AudioContext) {
        try {
          const ctx = new AudioContext();
          const osc = ctx.createOscillator();
          osc.type = "sine";
          osc.frequency.setValueAtTime(880, ctx.currentTime);
          osc.connect(ctx.destination);
          osc.start();
          osc.stop(ctx.currentTime + 0.35);
        } catch {}
      }
    }
    return () => clearInterval(interval);
  }, [timerActive, timerSeconds]);

  // Cronômetro da Sessão de Treino
  useEffect(() => {
    let interval: any = null;
    if (sessionActive) {
      interval = setInterval(() => {
        setSessionSeconds((prev) => prev + 1);
      }, 1000);
    }
    return () => clearInterval(interval);
  }, [sessionActive]);

  const startRestTimer = (seconds: number) => {
    setTimerTotal(seconds);
    setTimerSeconds(seconds);
    setTimerActive(true);
    setTimerVisible(true);
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
      toast.success("Exercício concluído! Descanso iniciado.");
    }
  };

  // Focus Mode Actions
  const startFocusSession = () => {
    setInFocusMode(true);
    setFocusItemIndex(0);
    setSessionActive(true);
    toast.success("Sessão de treino iniciada! Bom treino!");
  };

  const toggleSetComplete = (itemId: string, setIdx: number, restSecs: number) => {
    const curSets = setLogs[itemId] || [];
    const updated = [...curSets];
    if (updated[setIdx]) {
      updated[setIdx].done = !updated[setIdx].done;
      setSetLogs((prev) => ({ ...prev, [itemId]: updated }));

      if (updated[setIdx].done) {
        startRestTimer(restSecs || 60);
      }

      const allDone = updated.every((s) => s.done);
      if (allDone) {
        setCompletedItems((prev) => ({ ...prev, [itemId]: true }));
      }
    }
  };

  const updateSetLoad = (itemId: string, setIdx: number, newLoad: number) => {
    const curSets = setLogs[itemId] || [];
    const updated = [...curSets];
    if (updated[setIdx]) {
      updated[setIdx].load = newLoad;
      setSetLogs((prev) => ({ ...prev, [itemId]: updated }));
      handleUpdateLoad(itemId, newLoad);
    }
  };

  const finishSession = () => {
    setSessionActive(false);
    setShowFinishSummary(true);
  };

  const handleSwapExercise = (alternative: Exercise) => {
    if (!swapModalItem || !planData?.plan) return;
    const currentWorkout = planData.plan.workouts[selectedWorkoutIndex];
    if (!currentWorkout) return;

    const item = currentWorkout.items.find((it) => it.id === swapModalItem.itemId);
    if (item) {
      item.exercise = alternative;
      item.exercise_id = alternative.id;
      toast.success(`Aparelho substituído por "${alternative.name}" com sucesso!`);
    }
    setSwapModalItem(null);
  };

  const handleAddWater = (ml: number) => {
    setWaterMl((prev) => {
      const next = Math.min(waterTarget + 1000, prev + ml);
      try { localStorage.setItem("unika_water", JSON.stringify({ date: isoDate(), ml: next })); } catch { /* sem armazenamento */ }
      return next;
    });
    toast.success(`+${ml}ml de água registrados! Hidratação em dia.`);
  };

  // Cálculo 1RM (Fórmula de Brzycki)
  const calculated1RM = Math.round(oneRmWeight * (36 / (37 - Math.min(10, oneRmReps))));

  const handleSavePr = () => {
    setPrRecords((prev) => {
      const next = [{ exercise: oneRmExercise, pr: calculated1RM, date: new Date().toLocaleDateString("pt-BR", { day: "2-digit", month: "short" }) }, ...prev.slice(0, 5)];
      try { localStorage.setItem("unika_prs", JSON.stringify(next)); } catch { /* sem armazenamento */ }
      return next;
    });
    toast.success(`Novo recorde de ${calculated1RM}kg registrado em ${oneRmExercise}! 🏆`);
  };

  const handleCopyPix = (pixCode: string) => {
    navigator.clipboard.writeText(pixCode);
    setCopiedPix(true);
    toast.success("Código PIX copiado para a área de transferência!");
    setTimeout(() => setCopiedPix(false), 3000);
  };

  const handleBoleto = async (invId: string) => {
    try {
      setBoletoHtml(await api.getBoletoHtml(invId));
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  const handleSimulatePayment = async (invId: string) => {
    setPayingInvoiceId(invId);
    try {
      const res = await api.mockPay(invId);
      toast.success(res.message || "Pagamento confirmado com sucesso!");
      loadStudentData();
    } catch (err: any) {
      toast.error("Falha na simulação: " + err.message);
    } finally {
      setPayingInvoiceId(null);
    }
  };

  const handleCheckIn = async (value: string) => {
    setCheckInStatus(null);
    setCheckInError(null);
    setCheckingIn(true);
    try {
      const res = await api.doCheckIn(value);
      setCheckInStatus(res.message);
      toast[res.already ? "info" : "success"](res.message);
      setCode("");
      api.myCheckIns().then(setFreq).catch(() => {});
    } catch (err: any) {
      const msg = err.message || "Acesso não liberado";
      setCheckInError(msg);
      toast.error(msg);
    } finally {
      setCheckingIn(false);
    }
  };

  const onScanned = useCallback((value: string) => {
    setScanning(false);
    handleCheckIn(value);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSaveProfile = async () => {
    if (!planData?.student) return;
    setSavingProfile(true);
    try {
      await api.updateMyProfile({ weight: editWeight, height: editHeight, notes: editNotes });
      toast.success("Perfil e medidas físicas atualizados!");
      loadStudentData();
    } catch (err: any) {
      toast.error("Erro ao salvar perfil: " + err.message);
    } finally {
      setSavingProfile(false);
    }
  };

  const handleExportLGPD = async () => {
    try {
      toast.success("Gerando arquivo de portabilidade...");
      const data = await api.exportMyData();
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `portabilidade-gymsync-${planData?.student?.user?.name?.toLowerCase().replace(/\s+/g, "-") || "aluno"}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success("Download do arquivo de dados concluído!");
    } catch (err: any) {
      toast.error(err.message || "Erro ao exportar dados.");
    }
  };

  const handleSaveMyMeasurement = async () => {
    setSavingMyMeasure(true);
    try {
      const res = await api.createMyAssessment(myMeasureForm);
      toast.success("Nova medição registrada com sucesso!");
      setMyAssessments([res, ...myAssessments]);
      if (res.weight) setEditWeight(res.weight);
      if (res.height) setEditHeight(res.height);
      setShowMyMeasurementModal(false);
    } catch (err: any) {
      toast.error(err.message || "Erro ao salvar medição");
    } finally {
      setSavingMyMeasure(false);
    }
  };

  const activeWorkout: Workout | undefined = planData?.plan?.workouts?.[selectedWorkoutIndex];
  const doneCount = activeWorkout ? activeWorkout.items.filter((it) => completedItems[it.id]).length : 0;
  const totalCount = activeWorkout?.items.length || 0;
  const progressPct = totalCount ? Math.round((doneCount / totalCount) * 100) : 0;
  const totalSets = activeWorkout?.items.reduce((s, it) => s + (it.sets || 0), 0) || 0;
  const estMinutes = activeWorkout
    ? Math.round(activeWorkout.items.reduce((s, it) => s + it.sets * (45 + (it.rest_seconds || 60)), 0) / 60)
    : 0;

  const totalVolumeKg = activeWorkout
    ? activeWorkout.items.reduce((acc, it) => {
        const itemSetLogs = setLogs[it.id] || [];
        const itemVol = itemSetLogs.reduce((sAcc, s) => {
          const repsNum = parseInt(s.reps) || 10;
          return sAcc + (s.done ? s.load * repsNum : 0);
        }, 0);
        return acc + itemVol;
      }, 0)
    : 0;

  const openInvoices = invoices.filter((i) => i.status !== "PAGO");
  const openTotal = openInvoices.reduce((s, i) => s + (i.amount - i.discount), 0);
  const firstName = planData?.student?.user?.name?.split(" ")[0] || "Atleta";
  const fmtTime = (s: number) => `${Math.floor(s / 60).toString().padStart(2, "0")}:${(s % 60).toString().padStart(2, "0")}`;

  const heightM = (editHeight || 178) / 100;
  const imc = heightM > 0 ? (editWeight / (heightM * heightM)).toFixed(1) : "22.5";

  return (
    <div className="flex-1 max-w-xl w-full mx-auto px-4 pt-6 pb-36 space-y-6">
      {/* Cabeçalho estilo App */}
      <div className="flex items-center justify-between animate-fade-up">
        <div className="flex items-center gap-3">
          <button onClick={() => setActiveTab("profile")} className="text-left group">
            <Avatar name={planData?.student?.user?.name} size={52} />
          </button>
          <div>
            <div className="text-lg font-semibold text-white leading-tight flex items-center gap-2">
              Olá, {firstName} 👋
            </div>
            <div className="text-sm text-ink-300">Pronto para treinar hoje?</div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {planData?.student && (
            <span className={planData.student.status === "ATIVO" ? "badge-ok" : "badge-bad"}>
              <span className={`w-1.5 h-1.5 rounded-full ${planData.student.status === "ATIVO" ? "bg-lime" : "bg-[#FF7A5C]"}`} />
              {planData.student.status}
            </span>
          )}
        </div>
      </div>

      <WeekStrip />

      {loading && !planData && (
        <div className="space-y-3">
          <div className="h-56 rounded-4xl bg-ink-800 animate-pulse" />
          <div className="h-24 rounded-3xl bg-ink-800 animate-pulse" />
          <div className="h-24 rounded-3xl bg-ink-800 animate-pulse" />
        </div>
      )}

      {/* ABA: FICHA DE TREINO */}
      {activeTab === "workout" && planData && (
        <div className="space-y-6">
          {planData.has_plan && planData.plan ? (
            <>
              {/* Card Hero do Treino do Dia */}
              <div className="relative h-64 rounded-4xl overflow-hidden animate-fade-up group">
                <img src={IMG.kettlebell} alt="" className="absolute inset-0 w-full h-full object-cover transition-transform duration-700 group-hover:scale-105" />
                <div className="absolute inset-0 bg-gradient-to-t from-black via-black/50 to-transparent" />
                <div className="absolute top-4 left-4 flex items-center gap-2">
                  <span className="badge bg-black/50 backdrop-blur text-white">{planData.plan.title}</span>
                  <span className="badge bg-lime/20 text-lime font-semibold">Ativo</span>
                </div>

                <div className="absolute bottom-0 inset-x-0 p-5 flex items-end justify-between gap-4">
                  <div>
                    <div className="text-xs text-lime font-semibold mb-1">Treino {activeWorkout?.letter}</div>
                    <div className="text-2xl font-semibold text-white leading-tight">{activeWorkout?.name}</div>
                    <div className="text-xs text-ink-200 mt-1.5 flex items-center gap-3">
                      <span className="flex items-center gap-1"><Layers className="w-3.5 h-3.5" /> {totalCount} exercícios</span>
                      <span className="flex items-center gap-1"><Timer className="w-3.5 h-3.5" /> ~{estMinutes} min</span>
                    </div>
                  </div>
                  <div className="shrink-0 p-1 rounded-full bg-black/50 backdrop-blur">
                    <Ring value={progressPct} size={68} stroke={6}>
                      <span className="text-sm font-semibold text-white">{progressPct}%</span>
                    </Ring>
                  </div>
                </div>
              </div>

              {/* Botão de Iniciar Treino em Andamento (Focus Mode) */}
              <div className="animate-fade-up">
                <button
                  onClick={startFocusSession}
                  className="btn-lime w-full !py-4 text-base font-bold shadow-glow flex items-center justify-center gap-3 group"
                >
                  <Play className="w-5 h-5 fill-current transition-transform group-hover:scale-110" />
                  <span>Iniciar Sessão de Treino Guiada</span>
                </button>
              </div>

              {/* Seletor de Divisões (A, B, C...) */}
              <div className="flex items-center gap-2 overflow-x-auto no-scrollbar -mx-4 px-4">
                {planData.plan.workouts.map((w, idx) => (
                  <button
                    key={w.id}
                    onClick={() => setSelectedWorkoutIndex(idx)}
                    className={selectedWorkoutIndex === idx ? "pill-on" : "pill-off"}
                  >
                    <span className={`w-6 h-6 -ml-2 rounded-full flex items-center justify-center text-xs font-bold ${selectedWorkoutIndex === idx ? "bg-black text-lime" : "bg-ink-600 text-white"}`}>
                      {w.letter}
                    </span>
                    {w.name}
                  </button>
                ))}
              </div>

              {/* Mini Métricas de Desempenho */}
              <div className="grid grid-cols-3 gap-2">
                {[
                  { k: "Concluídos", v: `${doneCount}/${totalCount}`, icon: <CheckCircle2 className="w-4 h-4" /> },
                  { k: "Séries", v: totalSets, icon: <Layers className="w-4 h-4" /> },
                  { k: "Tempo Est.", v: `${estMinutes}m`, icon: <Clock className="w-4 h-4" /> },
                ].map((m) => (
                  <div key={m.k} className="card p-3.5">
                    <div className="flex items-center gap-1.5 text-lime text-[11px] font-medium">
                      {m.icon}
                      <span className="text-ink-300">{m.k}</span>
                    </div>
                    <div className="text-xl font-semibold text-white mt-1.5">{m.v}</div>
                  </div>
                ))}
              </div>

              {/* Lista de Exercícios */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <h3 className="section-title">Exercícios da Ficha</h3>
                  <button
                    onClick={() => setShow1RMModal(true)}
                    className="text-xs text-lime flex items-center gap-1 hover:underline font-semibold"
                  >
                    <BarChart2 className="w-3.5 h-3.5" /> Calculadora 1RM
                  </button>
                </div>

                {activeWorkout ? (
                  <div className="space-y-3">
                    {activeWorkout.items.map((it, idx) => {
                      const isDone = completedItems[it.id];
                      const currentLoad = itemLoads[it.id] ?? it.load_kg;
                      const setsDetail = setLogs[it.id] || [];

                      return (
                        <div
                          key={it.id}
                          className={`rounded-3xl p-4 transition-all animate-fade-up ${isDone ? "bg-lime text-black shadow-glow" : "bg-ink-800 border border-white/[0.04]"}`}
                          style={{ animationDelay: `${idx * 40}ms` }}
                        >
                          <div className="flex items-center gap-3">
                            <div className={`w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 text-base font-bold ${isDone ? "bg-black/10" : "bg-ink-700 text-lime"}`}>
                              {String(idx + 1).padStart(2, "0")}
                            </div>

                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <span className={`text-[15px] font-semibold truncate ${isDone ? "text-black" : "text-white"}`}>
                                  {it.exercise?.name}
                                </span>
                                {it.exercise?.video_url && (
                                  <button
                                    onClick={() => it.exercise && setViewExercise(it.exercise)}
                                    className={`p-1 rounded-full ${isDone ? "bg-black/10 text-black" : "bg-ink-700 text-lime hover:bg-lime hover:text-black"} transition-colors`}
                                    title="Assistir execução"
                                  >
                                    <Video className="w-3.5 h-3.5" />
                                  </button>
                                )}
                              </div>
                              <div className={`text-xs mt-0.5 truncate ${isDone ? "text-black/60" : "text-ink-300"}`}>
                                {it.sets} séries × {it.reps} · {it.exercise?.muscle_group}
                              </div>
                            </div>

                            {/* Botão de Aparelho Ocupado / Substituição */}
                            <button
                              onClick={() => it.exercise && setSwapModalItem({ itemId: it.id, currentExercise: it.exercise })}
                              className={`p-2 rounded-xl text-xs font-semibold flex items-center gap-1 ${isDone ? "bg-black/10 text-black" : "bg-ink-750 text-ink-300 hover:text-white"}`}
                              title="Aparelho Ocupado? Trocar exercício"
                            >
                              <Shuffle className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={() => toggleItemComplete(it.id, it.rest_seconds)}
                              className={`btn-icon shrink-0 ${isDone ? "bg-black text-lime" : "bg-ink-700 text-ink-300 hover:bg-lime hover:text-black"}`}
                              title={isDone ? "Desmarcar" : "Concluir exercício"}
                            >
                              <Check className="w-5 h-5" strokeWidth={2.6} />
                            </button>
                          </div>

                          {/* Ajuste de Carga e Séries Rápidas */}
                          <div className="flex items-center gap-2 mt-3.5">
                            <div className={`flex-1 flex items-center justify-between rounded-full p-1 ${isDone ? "bg-black/10" : "bg-ink-750"}`}>
                              <button
                                onClick={() => handleUpdateLoad(it.id, Math.max(0, currentLoad - 2.5))}
                                className={`w-8 h-8 rounded-full flex items-center justify-center ${isDone ? "hover:bg-black/10" : "hover:bg-ink-600 text-ink-200"}`}
                              >
                                <Minus className="w-3.5 h-3.5" />
                              </button>
                              <div className="flex items-baseline gap-1">
                                <input
                                  type="number"
                                  value={currentLoad}
                                  onChange={(e) => handleUpdateLoad(it.id, parseFloat(e.target.value) || 0)}
                                  className={`w-12 bg-transparent font-semibold text-center focus:outline-none ${isDone ? "text-black" : "text-white"}`}
                                />
                                <span className={`text-[11px] ${isDone ? "text-black/60" : "text-ink-300"}`}>kg</span>
                              </div>
                              <button
                                onClick={() => handleUpdateLoad(it.id, currentLoad + 2.5)}
                                className={`w-8 h-8 rounded-full flex items-center justify-center ${isDone ? "hover:bg-black/10" : "hover:bg-ink-600 text-ink-200"}`}
                              >
                                <Plus className="w-3.5 h-3.5" />
                              </button>
                            </div>

                            <button
                              onClick={() => startRestTimer(it.rest_seconds || 60)}
                              className={`flex items-center gap-1.5 pl-1 pr-3.5 py-1 rounded-full text-xs font-semibold ${isDone ? "bg-black text-white" : "bg-ink-750 text-white hover:bg-ink-700"}`}
                            >
                              <span className="w-8 h-8 rounded-full bg-lime text-black flex items-center justify-center">
                                <Play className="w-3.5 h-3.5 fill-current" />
                              </span>
                              {it.rest_seconds}s
                            </button>
                          </div>

                          {/* Chips de Séries Individuais */}
                          {setsDetail.length > 0 && (
                            <div className="flex items-center gap-2 mt-3 pt-3 border-t border-black/5">
                              <span className={`text-[10px] font-semibold uppercase tracking-wider ${isDone ? "text-black/60" : "text-ink-400"}`}>Séries:</span>
                              <div className="flex flex-wrap gap-1.5">
                                {setsDetail.map((s, sIdx) => (
                                  <button
                                    key={sIdx}
                                    onClick={() => toggleSetComplete(it.id, sIdx, it.rest_seconds)}
                                    className={`px-2.5 py-1 rounded-full text-[11px] font-bold flex items-center gap-1 transition-all ${
                                      s.done
                                        ? isDone ? "bg-black text-lime" : "bg-lime text-black"
                                        : isDone ? "bg-black/15 text-black" : "bg-ink-700 text-ink-300 hover:text-white"
                                    }`}
                                  >
                                    <span>#{sIdx + 1}</span>
                                    {s.done && <Check className="w-3 h-3 stroke-[3]" />}
                                  </button>
                                ))}
                              </div>
                            </div>
                          )}

                          {it.notes && (
                            <div className={`text-xs mt-2.5 px-3 py-2 rounded-2xl ${isDone ? "bg-black/10 text-black/70" : "bg-ink-750 text-ink-200"}`}>
                              💡 {it.notes}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="card">
                    <Empty>Nenhuma divisão selecionada.</Empty>
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="card">
              <Empty icon={<Dumbbell className="w-6 h-6" />} title="Nenhum treino prescrito ainda">
                Seu professor ou personal ainda não configurou sua ficha. Fale com ele na recepção ou aguarde a prescrição!
              </Empty>
            </div>
          )}
        </div>
      )}

      {/* ABA: PROGRESSO, CALENDÁRIO, HIDRATAÇÃO & CONQUISTAS */}
      {activeTab === "progress" && planData && (
        <div className="space-y-6 animate-fade-up">
          {/* Widget de Hidratação Diária */}
          <div className="card p-5 space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-2xl bg-blue-500/15 text-blue-400 flex items-center justify-center">
                  <Droplets className="w-5 h-5" />
                </div>
                <div>
                  <div className="text-xs text-ink-300">Hidratação Hoje</div>
                  <div className="text-lg font-bold text-white">
                    {waterMl} / {waterTarget} ml
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleAddWater(250)}
                  className="btn bg-ink-750 hover:bg-blue-500/20 text-blue-400 !px-3 !py-1.5 text-xs font-semibold"
                >
                  +250ml
                </button>
                <button
                  onClick={() => handleAddWater(500)}
                  className="btn bg-ink-750 hover:bg-blue-500/20 text-blue-400 !px-3 !py-1.5 text-xs font-semibold"
                >
                  +500ml
                </button>
              </div>
            </div>

            {/* Barra de Água */}
            <div className="h-2.5 rounded-full bg-white/5 overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-blue-500 to-cyan-400 rounded-full transition-all duration-500"
                style={{ width: `${Math.min(100, Math.round((waterMl / waterTarget) * 100))}%` }}
              />
            </div>
          </div>

          {/* Card de Frequência & Streak */}
          <div className="card p-6 space-y-5">
            <div className="flex items-center justify-between pb-4 border-b border-white/5">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-full bg-[#FF5A36]/20 flex items-center justify-center">
                  <Flame className="w-6 h-6 text-[#FF5A36]" fill="#FF5A36" />
                </div>
                <div>
                  <div className="eyebrow">Sequência ativa</div>
                  <div className="text-2xl font-bold text-white">
                    {freq?.streak_weeks ?? 0} <span className="text-sm font-normal text-ink-300">{freq?.streak_weeks === 1 ? "semana seguida" : "semanas seguidas"}</span>
                  </div>
                </div>
              </div>
              {(freq?.streak_weeks ?? 0) >= 2 && <span className="badge-ok">Em chamas 🔥</span>}
            </div>

            {/* Calendário do mês com as presenças reais */}
            {(() => {
              const now = new Date();
              const first = new Date(now.getFullYear(), now.getMonth(), 1);
              const daysInMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
              const present = freq?.days || [];
              return (
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <h4 className="text-sm font-semibold text-white">{monthLabel(monthKey())}</h4>
                    <span className="text-xs text-lime font-semibold">{freq?.month_count ?? 0} dia(s) de treino</span>
                  </div>
                  <div className="grid grid-cols-7 gap-1.5 text-center text-xs">
                    {["D", "S", "T", "Q", "Q", "S", "S"].map((d, i) => (
                      <span key={i} className="text-ink-400 font-semibold py-1">{d}</span>
                    ))}
                    {Array.from({ length: first.getDay() }, (_, i) => <span key={`b${i}`} />)}
                    {Array.from({ length: daysInMonth }, (_, i) => {
                      const day = i + 1;
                      const isPresent = present.includes(day);
                      const isToday = day === now.getDate();
                      return (
                        <div key={day} className={`h-9 rounded-xl flex items-center justify-center font-semibold text-xs transition-all ${isPresent ? "bg-lime text-black font-bold shadow-glow" : isToday ? "border border-lime text-white" : "bg-ink-750 text-ink-300"}`}>
                          {day}
                        </div>
                      );
                    })}
                  </div>
                </div>
              );
            })()}
          </div>

          {/* Medidas Físicas, Composição e Histórico */}
          <div className="card p-6 space-y-5">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="section-title flex items-center gap-2">
                  <TrendingUp className="w-4 h-4 text-lime" /> Evolução Corporal & Medidas
                </h3>
                <p className="text-xs text-ink-300 mt-0.5">Acompanhamento contínuo dos seus resultados</p>
              </div>
              <button
                onClick={() => {
                  setMyMeasureForm((prev) => ({
                    ...prev,
                    weight: editWeight || 70,
                    height: editHeight || 170,
                  }));
                  setShowMyMeasurementModal(true);
                }}
                className="btn-lime !py-1.5 !px-3.5 text-xs font-bold shadow-glow"
              >
                + Nova Medição
              </button>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div className="p-3.5 rounded-2xl bg-ink-750">
                <div className="text-xs text-ink-300">Peso Atual</div>
                <div className="text-xl font-bold text-white mt-1">{editWeight} kg</div>
                {myAssessments.length >= 2 && myAssessments[0].weight > 0 && myAssessments[1].weight > 0 && (
                  <div
                    className={`text-[11px] font-semibold mt-1 ${
                      myAssessments[0].weight <= myAssessments[1].weight ? "text-emerald-400" : "text-amber-400"
                    }`}
                  >
                    {(myAssessments[0].weight - myAssessments[1].weight).toFixed(1)} kg vs anterior
                  </div>
                )}
              </div>
              <div className="p-3.5 rounded-2xl bg-ink-750">
                <div className="text-xs text-ink-300">Altura</div>
                <div className="text-xl font-bold text-white mt-1">{editHeight} cm</div>
              </div>
              <div className="p-3.5 rounded-2xl bg-ink-750">
                <div className="text-xs text-ink-300">IMC Estimado</div>
                <div className="text-xl font-bold text-lime mt-1">{imc}</div>
              </div>
              <div className="p-3.5 rounded-2xl bg-ink-750">
                <div className="text-xs text-ink-300">% Gordura</div>
                <div className="text-xl font-bold text-cyan-400 mt-1">
                  {myAssessments[0]?.body_fat ? `${myAssessments[0].body_fat}%` : "—"}
                </div>
              </div>
            </div>

            {/* Histórico Recente de Medidas */}
            {myAssessments.length > 0 && (
              <div className="space-y-3 pt-3 border-t border-white/5">
                <div className="text-xs font-semibold text-ink-300 uppercase tracking-wider">
                  Histórico de Avaliações ({myAssessments.length})
                </div>
                <div className="space-y-2">
                  {myAssessments.slice(0, 3).map((a, idx) => (
                    <div
                      key={a.id}
                      className="p-3 rounded-2xl bg-ink-800/80 border border-white/5 flex flex-wrap items-center justify-between gap-3 text-xs"
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-white">
                          {new Date(a.date).toLocaleDateString("pt-BR")}
                        </span>
                        {idx === 0 && <span className="badge-ok text-[10px]">Mais recente</span>}
                      </div>
                      <div className="flex items-center gap-4 text-ink-300">
                        <span>Peso: <strong className="text-white">{a.weight}kg</strong></span>
                        {a.body_fat > 0 && <span>Gordura: <strong className="text-lime">{a.body_fat}%</strong></span>}
                        {a.waist > 0 && <span>Cintura: <strong className="text-white">{a.waist}cm</strong></span>}
                        {a.chest > 0 && <span>Tórax: <strong className="text-white">{a.chest}cm</strong></span>}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Conquistas / Badges */}
          <div className="card p-6 space-y-4">
            <h3 className="section-title flex items-center gap-2">
              <Trophy className="w-4 h-4 text-amber-400" /> Suas Conquistas
            </h3>

            <div className="grid grid-cols-2 gap-3">
              {[
                { title: "Primeiro Treino", desc: "Completou o 1º treino", unlocked: true, icon: <CheckCircle2 className="w-5 h-5 text-lime" /> },
                { title: "Fogo Constante", desc: "7 dias seguidos", unlocked: true, icon: <Flame className="w-5 h-5 text-[#FF5A36]" /> },
                { title: "10 Toneladas", desc: "10.000kg levantados", unlocked: true, icon: <Award className="w-5 h-5 text-amber-400" /> },
                { title: "Frequência 100%", desc: "Bateu meta do mês", unlocked: false, icon: <Trophy className="w-5 h-5 text-ink-400" /> },
              ].map((b, i) => (
                <div
                  key={i}
                  className={`p-4 rounded-3xl border flex items-center gap-3 ${
                    b.unlocked ? "bg-ink-750 border-white/5 text-white" : "bg-ink-850/60 border-white/5 opacity-40 text-ink-400"
                  }`}
                >
                  <div className="shrink-0">{b.icon}</div>
                  <div>
                    <div className="text-xs font-bold leading-tight">{b.title}</div>
                    <div className="text-[10px] text-ink-300 mt-0.5">{b.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ABA: FATURAS E MENSALIDADES */}
      {activeTab === "finance" && planData && (
        <div className="space-y-5 animate-fade-up">
          <div className={`rounded-4xl p-6 ${openInvoices.length ? "bg-ink-800 border border-white/[0.04]" : "bg-lime text-black"}`}>
            <div className="flex items-center justify-between">
              <span className={`text-sm font-medium flex items-center gap-2 ${openInvoices.length ? "text-ink-200" : "text-black/70"}`}>
                <Wallet className="w-4 h-4" /> Em aberto
              </span>
              <span className={openInvoices.length ? "badge-bad" : "badge bg-black/10 text-black"}>
                {openInvoices.length ? `${openInvoices.length} pendente(s)` : "Tudo em dia"}
              </span>
            </div>
            <div className={`text-4xl font-semibold tracking-tight mt-4 ${openInvoices.length ? "text-white" : "text-black"}`}>
              {brl(openTotal)}
            </div>
            <div className={`text-xs mt-1 ${openInvoices.length ? "text-ink-300" : "text-black/60"}`}>
              {planData.student?.plan_name} · vence todo dia {planData.student?.due_day}
            </div>
          </div>

          <div>
            <h3 className="section-title mb-3">Mensalidades</h3>
            {invoices.length > 0 ? (
              <div className="space-y-2.5">
                {invoices.map((inv) => {
                  const paid = inv.status === "PAGO";
                  return (
                    <div key={inv.id} className="card p-4 space-y-4 animate-fade-up">
                      <div className="flex items-center gap-3">
                        <div className={`w-12 h-12 rounded-2xl flex items-center justify-center ${paid ? "bg-lime/15 text-lime" : "bg-[#FF5A36]/15 text-[#FF7A5C]"}`}>
                          {paid ? <CheckCircle2 className="w-5 h-5" /> : <CreditCard className="w-5 h-5" />}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-semibold text-white truncate">{inv.description}</div>
                          <div className="text-xs text-ink-300">
                            Vencimento {new Date(inv.due_date).toLocaleDateString("pt-BR")}
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="text-base font-semibold text-white">{brl(paid ? inv.paid_amount : inv.amount - inv.discount)}</div>
                          <span className={paid ? "badge-ok" : inv.status === "VENCIDO" ? "badge-bad" : "badge-mute"}>{paid ? "Pago" : inv.status === "VENCIDO" ? "Vencido" : "Em aberto"}</span>
                        </div>
                      </div>

                      {!paid && (
                        <div className="space-y-2">
                          <div className="grid grid-cols-2 gap-2">
                            <button onClick={() => handleBoleto(inv.id)} className="btn-dark !py-2.5 text-xs">
                              <ExternalLink className="w-3.5 h-3.5 text-lime" />
                              Boleto
                            </button>
                            <button onClick={() => handleCopyPix(inv.pix_copy_paste)} className="btn-dark !py-2.5 text-xs">
                              <Copy className="w-3.5 h-3.5 text-lime" />
                              {copiedPix ? "Copiado!" : "Copiar PIX"}
                            </button>
                          </div>

                          {process.env.NODE_ENV !== "production" && (
                            <button
                              onClick={() => handleSimulatePayment(inv.id)}
                              disabled={payingInvoiceId === inv.id}
                              className="btn-lime w-full !py-3 text-xs"
                            >
                              <Sparkles className="w-4 h-4" />
                              {payingInvoiceId === inv.id ? "Confirmando..." : "Simular pagamento (ambiente de teste)"}
                            </button>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="card">
                <Empty icon={<CreditCard className="w-6 h-6" />}>Nenhuma fatura encontrada.</Empty>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ABA: CHECK-IN / ENTRADA NA ACADEMIA */}
      {activeTab === "checkin" && planData && (
        <div className="space-y-4 animate-fade-up">
          <div className="card p-5 animate-fade-up">
            <div className="flex items-center gap-4 pb-5 border-b border-white/5">
              <div className="w-14 h-14 rounded-full bg-[#FF5A36]/15 flex items-center justify-center shadow-[0_0_30px_rgba(255,90,54,0.35)]">
                <Flame className="w-7 h-7 text-[#FF5A36]" fill="#FF5A36" />
              </div>
              <div>
                <div className="eyebrow">Frequência</div>
                <div className="text-2xl font-semibold text-white">Controle de entrada</div>
              </div>
            </div>
            <div className="flex justify-between pt-5">
              {["Seg", "Ter", "Qua", "Qui", "Sex", "Sáb", "Dom"].map((d, i) => {
                const todayIdx = (new Date().getDay() + 6) % 7;
                const isToday = i === todayIdx;
                const dayDate = new Date();
                dayDate.setDate(dayDate.getDate() - todayIdx + i);
                const sameMonth = dayDate.getMonth() === new Date().getMonth();
                const done = (sameMonth ? freq?.days : prevMonthDays)?.includes(dayDate.getDate()) ?? false;
                return (
                  <div key={d} className="flex flex-col items-center gap-2">
                    <div
                      className={`w-10 h-10 rounded-full flex items-center justify-center ${done ? "bg-lime text-black shadow-glow" : isToday ? "border-2 border-lime" : "bg-ink-700"}`}
                    >
                      {done && <Check className="w-5 h-5 stroke-[3]" />}
                    </div>
                    <span className="text-[11px] text-ink-300">{d}</span>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="card p-6 text-center space-y-5 animate-fade-up">
            {scanning ? (
              <QrScanner onScan={onScanned} onCancel={() => setScanning(false)} />
            ) : (
              <>
                <p className="text-sm text-ink-300">Escaneie o QR Code que aparece na tela da recepção. O código muda a cada poucos segundos.</p>
                <button onClick={() => setScanning(true)} className="btn-lime w-full !py-4 font-bold shadow-glow" disabled={checkingIn}>
                  <QrCode className="w-5 h-5" />
                  Escanear QR da recepção
                </button>
                <div className="flex items-center gap-3 text-[11px] text-ink-400"><span className="flex-1 h-px bg-white/10" />ou digite o código<span className="flex-1 h-px bg-white/10" /></div>
                <form onSubmit={(e) => { e.preventDefault(); if (code.length >= 6) handleCheckIn(code); }} className="flex gap-2">
                  <input className="field text-center tracking-[0.4em] font-mono text-lg" inputMode="numeric" maxLength={6} placeholder="000000" value={code} onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))} aria-label="Código de 6 números" />
                  <button type="submit" className="btn-dark px-5" disabled={checkingIn || code.length < 6}>Entrar</button>
                </form>
              </>
            )}

            {checkInStatus && (
              <Notice tone="ok" icon={<CheckCircle2 className="w-4 h-4" />}>{checkInStatus}</Notice>
            )}
            {checkInError && (
              <Notice tone="bad" icon={<AlertCircle className="w-4 h-4" />}>{checkInError}</Notice>
            )}
          </div>
        </div>
      )}

      {/* ABA: AGENDA DE AULAS */}
      {activeTab === "agenda" && (
        <div className="space-y-5 animate-fade-up">
          {bookings.length > 0 && (
            <section className="space-y-2">
              <h3 className="section-title">Minhas reservas</h3>
              {bookings.map((b) => (
                <div key={b.id} className="card p-4 flex items-center gap-3">
                  <span className="w-1.5 self-stretch rounded-full" style={{ background: b.session?.modality?.color || "#D7FF3A" }} />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-semibold text-white truncate">{b.session?.title || b.session?.modality?.name}</div>
                    <div className="text-xs text-ink-300">{dateTimeBR(b.session?.starts_at)}{b.session?.teacher ? ` · ${b.session.teacher.name}` : ""}</div>
                  </div>
                  <span className={b.status === "ESPERA" ? "badge bg-amber-400/15 text-amber-300" : "badge-ok"}>{b.status === "ESPERA" ? "Lista de espera" : b.status === "PRESENTE" ? "Presente" : "Confirmada"}</span>
                </div>
              ))}
            </section>
          )}

          <section className="space-y-3">
            <h3 className="section-title">Próximas aulas</h3>
            {agendaLoading && sessions.length === 0 ? (
              <div className="space-y-2"><div className="h-20 rounded-3xl bg-ink-800 animate-pulse" /><div className="h-20 rounded-3xl bg-ink-800 animate-pulse" /></div>
            ) : sessions.filter((x) => x.status === "AGENDADA" && new Date(x.ends_at) > new Date()).length === 0 ? (
              <div className="card"><Empty icon={<Calendar className="w-6 h-6" />} title="Sem aulas nos próximos dias">A academia ainda não publicou horários.</Empty></div>
            ) : (
              Object.entries(
                sessions.filter((x) => x.status === "AGENDADA" && new Date(x.ends_at) > new Date()).reduce<Record<string, ClassSession[]>>((acc, x) => {
                  (acc[isoDate(new Date(x.starts_at))] ||= []).push(x);
                  return acc;
                }, {}),
              ).map(([day, list]) => (
                <div key={day} className="space-y-2">
                  <div className="eyebrow pt-1">{WEEKDAYS[new Date(day + "T12:00:00").getDay()]}, {new Date(day + "T12:00:00").toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" })}</div>
                  {list.map((c) => {
                    const mine = c.my_status === "CONFIRMADA" || c.my_status === "ESPERA" || c.my_status === "PRESENTE";
                    const full = (c.booked ?? 0) >= c.capacity;
                    const started = new Date(c.starts_at) <= new Date();
                    return (
                      <div key={c.id} className="card p-4 flex items-center gap-3">
                        <span className="w-1.5 self-stretch rounded-full" style={{ background: c.modality?.color || "#D7FF3A" }} />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-semibold text-white truncate">{c.title || c.modality?.name}</div>
                          <div className="text-xs text-ink-300">{timeBR(c.starts_at)}–{timeBR(c.ends_at)}{c.teacher ? ` · ${c.teacher.name}` : ""}</div>
                          <div className="text-[11px] text-ink-400 mt-0.5">{full ? "Lotada" : `${c.capacity - (c.booked ?? 0)} vaga(s)`}{(c.waitlist ?? 0) > 0 ? ` · ${c.waitlist} na espera` : ""}</div>
                        </div>
                        {mine ? (
                          c.my_status === "PRESENTE" ? <span className="badge-ok">Presente</span> :
                          <button className="btn-dark !px-4 !py-2 text-xs" disabled={busySession === c.id || started} onClick={() => handleCancelBooking(c)}>{busySession === c.id ? "…" : c.my_status === "ESPERA" ? "Sair da fila" : "Cancelar"}</button>
                        ) : (
                          <button className="btn-lime !px-4 !py-2 text-xs" disabled={busySession === c.id || started} onClick={() => handleBook(c)}>{busySession === c.id ? "…" : full ? "Entrar na fila" : "Reservar"}</button>
                        )}
                      </div>
                    );
                  })}
                </div>
              ))
            )}
          </section>
        </div>
      )}

      {/* ABA: PERFIL & MEDIDAS */}
      {activeTab === "profile" && planData && (
        <div className="space-y-5 animate-fade-up">
          <div className="card p-6 space-y-5">
            <div className="flex items-center gap-4 pb-4 border-b border-white/5">
              <Avatar name={planData.student?.user?.name} size={64} />
              <div>
                <h3 className="text-lg font-bold text-white">{planData.student?.user?.name}</h3>
                <div className="text-xs text-ink-300">{planData.student?.user?.email}</div>
                <div className="text-xs text-lime mt-1 font-semibold">
                  Plano: {planData.student?.plan_name} (R$ {planData.student?.plan_price?.toFixed(2)}/mês)
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <h4 className="text-sm font-semibold text-white">Medidas Corporais & Foco</h4>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Peso (kg)</label>
                  <input
                    type="number"
                    step="0.1"
                    value={editWeight}
                    onChange={(e) => setEditWeight(parseFloat(e.target.value) || 0)}
                    className="field"
                  />
                </div>
                <div>
                  <label className="label">Altura (cm)</label>
                  <input
                    type="number"
                    value={editHeight}
                    onChange={(e) => setEditHeight(parseFloat(e.target.value) || 0)}
                    className="field"
                  />
                </div>
              </div>

              <div>
                <label className="label">Notas de Saúde / Restrições / Objetivos</label>
                <textarea
                  rows={3}
                  value={editNotes}
                  onChange={(e) => setEditNotes(e.target.value)}
                  placeholder="Ex: Foco em hipertrofia, recuperação de tendinite no ombro direito..."
                  className="field resize-none"
                />
              </div>

              <button
                onClick={handleSaveProfile}
                disabled={savingProfile}
                className="btn-lime w-full !py-3.5"
              >
                {savingProfile ? "Salvando..." : "Salvar Dados do Perfil"}
              </button>
            </div>
          </div>

          <div className="card p-6 space-y-3">
            <h4 className="text-sm font-semibold text-white">Alterar senha</h4>
            <input type="password" autoComplete="current-password" placeholder="Senha atual" className="field" value={pw.current} onChange={(e) => setPw({ ...pw, current: e.target.value })} />
            <input type="password" autoComplete="new-password" placeholder="Nova senha (mínimo 8 caracteres)" className="field" value={pw.next} onChange={(e) => setPw({ ...pw, next: e.target.value })} />
            <button className="btn-dark w-full" disabled={!pw.current || pw.next.length < 8}
              onClick={async () => { try { await api.changePassword(pw.current, pw.next); setPw({ current: "", next: "" }); toast.success("Senha alterada"); } catch (e: any) { toast.error(e.message); } }}>
              Salvar nova senha
            </button>
          </div>

          {/* LGPD & Privacidade (Art. 18 Lei 13.709/2018) */}
          <div className="card p-6 space-y-3.5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="w-9 h-9 rounded-full bg-lime/15 text-lime flex items-center justify-center">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-white">Privacidade & Meus Dados</h4>
                  <p className="text-[11px] text-ink-300">Conforme a LGPD (Lei 13.709/2018)</p>
                </div>
              </div>
              <span className="pill bg-lime/10 text-lime text-[11px] font-bold">Art. 18 LGPD</span>
            </div>

            <p className="text-xs text-ink-200 leading-relaxed">
              Você é dono(a) dos seus dados. É possível baixar uma cópia completa de suas informações (fichas de treino, evolução de medidas, check-ins e financeiro) a qualquer momento.
            </p>

            <div className="flex flex-col sm:flex-row gap-2.5 pt-1">
              <button
                onClick={handleExportLGPD}
                className="btn-dark flex-1 flex items-center justify-center gap-2 !py-3 text-xs text-lime border-lime/30 hover:bg-lime/10 transition-colors"
              >
                <Download className="w-4 h-4" />
                Exportar Meus Dados (JSON)
              </button>
              <Link
                href="/privacidade"
                target="_blank"
                className="btn-dark flex-1 flex items-center justify-center gap-2 !py-3 text-xs text-ink-200 hover:text-white transition-colors"
              >
                <ExternalLink className="w-4 h-4" />
                Política de Privacidade
              </Link>
            </div>
          </div>

          <div className="card p-6 space-y-3">
            <h4 className="text-sm font-semibold text-white">Excluir minha conta</h4>
            <p className="text-xs text-ink-300">Seus dados pessoais são apagados e o acesso encerrado. Reservas futuras são canceladas. Não dá para desfazer.</p>
            <button className="btn bg-[#FF5A36]/15 text-[#FF7A5C] w-full py-3" onClick={() => setShowDelete(true)}>Excluir conta</button>
          </div>
        </div>
      )}

      {/* MODAL: SUBSTITUIÇÃO DE APARELHO OCUPADO */}
      {swapModalItem && (
        <div className="fixed inset-0 z-[80] bg-black/85 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-ink-850 border border-white/10 rounded-4xl max-w-md w-full p-6 space-y-4 shadow-card animate-fade-up">
            <div className="flex items-center justify-between pb-3 border-b border-white/10">
              <div>
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Shuffle className="w-5 h-5 text-lime" /> Aparelho Ocupado?
                </h3>
                <p className="text-xs text-ink-300 mt-0.5">
                  Substitutos para: <b>{swapModalItem.currentExercise.name}</b>
                </p>
              </div>
              <button onClick={() => setSwapModalItem(null)} className="btn-icon w-8 h-8 bg-ink-700 text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-2 max-h-72 overflow-y-auto pr-1">
              {allExercises
                .filter(
                  (ex) =>
                    ex.muscle_group === swapModalItem.currentExercise.muscle_group &&
                    ex.id !== swapModalItem.currentExercise.id
                )
                .slice(0, 6)
                .map((alt) => (
                  <button
                    key={alt.id}
                    onClick={() => handleSwapExercise(alt)}
                    className="w-full text-left p-3 rounded-2xl bg-ink-750 hover:bg-lime hover:text-black transition-all flex items-center justify-between group"
                  >
                    <div>
                      <div className="text-sm font-semibold text-white group-hover:text-black">{alt.name}</div>
                      <div className="text-xs text-ink-300 group-hover:text-black/70">{alt.equipment}</div>
                    </div>
                    <span className="badge-ok group-hover:bg-black group-hover:text-lime">Substituir</span>
                  </button>
                ))}
            </div>

            <button onClick={() => setSwapModalItem(null)} className="btn-dark w-full !py-3 text-xs">
              Cancelar
            </button>
          </div>
        </div>
      )}

      {/* MODAL: CALCULADORA DE 1RM & RECORDES */}
      {show1RMModal && (
        <div className="fixed inset-0 z-[80] bg-black/85 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-ink-850 border border-white/10 rounded-4xl max-w-md w-full p-6 space-y-5 shadow-card animate-fade-up">
            <div className="flex items-center justify-between pb-3 border-b border-white/10">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <BarChart2 className="w-5 h-5 text-lime" /> Calculadora de Força (1RM)
              </h3>
              <button onClick={() => setShow1RMModal(false)} className="btn-icon w-8 h-8 bg-ink-700 text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3">
              <div>
                <label className="label">Exercício</label>
                <select
                  value={oneRmExercise}
                  onChange={(e) => setOneRmExercise(e.target.value)}
                  className="field"
                >
                  <option value="Supino Reto com Barra">Supino Reto com Barra</option>
                  <option value="Agachamento Livre">Agachamento Livre</option>
                  <option value="Levantamento Terra">Levantamento Terra</option>
                  <option value="Leg Press 45°">Leg Press 45°</option>
                  <option value="Desenvolvimento com Halteres">Desenvolvimento com Halteres</option>
                </select>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Carga levantada (kg)</label>
                  <input
                    type="number"
                    value={oneRmWeight}
                    onChange={(e) => setOneRmWeight(parseFloat(e.target.value) || 0)}
                    className="field"
                  />
                </div>
                <div>
                  <label className="label">Repetições feitas</label>
                  <input
                    type="number"
                    min="1"
                    max="12"
                    value={oneRmReps}
                    onChange={(e) => setOneRmReps(parseInt(e.target.value) || 1)}
                    className="field"
                  />
                </div>
              </div>

              {/* Resultado 1RM Estimada */}
              <div className="card p-4 bg-lime text-black flex items-center justify-between rounded-3xl">
                <div>
                  <div className="text-xs font-semibold text-black/70">1RM Estimada (Carga Máxima)</div>
                  <div className="text-3xl font-black">{calculated1RM} kg</div>
                </div>
                <button
                  onClick={handleSavePr}
                  className="btn bg-black text-lime hover:bg-black/80 !px-4 !py-2 text-xs font-bold"
                >
                  Salvar PR 🏆
                </button>
              </div>

              {/* Tabela de Percentuais */}
              <div className="grid grid-cols-4 gap-1.5 text-center text-xs pt-1">
                <div className="p-2 bg-ink-750 rounded-xl">
                  <div className="text-[10px] text-ink-300">95% (2 reps)</div>
                  <div className="font-bold text-white mt-0.5">{Math.round(calculated1RM * 0.95)}kg</div>
                </div>
                <div className="p-2 bg-ink-750 rounded-xl">
                  <div className="text-[10px] text-ink-300">85% (6 reps)</div>
                  <div className="font-bold text-white mt-0.5">{Math.round(calculated1RM * 0.85)}kg</div>
                </div>
                <div className="p-2 bg-ink-750 rounded-xl">
                  <div className="text-[10px] text-ink-300">75% (10 reps)</div>
                  <div className="font-bold text-white mt-0.5">{Math.round(calculated1RM * 0.75)}kg</div>
                </div>
                <div className="p-2 bg-ink-750 rounded-xl">
                  <div className="text-[10px] text-ink-300">65% (15 reps)</div>
                  <div className="font-bold text-white mt-0.5">{Math.round(calculated1RM * 0.65)}kg</div>
                </div>
              </div>

              {/* Histórico de Recordes Salvos */}
              <div className="pt-2">
                <div className="text-xs font-semibold text-ink-300 mb-2">Seus Recordes Pessoais (PRs):</div>
                <div className="space-y-1.5 max-h-32 overflow-y-auto">
                  {prRecords.map((pr, i) => (
                    <div key={i} className="p-2.5 rounded-xl bg-ink-750 flex items-center justify-between text-xs">
                      <span className="text-white font-medium">{pr.exercise}</span>
                      <div className="flex items-center gap-2">
                        <span className="text-lime font-bold">{pr.pr} kg</span>
                        <span className="text-[10px] text-ink-400">{pr.date}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <button onClick={() => setShow1RMModal(false)} className="btn-dark w-full !py-3 text-xs">
              Fechar
            </button>
          </div>
        </div>
      )}

      {/* MODAL FULLSCREEN: MODO TREINO EM ANDAMENTO (FOCUS MODE) */}
      {inFocusMode && activeWorkout && (
        <div className="fixed inset-0 z-[70] bg-ink-950 flex flex-col p-4 sm:p-6 overflow-y-auto animate-fade-up">
          <div className="max-w-xl w-full mx-auto flex items-center justify-between pb-4 border-b border-white/10">
            <div>
              <span className="badge bg-lime text-black font-bold">EM ANDAMENTO</span>
              <h3 className="text-xl font-bold text-white mt-1">Treino {activeWorkout.letter}: {activeWorkout.name}</h3>
            </div>
            <div className="text-right">
              <div className="text-xs text-ink-300 font-medium">Tempo de Sessão</div>
              <div className="text-2xl font-bold font-mono text-lime">{fmtTime(sessionSeconds)}</div>
            </div>
          </div>

          {activeWorkout.items[focusItemIndex] && (
            <div className="max-w-xl w-full mx-auto flex-1 flex flex-col justify-between py-6 space-y-6">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-ink-300">
                    Exercício {focusItemIndex + 1} de {activeWorkout.items.length}
                  </span>
                  <div className="flex items-center gap-3">
                    <button
                      onClick={() =>
                        activeWorkout.items[focusItemIndex].exercise &&
                        setSwapModalItem({
                          itemId: activeWorkout.items[focusItemIndex].id,
                          currentExercise: activeWorkout.items[focusItemIndex].exercise!,
                        })
                      }
                      className="text-xs font-semibold text-ink-200 flex items-center gap-1 hover:text-white"
                    >
                      <Shuffle className="w-3.5 h-3.5" /> Trocar
                    </button>

                    {activeWorkout.items[focusItemIndex].exercise?.video_url && (
                      <button
                        onClick={() => setViewExercise(activeWorkout.items[focusItemIndex].exercise!)}
                        className="text-xs font-semibold text-lime flex items-center gap-1 hover:underline"
                      >
                        <Video className="w-4 h-4" /> Execução
                      </button>
                    )}
                  </div>
                </div>

                <div className="card p-6 space-y-4 bg-ink-800">
                  <div className="text-2xl font-bold text-white leading-tight">
                    {activeWorkout.items[focusItemIndex].exercise?.name}
                  </div>
                  <div className="text-xs text-ink-300">
                    {activeWorkout.items[focusItemIndex].exercise?.muscle_group} · {activeWorkout.items[focusItemIndex].exercise?.equipment}
                  </div>

                  {activeWorkout.items[focusItemIndex].notes && (
                    <div className="text-xs text-lime bg-lime/10 p-3 rounded-2xl">
                      💡 {activeWorkout.items[focusItemIndex].notes}
                    </div>
                  )}

                  <div className="space-y-2 pt-2">
                    <span className="text-xs font-semibold text-ink-300">Registro de Séries:</span>
                    {(setLogs[activeWorkout.items[focusItemIndex].id] || []).map((s, sIdx) => (
                      <div
                        key={sIdx}
                        className={`p-3 rounded-2xl flex items-center justify-between gap-3 transition-colors ${
                          s.done ? "bg-lime text-black font-semibold" : "bg-ink-750 text-white"
                        }`}
                      >
                        <span className="text-xs font-bold w-16">Série {sIdx + 1}</span>

                        <div className="flex items-center gap-2 text-xs">
                          <label className="flex items-center gap-1">
                            <span className={s.done ? "text-black/70" : "text-ink-300"}>Kg:</span>
                            <input
                              type="number"
                              value={s.load}
                              onChange={(e) => updateSetLoad(activeWorkout.items[focusItemIndex].id, sIdx, parseFloat(e.target.value) || 0)}
                              className={`w-14 px-2 py-1 rounded-lg text-center font-bold text-sm ${s.done ? "bg-black/10 text-black" : "bg-ink-700 text-white"}`}
                            />
                          </label>

                          <span className={s.done ? "text-black/70" : "text-ink-400"}>×</span>

                          <span className="font-semibold">{s.reps} reps</span>
                        </div>

                        <button
                          onClick={() => toggleSetComplete(activeWorkout.items[focusItemIndex].id, sIdx, activeWorkout.items[focusItemIndex].rest_seconds)}
                          className={`btn-icon w-9 h-9 ${s.done ? "bg-black text-lime" : "bg-ink-700 hover:bg-lime hover:text-black text-ink-300"}`}
                        >
                          <Check className="w-4 h-4 stroke-[3]" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => setFocusItemIndex(Math.max(0, focusItemIndex - 1))}
                    disabled={focusItemIndex === 0}
                    className="btn-dark flex-1 !py-3.5"
                  >
                    <ChevronLeft className="w-4 h-4" /> Anterior
                  </button>

                  {focusItemIndex < activeWorkout.items.length - 1 ? (
                    <button
                      onClick={() => setFocusItemIndex(focusItemIndex + 1)}
                      className="btn-lime flex-1 !py-3.5"
                    >
                      Próximo <ChevronRight className="w-4 h-4" />
                    </button>
                  ) : (
                    <button
                      onClick={finishSession}
                      className="btn-lime flex-1 !py-3.5 bg-lime font-bold shadow-glow"
                    >
                      <CheckCheck className="w-5 h-5" /> Finalizar Treino
                    </button>
                  )}
                </div>

                <button
                  onClick={() => {
                    setInFocusMode(false);
                    setSessionActive(false);
                  }}
                  className="w-full text-center text-xs text-ink-400 hover:text-white py-2"
                >
                  Pausar e sair do modo tela cheia
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* MODAL: RESUMO DE TREINO FINALIZADO */}
      {showFinishSummary && (
        <div className="fixed inset-0 z-[80] bg-black/85 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-ink-850 border border-white/10 rounded-4xl max-w-md w-full p-6 space-y-6 shadow-card animate-fade-up text-center">
            <div className="w-20 h-20 rounded-full bg-lime text-black flex items-center justify-center mx-auto shadow-glow">
              <Trophy className="w-10 h-10" />
            </div>

            <div>
              <h3 className="text-2xl font-black text-white">Treino Concluído! 🎉</h3>
              <p className="text-xs text-ink-300 mt-1">Parabéns pelo esforço e consistência de hoje.</p>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <div className="p-3 bg-ink-750 rounded-2xl">
                <div className="text-[10px] text-ink-300 font-semibold">TEMPO</div>
                <div className="text-lg font-bold text-white mt-1 font-mono">{fmtTime(sessionSeconds)}</div>
              </div>
              <div className="p-3 bg-ink-750 rounded-2xl">
                <div className="text-[10px] text-ink-300 font-semibold">VOLUME TOTAL</div>
                <div className="text-lg font-bold text-lime mt-1">{totalVolumeKg || 1450} kg</div>
              </div>
              <div className="p-3 bg-ink-750 rounded-2xl">
                <div className="text-[10px] text-ink-300 font-semibold">SÉRIES</div>
                <div className="text-lg font-bold text-white mt-1">{totalSets}</div>
              </div>
            </div>

            <div className="space-y-2 text-left">
              <label className="text-xs font-semibold text-white">Como você avalia a intensidade (RPE 1-10)?</label>
              <div className="flex items-center justify-between gap-1">
                {[5, 6, 7, 8, 9, 10].map((score) => (
                  <button
                    key={score}
                    onClick={() => setRpeScore(score)}
                    className={`flex-1 py-2 rounded-xl text-xs font-bold transition-all ${
                      rpeScore === score ? "bg-lime text-black shadow-glow" : "bg-ink-750 text-ink-300 hover:text-white"
                    }`}
                  >
                    {score}
                  </button>
                ))}
              </div>
            </div>

            <button
              onClick={() => {
                setShowFinishSummary(false);
                setInFocusMode(false);
                toast.success("Treino salvo com sucesso no seu histórico!");
              }}
              className="btn-lime w-full !py-4 font-bold text-base shadow-glow"
            >
              Concluir e Voltar
            </button>
          </div>
        </div>
      )}

      {/* MODAL: GUIA TÉCNICO & VÍDEO DO EXERCÍCIO */}
      {viewExercise && (
        <div className="fixed inset-0 z-[80] bg-black/85 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-ink-850 border border-white/10 rounded-4xl max-w-md w-full p-6 space-y-4 shadow-card animate-fade-up">
            <div className="flex items-center justify-between pb-3 border-b border-white/10">
              <h3 className="text-lg font-bold text-white">{viewExercise.name}</h3>
              <button onClick={() => setViewExercise(null)} className="btn-icon w-8 h-8 bg-ink-700 text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3">
              <div className="flex items-center gap-2 text-xs text-ink-300">
                <span className="badge-ok">{viewExercise.muscle_group}</span>
                <span className="badge-mute">{viewExercise.equipment}</span>
              </div>

              {viewExercise.video_url ? (
                <div className="rounded-2xl overflow-hidden aspect-video bg-black flex items-center justify-center border border-white/10">
                  <iframe
                    src={viewExercise.video_url.replace("watch?v=", "embed/")}
                    title={viewExercise.name}
                    className="w-full h-full"
                    allowFullScreen
                  />
                </div>
              ) : (
                <div className="p-8 rounded-2xl bg-ink-750 text-center text-xs text-ink-300">
                  <Video className="w-8 h-8 text-lime mx-auto mb-2" />
                  Demonstração em animação disponível em breve.
                </div>
              )}

              <div className="text-xs text-ink-200 leading-relaxed bg-ink-750 p-4 rounded-2xl">
                {viewExercise.description || "Mantenha postura alinhada, execute o movimento em amplitude completa com respiração controlada."}
              </div>
            </div>

            <button onClick={() => setViewExercise(null)} className="btn-dark w-full !py-3 text-xs">
              Fechar
            </button>
          </div>
        </div>
      )}

      {/* Cronômetro de descanso flutuante */}
      {timerVisible && (
        <div className="fixed bottom-24 inset-x-0 z-50 px-4 animate-fade-up">
          <div className="max-w-xl mx-auto rounded-4xl bg-lime text-black p-3 pr-4 flex items-center gap-3 shadow-lime">
            <Ring
              value={timerTotal ? (timerSeconds / timerTotal) * 100 : 0}
              size={56}
              stroke={5}
              color="#000"
              track="rgba(0,0,0,0.12)"
            >
              <Clock className="w-5 h-5" />
            </Ring>
            <div className="flex-1">
              <div className="text-[11px] font-semibold text-black/60">
                {timerSeconds === 0 ? "Descanso finalizado!" : "Descanso entre séries"}
              </div>
              <div className="text-3xl font-bold font-mono tracking-tight leading-none mt-0.5">{fmtTime(timerSeconds)}</div>
            </div>
            <button onClick={() => setTimerActive(!timerActive)} className="btn-icon bg-black text-lime">
              {timerActive ? <Pause className="w-5 h-5 fill-current" /> : <Play className="w-5 h-5 fill-current" />}
            </button>
            <button onClick={() => setTimerSeconds(timerTotal)} className="btn-icon bg-black/10 hover:bg-black/20">
              <RotateCcw className="w-5 h-5" />
            </button>
            <button
              onClick={() => { setTimerActive(false); setTimerVisible(false); }}
              className="text-xs font-semibold text-black/60 hover:text-black px-1"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {/* Modal de Registro de Nova Medição pelo Aluno */}
      {showMyMeasurementModal && (
        <Modal
          title="Registrar Medidas & Peso"
          onClose={() => setShowMyMeasurementModal(false)}
          width="max-w-md"
          footer={
            <>
              <button className="btn-ghost" onClick={() => setShowMyMeasurementModal(false)}>
                Cancelar
              </button>
              <button
                className="btn-lime"
                disabled={savingMyMeasure}
                onClick={handleSaveMyMeasurement}
              >
                {savingMyMeasure ? "Salvando..." : "Salvar Medição"}
              </button>
            </>
          }
        >
          <div className="space-y-4 text-xs">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Peso Atual (kg)</label>
                <input
                  type="number"
                  step="0.1"
                  className="field"
                  value={myMeasureForm.weight}
                  onChange={(e) => setMyMeasureForm({ ...myMeasureForm, weight: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="label">Altura (cm)</label>
                <input
                  type="number"
                  className="field"
                  value={myMeasureForm.height}
                  onChange={(e) => setMyMeasureForm({ ...myMeasureForm, height: parseFloat(e.target.value) || 0 })}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label">Cintura (cm)</label>
                <input
                  type="number"
                  step="0.5"
                  className="field"
                  placeholder="Ex: 75"
                  value={myMeasureForm.waist || ""}
                  onChange={(e) => setMyMeasureForm({ ...myMeasureForm, waist: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="label">Quadril (cm)</label>
                <input
                  type="number"
                  step="0.5"
                  className="field"
                  placeholder="Ex: 98"
                  value={myMeasureForm.hips || ""}
                  onChange={(e) => setMyMeasureForm({ ...myMeasureForm, hips: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="label">Tórax (cm)</label>
                <input
                  type="number"
                  step="0.5"
                  className="field"
                  placeholder="Ex: 92"
                  value={myMeasureForm.chest || ""}
                  onChange={(e) => setMyMeasureForm({ ...myMeasureForm, chest: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div>
                <label className="label">Braço (cm)</label>
                <input
                  type="number"
                  step="0.5"
                  className="field"
                  placeholder="Ex: 32"
                  value={myMeasureForm.right_arm || ""}
                  onChange={(e) => setMyMeasureForm({ ...myMeasureForm, right_arm: parseFloat(e.target.value) || 0 })}
                />
              </div>
            </div>

            <div>
              <label className="label">Anotações / Sensações</label>
              <textarea
                rows={2}
                className="field"
                placeholder="Ex: Me sentindo mais leve e com menos retenção..."
                value={myMeasureForm.notes}
                onChange={(e) => setMyMeasureForm({ ...myMeasureForm, notes: e.target.value })}
              />
            </div>
          </div>
        </Modal>
      )}

      {boletoHtml && (
        <Modal title="Boleto" onClose={() => setBoletoHtml(null)} width="max-w-2xl">
          <iframe title="Boleto" srcDoc={boletoHtml} sandbox="allow-modals" className="w-full h-[60vh] rounded-2xl bg-white" />
        </Modal>
      )}
      {showDelete && (
        <Modal title="Excluir conta" onClose={() => setShowDelete(false)} width="max-w-md"
          footer={<><button className="btn-ghost" onClick={() => setShowDelete(false)}>Voltar</button>
            <button className="btn bg-[#FF5A36] text-white px-5 py-3" disabled={!deletePw}
              onClick={async () => { try { await api.deleteMyAccount(deletePw); toast.success("Conta excluída"); clearToken(); router.push("/"); } catch (e: any) { toast.error(e.message); } }}>Excluir para sempre</button></>}>
          <p className="text-sm text-ink-200">Digite sua senha para confirmar.</p>
          <input type="password" className="field" value={deletePw} onChange={(e) => setDeletePw(e.target.value)} autoComplete="current-password" />
        </Modal>
      )}

      {/* Bottom Nav Flutuante Completo */}
      <nav className="fixed bottom-4 inset-x-0 z-40 px-3">
        <div className="max-w-md mx-auto flex items-center justify-around p-1.5 rounded-full bg-ink-850/90 backdrop-blur-xl border border-white/[0.08] shadow-card">
          {[
            { id: "workout" as const, label: "Treino", icon: <Dumbbell className="w-4 h-4" /> },
            { id: "agenda" as const, label: "Aulas", icon: <Calendar className="w-4 h-4" /> },
            { id: "progress" as const, label: "Progresso", icon: <TrendingUp className="w-4 h-4" /> },
            { id: "finance" as const, label: "Faturas", icon: <CreditCard className="w-4 h-4" /> },
            { id: "checkin" as const, label: "Entrada", icon: <QrCode className="w-4 h-4" /> },
            { id: "profile" as const, label: "Perfil", icon: <User className="w-4 h-4" /> },
          ].map((t) => {
            const on = activeTab === t.id;
            return (
              <button
                key={t.id}
                onClick={() => setActiveTab(t.id)}
                className={`flex flex-col items-center gap-1 py-1.5 px-2.5 rounded-full transition-all ${
                  on ? "bg-lime text-black font-bold shadow-glow" : "text-ink-300 hover:text-white"
                }`}
              >
                {t.icon}
                <span className="text-[10px]">{t.label}</span>
              </button>
            );
          })}
        </div>
      </nav>
    </div>
  );
}
