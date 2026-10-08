"use client";

import React, { useEffect, useState, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import {
  Dumbbell, Plus, Trash2, Search, CheckCircle, Sparkles, Save, User,
  ArrowUp, ArrowDown, Printer, Video, X, Zap, Activity, Filter, Eye,
  Copy, MessageSquare, Clock, Users, ChevronDown, ChevronUp, BarChart3,
  Share2, Flame, Award, BookOpen, Bookmark, FileText, Check
} from "lucide-react";
import { api } from "@/services/api";
import { Exercise, Student, WorkoutPlan, WorkoutTemplate, PhysicalAssessment } from "@/types";
import { Avatar, Empty } from "@/components/ui";
import { IMG } from "@/lib/images";
import { useToast } from "@/context/ToastContext";

function ProfessorContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialStudentId = searchParams.get("studentId");
  const toast = useToast();

  const [students, setStudents] = useState<Student[]>([]);
  const [selectedStudent, setSelectedStudent] = useState<Student | null>(null);
  const [exercises, setExercises] = useState<Exercise[]>([]);
  const [muscleGroups, setMuscleGroups] = useState<{ group: string; count: number }[]>([]);
  const [selectedGroup, setSelectedGroup] = useState("");
  const [searchExercise, setSearchExercise] = useState("");
  const [activePlans, setActivePlans] = useState<WorkoutPlan[]>([]);
  const [todayCheckins, setTodayCheckins] = useState<any[]>([]);

  // Filtros de Aluno
  const [studentFilter, setStudentFilter] = useState("");
  const [studentFilterMode, setStudentFilterMode] = useState<"ALL" | "TODAY" | "NO_PLAN">("ALL");

  // Estado do formulário de criação de treino
  const [planTitle, setPlanTitle] = useState("Treino Hipertrofia 2026");
  const [planGoal, setPlanGoal] = useState("Ganho de massa muscular magra");
  const [workouts, setWorkouts] = useState<any[]>([
    {
      letter: "A",
      name: "Peitoral, Tríceps e Ombros",
      description: "Push Day",
      items: [],
    },
    {
      letter: "B",
      name: "Costas e Bíceps",
      description: "Pull Day",
      items: [],
    },
  ]);
  const [activeWorkoutTab, setActiveWorkoutTab] = useState(0);
  const [saving, setSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [showVolumeWidget, setShowVolumeWidget] = useState(true);
  const [showFloorModal, setShowFloorModal] = useState(false);

  // Modal Novo Exercício
  const [showNewExerciseModal, setShowNewExerciseModal] = useState(false);
  const [newExercise, setNewExercise] = useState({
    name: "",
    muscle_group: "Peitoral",
    equipment: "Halteres",
    description: "",
    video_url: "",
  });
  const [creatingExercise, setCreatingExercise] = useState(false);

  // Biblioteca de Modelos de Treino
  const [showTemplatesModal, setShowTemplatesModal] = useState(false);
  const [templateList, setTemplateList] = useState<WorkoutTemplate[]>([]);
  const [loadingTemplates, setLoadingTemplates] = useState(false);
  const [templateGoalFilter, setTemplateGoalFilter] = useState("TODOS");
  const [templateLevelFilter, setTemplateLevelFilter] = useState("TODOS");
  const [templateSearch, setTemplateSearch] = useState("");
  const [previewTemplate, setPreviewTemplate] = useState<WorkoutTemplate | null>(null);

  // Modal Salvar Treino Atual como Novo Modelo
  const [showSaveTemplateModal, setShowSaveTemplateModal] = useState(false);
  const [saveTmplTitle, setSaveTmplTitle] = useState("");
  const [saveTmplGoal, setSaveTmplGoal] = useState("Hipertrofia");
  const [saveTmplLevel, setSaveTmplLevel] = useState("Intermediário");
  const [saveTmplDesc, setSaveTmplDesc] = useState("");
  const [savingTemplate, setSavingTemplate] = useState(false);

  // Modal de Avaliação Física do Aluno
  const [showAssessmentModal, setShowAssessmentModal] = useState(false);
  const [assessments, setAssessments] = useState<PhysicalAssessment[]>([]);
  const [loadingAssessments, setLoadingAssessments] = useState(false);
  const [showNewAssessmentForm, setShowNewAssessmentForm] = useState(false);
  const [newAssessment, setNewAssessment] = useState({
    weight: 70,
    height: 170,
    body_fat: 18,
    muscle_mass: 32,
    chest: 95,
    waist: 78,
    abdomen: 82,
    hips: 100,
    right_arm: 32,
    left_arm: 32,
    right_thigh: 56,
    left_thigh: 56,
    right_calf: 36,
    left_calf: 36,
    notes: "",
  });
  const [savingAssessment, setSavingAssessment] = useState(false);

  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      const [stdRes, exRes, chkRes] = await Promise.all([
        api.listStudents("", "", 1, 100),
        api.listExercises(""),
        api.getTodayCheckIns().catch(() => []), // sem a permissão de recepção a lista fica vazia
      ]);

      setStudents(stdRes.data);
      setExercises(exRes.data);
      setMuscleGroups(exRes.groups);
      setTodayCheckins(chkRes);

      if (initialStudentId) {
        const found = stdRes.data.find((s) => s.id === initialStudentId);
        if (found) selectStudent(found);
      } else if (stdRes.data.length > 0) {
        selectStudent(stdRes.data[0]);
      }
    } catch (err: any) {
      console.error(err);
      toast.error(err.message || "Não foi possível carregar os alunos");
    }
  };

  const selectStudent = async (student: Student) => {
    setSelectedStudent(student);
    setSaveSuccess(false);
    try {
      const plans = await api.getStudentWorkouts(student.id);
      setActivePlans(plans);
    } catch (err) {
      console.error(err);
    }
  };

  // Carrega template pronto
  const applyTemplate = (templateType: "ABC" | "AB") => {
    if (templateType === "ABC") {
      setPlanTitle("Treino ABC - Clássico Hipertrofia");
      setPlanGoal("Foco em volume muscular e intensidade");

      const supino = exercises.find((e) => e.name.includes("Supino Reto")) || exercises[0];
      const crossover = exercises.find((e) => e.name.includes("Crossover")) || exercises[1];
      const triceps = exercises.find((e) => e.name.includes("Tríceps Pulley")) || exercises[2];
      const puxada = exercises.find((e) => e.name.includes("Puxada Frontal")) || exercises[3];
      const remada = exercises.find((e) => e.name.includes("Remada")) || exercises[4];
      const rosca = exercises.find((e) => e.name.includes("Rosca Direta")) || exercises[5];
      const agachamento = exercises.find((e) => e.name.includes("Agachamento Livre")) || exercises[6];
      const leg = exercises.find((e) => e.name.includes("Leg Press")) || exercises[7];

      setWorkouts([
        {
          letter: "A",
          name: "Peito, Tríceps e Deltoides",
          description: "Push Day",
          items: [
            { exercise: supino, sets: 4, reps: "8-10", load_kg: 60, rest_seconds: 90, technique: "Padrão", notes: "Cadência controlada" },
            { exercise: crossover, sets: 3, reps: "12-15", load_kg: 25, rest_seconds: 60, technique: "Pico de Contração", notes: "Pico de contração no meio" },
            { exercise: triceps, sets: 4, reps: "12", load_kg: 35, rest_seconds: 45, technique: "Drop-Set", notes: "Última série em drop" },
          ],
        },
        {
          letter: "B",
          name: "Costas, Bíceps e Trapézio",
          description: "Pull Day",
          items: [
            { exercise: puxada, sets: 4, reps: "10-12", load_kg: 50, rest_seconds: 75, technique: "Padrão", notes: "" },
            { exercise: remada, sets: 4, reps: "8-10", load_kg: 55, rest_seconds: 75, technique: "Rest-Pause", notes: "" },
            { exercise: rosca, sets: 3, reps: "10-12", load_kg: 28, rest_seconds: 60, technique: "Padrão", notes: "" },
          ],
        },
        {
          letter: "C",
          name: "Pernas Completo e Abdômen",
          description: "Legs & Core Day",
          items: [
            { exercise: agachamento, sets: 4, reps: "8-10", load_kg: 80, rest_seconds: 120, technique: "Padrão", notes: "Amplitude completa" },
            { exercise: leg, sets: 4, reps: "12", load_kg: 180, rest_seconds: 90, technique: "Pirâmide", notes: "" },
          ],
        },
      ]);
      toast.success("Template ABC carregado na ficha!");
    } else {
      setPlanTitle("Treino AB - Adaptação e Condicionamento");
      setPlanGoal("Resistência muscular e condicionamento geral");
      const supino = exercises.find((e) => e.name.includes("Supino")) || exercises[0];
      const puxada = exercises.find((e) => e.name.includes("Puxada")) || exercises[1];
      const leg = exercises.find((e) => e.name.includes("Leg Press")) || exercises[2];

      setWorkouts([
        {
          letter: "A",
          name: "Membros Superiores",
          description: "Tronco e Braços",
          items: [
            { exercise: supino, sets: 3, reps: "12-15", load_kg: 30, rest_seconds: 60, technique: "Padrão", notes: "" },
            { exercise: puxada, sets: 3, reps: "12-15", load_kg: 35, rest_seconds: 60, technique: "Padrão", notes: "" },
          ],
        },
        {
          letter: "B",
          name: "Membros Inferiores & Core",
          description: "Pernas e Abdômen",
          items: [
            { exercise: leg, sets: 3, reps: "12-15", load_kg: 80, rest_seconds: 60, technique: "Padrão", notes: "" },
          ],
        },
      ]);
      toast.success("Template AB carregado na ficha!");
    }
  };

  const openTemplatesModal = async () => {
    setShowTemplatesModal(true);
    setLoadingTemplates(true);
    try {
      const res = await api.listWorkoutTemplates();
      setTemplateList(res);
      if (res.length > 0) setPreviewTemplate(res[0]);
    } catch (err: any) {
      toast.error(err.message || "Erro ao carregar modelos");
    } finally {
      setLoadingTemplates(false);
    }
  };

  const applyTemplateFromLibrary = (tmpl: WorkoutTemplate) => {
    setPlanTitle(tmpl.title);
    setPlanGoal(tmpl.goal);
    const newWorkouts = tmpl.workouts.map((w) => ({
      letter: w.letter,
      name: w.name,
      description: w.description || "",
      items: w.items.map((it) => {
        const foundEx = exercises.find((e) => e.id === it.exercise_id) || it.exercise || exercises[0];
        return {
          exercise: foundEx,
          sets: it.sets || 4,
          reps: it.reps || "10-12",
          load_kg: it.load_kg || 0,
          rest_seconds: it.rest_seconds || 60,
          technique: "Padrão",
          notes: it.notes || "",
        };
      }),
    }));
    setWorkouts(newWorkouts);
    setActiveWorkoutTab(0);
    setShowTemplatesModal(false);
    toast.success(`Modelo "${tmpl.title}" aplicado na ficha com sucesso!`);
  };

  const handleDeleteTemplate = async (tmplId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm("Deseja realmente excluir este modelo da academia?")) return;
    try {
      await api.deleteWorkoutTemplate(tmplId);
      toast.success("Modelo removido da biblioteca");
      setTemplateList(prev => prev.filter(t => t.id !== tmplId));
      if (previewTemplate?.id === tmplId) {
        setPreviewTemplate(templateList.find(t => t.id !== tmplId) || null);
      }
    } catch (err: any) {
      toast.error(err.message || "Erro ao excluir modelo");
    }
  };

  const handleSaveAsTemplate = async () => {
    if (!saveTmplTitle.trim()) {
      toast.error("Informe o título do modelo");
      return;
    }
    setSavingTemplate(true);
    try {
      const payload = {
        title: saveTmplTitle,
        goal: saveTmplGoal,
        level: saveTmplLevel,
        description: saveTmplDesc,
        workouts: workouts.map((w, wIdx) => ({
          letter: w.letter,
          name: w.name,
          description: w.description,
          order_index: wIdx + 1,
          items: w.items.map((it: any, itIdx: number) => ({
            exercise_id: it.exercise.id,
            sets: it.sets,
            reps: it.reps,
            load_kg: it.load_kg,
            rest_seconds: it.rest_seconds,
            notes: it.notes,
            order_index: itIdx + 1,
          })),
        })),
      };
      await api.createWorkoutTemplate(payload);
      toast.success("Modelo salvo na biblioteca da academia!");
      setShowSaveTemplateModal(false);
      setSaveTmplTitle("");
      setSaveTmplDesc("");
    } catch (err: any) {
      toast.error(err.message || "Erro ao salvar modelo");
    } finally {
      setSavingTemplate(false);
    }
  };

  const openAssessmentModal = async () => {
    if (!selectedStudent) return;
    setShowAssessmentModal(true);
    setLoadingAssessments(true);
    setShowNewAssessmentForm(false);
    if (selectedStudent.weight) {
      setNewAssessment(prev => ({
        ...prev,
        weight: selectedStudent.weight || 70,
        height: selectedStudent.height || 170,
      }));
    }
    try {
      const res = await api.listStudentAssessments(selectedStudent.id);
      setAssessments(res);
    } catch (err: any) {
      toast.error(err.message || "Erro ao buscar avaliações físicas");
    } finally {
      setLoadingAssessments(false);
    }
  };

  const handleCreateAssessment = async () => {
    if (!selectedStudent) return;
    setSavingAssessment(true);
    try {
      const res = await api.createStudentAssessment(selectedStudent.id, newAssessment);
      toast.success("Avaliação física registrada com sucesso!");
      setAssessments([res, ...assessments]);
      setShowNewAssessmentForm(false);
      setSelectedStudent({ ...selectedStudent, weight: newAssessment.weight, height: newAssessment.height });
    } catch (err: any) {
      toast.error(err.message || "Erro ao salvar avaliação");
    } finally {
      setSavingAssessment(false);
    }
  };

  const addExerciseToCurrentWorkout = (ex: Exercise) => {
    const updated = [...workouts];
    updated[activeWorkoutTab].items.push({
      exercise: ex,
      sets: 4,
      reps: "10-12",
      load_kg: 20,
      rest_seconds: 60,
      technique: "Padrão",
      notes: "",
    });
    setWorkouts(updated);
    toast.success(`"${ex.name}" adicionado ao Treino ${updated[activeWorkoutTab].letter}`);
  };

  const removeExerciseFromWorkout = (itemIndex: number) => {
    const updated = [...workouts];
    updated[activeWorkoutTab].items.splice(itemIndex, 1);
    setWorkouts(updated);
  };

  const moveExercise = (itemIndex: number, direction: "up" | "down") => {
    const currentItems = [...workouts[activeWorkoutTab].items];
    const targetIndex = direction === "up" ? itemIndex - 1 : itemIndex + 1;
    if (targetIndex < 0 || targetIndex >= currentItems.length) return;

    const temp = currentItems[itemIndex];
    currentItems[itemIndex] = currentItems[targetIndex];
    currentItems[targetIndex] = temp;

    const updated = [...workouts];
    updated[activeWorkoutTab].items = currentItems;
    setWorkouts(updated);
  };

  const handleSaveWorkoutPlan = async () => {
    if (!selectedStudent) return;
    setSaving(true);
    setSaveSuccess(false);

    try {
      const payload = {
        title: planTitle,
        goal: planGoal,
        workouts: workouts.map((w, idx) => ({
          letter: w.letter,
          name: w.name,
          description: w.description,
          order_index: idx + 1,
          items: w.items.map((it: any) => ({
            exercise_id: it.exercise.id,
            sets: it.sets,
            reps: it.reps,
            load_kg: it.load_kg,
            rest_seconds: it.rest_seconds,
            notes: it.technique && it.technique !== "Padrão" ? `[${it.technique}] ${it.notes || ""}`.trim() : it.notes,
          })),
        })),
      };

      await api.createWorkoutPlan(selectedStudent.id, payload);
      setSaveSuccess(true);
      toast.success(`Ficha salva e ativada no app de ${selectedStudent.user?.name}!`);
      const plans = await api.getStudentWorkouts(selectedStudent.id);
      setActivePlans(plans);
    } catch (err: any) {
      toast.error("Erro ao salvar treino: " + err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleCreateExercise = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreatingExercise(true);
    try {
      const created = await api.createExercise(newExercise);
      toast.success(`Exercício "${created.name}" cadastrado com sucesso!`);
      setShowNewExerciseModal(false);
      setNewExercise({
        name: "",
        muscle_group: "Peitoral",
        equipment: "Halteres",
        description: "",
        video_url: "",
      });
      // Recarrega catálogo
      const exRes = await api.listExercises("");
      setExercises(exRes.data);
      setMuscleGroups(exRes.groups);
    } catch (err: any) {
      toast.error("Erro ao criar exercício: " + err.message);
    } finally {
      setCreatingExercise(false);
    }
  };

  const handlePrint = () => {
    window.print();
  };

  const filteredExercises = exercises.filter((ex) => {
    const matchesGroup = selectedGroup ? ex.muscle_group === selectedGroup : true;
    const matchesSearch = searchExercise
      ? ex.name.toLowerCase().includes(searchExercise.toLowerCase())
      : true;
    return matchesGroup && matchesSearch;
  });

  const todayStudentUserIds = todayCheckins.map((c) => c.student?.user_id);

  const visibleStudents = students.filter((s) => {
    const matchesSearch = studentFilter
      ? s.user?.name?.toLowerCase().includes(studentFilter.toLowerCase())
      : true;

    if (!matchesSearch) return false;

    if (studentFilterMode === "TODAY") {
      return todayStudentUserIds.includes(s.user_id);
    }
    return true;
  });

  const currentWorkout = workouts[activeWorkoutTab];

  const updateItem = (itemIdx: number, patch: Record<string, any>) => {
    const updated = [...workouts];
    Object.assign(updated[activeWorkoutTab].items[itemIdx], patch);
    setWorkouts(updated);
  };

  // Volume semanal por grupo muscular
  const volumeByGroup = React.useMemo(() => {
    const map: Record<string, { sets: number; exercises: number }> = {};
    workouts.forEach((w) => {
      w.items?.forEach((it: any) => {
        const group = it.exercise?.muscle_group || "Geral";
        if (!map[group]) map[group] = { sets: 0, exercises: 0 };
        map[group].sets += Number(it.sets) || 0;
        map[group].exercises += 1;
      });
    });

    return Object.entries(map).map(([group, data]) => {
      let badgeClass = "badge-ok text-lime border-lime/30";
      let statusLabel = "Hipertrofia Ótima (10-20)";
      if (data.sets < 10) {
        badgeClass = "bg-amber-500/10 text-amber-300 border-amber-500/20";
        statusLabel = "Manutenção / Baixo (<10)";
      } else if (data.sets > 22) {
        badgeClass = "bg-rose-500/10 text-rose-300 border-rose-500/20";
        statusLabel = "Fadiga Alta (>22)";
      }
      return {
        group,
        sets: data.sets,
        exercises: data.exercises,
        badgeClass,
        statusLabel,
        percent: Math.min(100, Math.round((data.sets / 24) * 100)),
      };
    }).sort((a, b) => b.sets - a.sets);
  }, [workouts]);

  // Duplicar divisão de treino
  const duplicateCurrentWorkout = () => {
    const letters = ["A", "B", "C", "D", "E", "F", "G", "H"];
    const nextLetter = letters[workouts.length] || `X${workouts.length + 1}`;
    const copy = JSON.parse(JSON.stringify(workouts[activeWorkoutTab]));
    copy.letter = nextLetter;
    copy.name = `${copy.name} (Cópia)`;
    setWorkouts([...workouts, copy]);
    setActiveWorkoutTab(workouts.length);
    toast.success(`Divisão ${nextLetter} duplicada a partir do Treino ${workouts[activeWorkoutTab].letter}!`);
  };

  // Enviar treino completo no WhatsApp
  const shareWorkoutViaWhatsApp = () => {
    if (!selectedStudent) return;
    const phone = selectedStudent.user?.phone?.replace(/\D/g, "");
    if (!phone) {
      toast.error("Aluno não possui telefone cadastrado!");
      return;
    }

    let text = `🏋️ *GymSync - Nova Ficha de Treino*\n`;
    text += `👤 *Aluno:* ${selectedStudent.user?.name}\n`;
    text += `🎯 *Objetivo:* ${planGoal}\n`;
    text += `📋 *Ficha:* ${planTitle}\n\n`;

    workouts.forEach((w) => {
      text += `*TREINO ${w.letter} - ${w.name}*\n`;
      if (w.description) text += `_${w.description}_\n`;
      w.items.forEach((it: any, idx: number) => {
        const tech = it.technique && it.technique !== "Padrão" ? ` [${it.technique}]` : "";
        text += `${idx + 1}. *${it.exercise.name}* - ${it.sets}x ${it.reps} (${it.load_kg}kg)${tech}\n`;
      });
      text += `\n`;
    });

    text += `Bons treinos! Qualquer dúvida, chame o professor na academia. 💪`;

    window.open(`https://wa.me/55${phone}?text=${encodeURIComponent(text)}`, "_blank");
    toast.success("Abrindo WhatsApp para enviar ficha ao aluno!");
  };

  // Tempo decorrido desde o check-in
  const getElapsedTime = (dateString: string) => {
    if (!dateString) return "Hoje";
    const diffMs = Date.now() - new Date(dateString).getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    if (diffMins < 60) return `${Math.max(1, diffMins)} min atrás`;
    const hours = Math.floor(diffMins / 60);
    const mins = diffMins % 60;
    return `${hours}h ${mins}m atrás`;
  };

  return (
    <div className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8 print:p-0">
      {/* Cabeçalho */}
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 animate-fade-up print:hidden">
        <div>
          <div className="eyebrow mb-2">Treinos</div>
          <h1 className="text-3xl sm:text-4xl font-semibold text-white tracking-tight">
            Prescrição de <span className="text-lime">treinos</span>
          </h1>
          <p className="text-sm text-ink-300 mt-2">
            Templates rápidos, técnicas avançadas, reordenação de exercícios e acompanhamento do salão.
          </p>
        </div>

        {/* Live Salão e Ações Rápidas */}
        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowNewExerciseModal(true)}
            className="btn-dark !py-2.5 text-xs font-semibold"
          >
            <Plus className="w-4 h-4 text-lime" /> Novo Exercício
          </button>

          <button
            onClick={handlePrint}
            className="btn-dark !py-2.5 text-xs font-semibold"
            title="Imprimir ficha de treino"
          >
            <Printer className="w-4 h-4 text-ink-200" /> Imprimir
          </button>

          <button
            onClick={() => setShowFloorModal(true)}
            className="flex items-center gap-3 pl-2 pr-4 py-2 rounded-full bg-ink-800 hover:bg-ink-750 border border-white/[0.04] transition-all cursor-pointer group"
            title="Clique para ver quem está treinando agora"
          >
            <span className="relative w-10 h-10 rounded-full bg-lime/15 flex items-center justify-center group-hover:scale-105 transition-transform">
              <span className="w-2.5 h-2.5 rounded-full bg-lime animate-ping absolute" />
              <span className="w-2.5 h-2.5 rounded-full bg-lime" />
            </span>
            <div className="leading-tight text-left">
              <div className="text-[11px] text-ink-300 flex items-center gap-1">
                <span>Entradas hoje</span>
              </div>
              <div className="text-sm font-semibold text-white group-hover:text-lime transition-colors">
                {todayCheckins.length} alunos
              </div>
            </div>
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-4">
        {/* Coluna 1: Seleção de Aluno */}
        <aside className="card p-4 space-y-3 h-fit lg:sticky lg:top-28 print:hidden">
          <div className="flex items-center justify-between px-1">
            <h3 className="section-title flex items-center gap-2">
              <User className="w-4 h-4 text-lime" /> Alunos
            </h3>
            <span className="badge-mute">{visibleStudents.length}</span>
          </div>

          <div className="relative">
            <Search className="w-4 h-4 text-ink-400 absolute left-4 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Filtrar aluno..."
              value={studentFilter}
              onChange={(e) => setStudentFilter(e.target.value)}
              className="field !rounded-full !py-2.5 pl-11"
            />
          </div>

          {/* Filtros rápidos: Todos vs No Salão Hoje */}
          <div className="flex items-center gap-1 p-1 bg-ink-750 rounded-full text-[11px]">
            <button
              onClick={() => setStudentFilterMode("ALL")}
              className={`flex-1 py-1.5 rounded-full font-semibold transition-all ${studentFilterMode === "ALL" ? "bg-lime text-black" : "text-ink-300 hover:text-white"}`}
            >
              Todos ({students.length})
            </button>
            <button
              onClick={() => setStudentFilterMode("TODAY")}
              className={`flex-1 py-1.5 rounded-full font-semibold transition-all ${studentFilterMode === "TODAY" ? "bg-lime text-black" : "text-ink-300 hover:text-white"}`}
            >
              No Salão ({todayCheckins.length})
            </button>
          </div>

          <div className="max-h-[520px] overflow-y-auto space-y-1.5 -mr-2 pr-2">
            {visibleStudents.map((std) => {
              const on = selectedStudent?.id === std.id;
              const isTrainingToday = todayStudentUserIds.includes(std.user_id);
              return (
                <button
                  key={std.id}
                  onClick={() => selectStudent(std)}
                  className={`w-full text-left p-2.5 rounded-2xl transition-all flex items-center gap-3 ${on ? "bg-lime text-black" : "hover:bg-ink-750"}`}
                >
                  <Avatar name={std.user?.name} size={38} tone={on ? "dark" : "dark"} />
                  <div className="min-w-0 flex-1">
                    <div className={`font-semibold text-sm truncate flex items-center gap-1.5 ${on ? "text-black" : "text-white"}`}>
                      <span>{std.user?.name}</span>
                      {isTrainingToday && (
                        <span className="w-2 h-2 rounded-full bg-emerald-400 shrink-0" title="No salão agora" />
                      )}
                    </div>
                    <div className={`text-[11px] flex items-center gap-1.5 ${on ? "text-black/60" : "text-ink-300"}`}>
                      <span className="truncate">{std.plan_name}</span>
                      <span
                        className={`w-1.5 h-1.5 rounded-full shrink-0 ${std.status === "ATIVO" ? (on ? "bg-black" : "bg-lime") : "bg-[#FF5A36]"}`}
                      />
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        </aside>

        {/* Montador de Ficha */}
        <div className="space-y-4 min-w-0">
          {selectedStudent && (
            <>
              {/* Hero do Aluno Selecionado */}
              <div className="relative rounded-4xl overflow-hidden min-h-[200px] animate-fade-up print:bg-white print:text-black">
                <img src={IMG.woman} alt="" className="absolute inset-0 w-full h-full object-cover object-[center_30%] print:hidden" />
                <div className="absolute inset-0 bg-gradient-to-r from-black via-black/85 to-black/30 print:hidden" />
                <div className="relative p-6 sm:p-8 h-full flex flex-col justify-between gap-6 min-h-[200px]">
                  <div className="flex items-center justify-between gap-3">
                    <span className="badge bg-black/50 backdrop-blur text-white print:border print:text-black">
                      {activePlans.length > 0 ? `${activePlans.length} ficha(s) cadastradas` : "Sem ficha ativa"}
                    </span>
                    <span className={selectedStudent.status === "ATIVO" ? "badge-ok" : "badge-bad"}>{selectedStudent.status}</span>
                  </div>

                  <div className="flex flex-col md:flex-row md:items-end md:justify-between gap-5">
                    <div>
                      <div className="text-xs text-lime font-semibold mb-1">Prescrevendo ficha de treino</div>
                      <h2 className="text-3xl font-semibold text-white tracking-tight">{selectedStudent.user?.name}</h2>
                      <div className="text-xs text-ink-200 mt-1">
                        CPF {selectedStudent.user?.document || "N/A"} · {selectedStudent.plan_name} · Peso {selectedStudent.weight || "—"}kg
                      </div>
                    </div>

                    {/* Biblioteca de Modelos e Avaliação Física */}
                    <div className="flex flex-wrap items-center gap-2 print:hidden">
                      <button onClick={openTemplatesModal} className="btn-lime !py-2.5 text-xs shadow-glow">
                        <BookOpen className="w-4 h-4" /> Biblioteca de Modelos
                      </button>
                      <button onClick={openAssessmentModal} className="btn bg-white/10 backdrop-blur text-white hover:bg-white/20 px-3.5 py-2.5 text-xs">
                        <Activity className="w-4 h-4 text-lime" /> Avaliação Física
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              {/* Informações da Ficha */}
              <div className="card p-5 grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="label">Título da ficha</label>
                  <input type="text" value={planTitle} onChange={(e) => setPlanTitle(e.target.value)} className="field" />
                </div>
                <div>
                  <label className="label">Objetivo / foco</label>
                  <input type="text" value={planGoal} onChange={(e) => setPlanGoal(e.target.value)} className="field" />
                </div>
              </div>

              {/* Calculadora de Volume Muscular Semanal */}
              <div className="card p-5 space-y-4 print:hidden">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-2xl bg-lime/10 text-lime flex items-center justify-center shrink-0">
                      <BarChart3 className="w-5 h-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="section-title">Equilíbrio de Volume Muscular</h3>
                        <span className="badge-ok text-[10px]">Diretriz de Hipertrofia</span>
                      </div>
                      <p className="text-xs text-ink-300">
                        Total de séries semanais calculadas em todas as divisões (faixa recomendada: 10 a 20 séries/semana por grupo)
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => setShowVolumeWidget(!showVolumeWidget)}
                    className="btn-icon w-8 h-8 text-ink-400 hover:text-white"
                    title="Expandir/Recolher"
                  >
                    {showVolumeWidget ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                  </button>
                </div>

                {showVolumeWidget && (
                  <div className="pt-2">
                    {volumeByGroup.length === 0 ? (
                      <div className="text-xs text-ink-400 italic py-2">
                        Adicione exercícios às divisões para visualizar a distribuição de séries por grupo muscular.
                      </div>
                    ) : (
                      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                        {volumeByGroup.map((vg) => (
                          <div key={vg.group} className="p-3.5 rounded-2xl bg-ink-750/70 border border-white/5 space-y-2">
                            <div className="flex items-center justify-between">
                              <span className="font-semibold text-white text-sm">{vg.group}</span>
                              <span className="text-sm font-bold text-lime">
                                {vg.sets} <span className="text-[11px] font-normal text-ink-300">séries</span>
                              </span>
                            </div>
                            <div className="w-full bg-ink-850 h-2 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-lime rounded-full transition-all duration-500"
                                style={{ width: `${vg.percent}%` }}
                              />
                            </div>
                            <div className="flex items-center justify-between text-[10px]">
                              <span className="text-ink-400">{vg.exercises} exercício(s)</span>
                              <span className={`px-2 py-0.5 rounded-full border ${vg.badgeClass}`}>{vg.statusLabel}</span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Divisões + Exercícios */}
              <div className="card p-5 space-y-5">
                <div className="flex items-center justify-between gap-3 flex-wrap">
                  <div className="flex items-center gap-2 overflow-x-auto no-scrollbar print:hidden">
                    {workouts.map((w, idx) => {
                      const on = activeWorkoutTab === idx;
                      return (
                        <button key={w.letter} onClick={() => setActiveWorkoutTab(idx)} className={on ? "pill-on" : "pill-off"}>
                          <span className={`w-6 h-6 -ml-2 rounded-full flex items-center justify-center text-xs font-bold ${on ? "bg-black text-lime" : "bg-ink-600 text-white"}`}>
                            {w.letter}
                          </span>
                          Treino {w.letter}
                          <span className={`text-[11px] ${on ? "text-black/60" : "text-ink-400"}`}>{w.items.length}</span>
                        </button>
                      );
                    })}

                    <button
                      onClick={() => {
                        const letters = ["A", "B", "C", "D", "E", "F"];
                        const nextLetter = letters[workouts.length] || "X";
                        setWorkouts([
                          ...workouts,
                          { letter: nextLetter, name: `Divisão ${nextLetter}`, description: "", items: [] },
                        ]);
                      }}
                      className="btn-icon bg-ink-750 text-ink-300 hover:bg-lime hover:text-black shrink-0"
                      title="Adicionar mais uma divisão"
                    >
                      <Plus className="w-5 h-5" />
                    </button>
                  </div>

                  <button
                    onClick={duplicateCurrentWorkout}
                    className="btn-dark !py-2 !px-3 text-xs text-ink-200 hover:text-white shrink-0 print:hidden"
                    title={`Criar uma cópia da Divisão ${currentWorkout?.letter}`}
                  >
                    <Copy className="w-3.5 h-3.5 text-lime" /> Duplicar Treino {currentWorkout?.letter}
                  </button>
                </div>

                <div className="flex items-end justify-between gap-3">
                  <div>
                    <div className="eyebrow">Treino {currentWorkout?.letter}</div>
                    <div className="text-lg font-semibold text-white">{currentWorkout?.name}</div>
                  </div>
                  <span className="text-xs text-ink-300">{currentWorkout?.items.length} exercícios configurados</span>
                </div>

                {currentWorkout?.items.length > 0 ? (
                  <div className="space-y-2.5">
                    {currentWorkout.items.map((it: any, itemIdx: number) => (
                      <div
                        key={itemIdx}
                        className="rounded-3xl bg-ink-750 p-3 flex flex-col xl:flex-row xl:items-center justify-between gap-3 animate-fade-up border border-white/[0.03]"
                      >
                        <div className="flex items-center gap-3 min-w-0">
                          {/* Botões de reordenação */}
                          <div className="flex flex-col gap-0.5 print:hidden">
                            <button
                              onClick={() => moveExercise(itemIdx, "up")}
                              disabled={itemIdx === 0}
                              className="text-ink-400 hover:text-lime disabled:opacity-20 p-0.5"
                              title="Subir na ordem"
                            >
                              <ArrowUp className="w-3.5 h-3.5" />
                            </button>
                            <button
                              onClick={() => moveExercise(itemIdx, "down")}
                              disabled={itemIdx === currentWorkout.items.length - 1}
                              className="text-ink-400 hover:text-lime disabled:opacity-20 p-0.5"
                              title="Descer na ordem"
                            >
                              <ArrowDown className="w-3.5 h-3.5" />
                            </button>
                          </div>

                          <div className="w-11 h-11 rounded-2xl bg-lime text-black flex items-center justify-center font-bold shrink-0">
                            {String(itemIdx + 1).padStart(2, "0")}
                          </div>

                          <div className="min-w-0">
                            <div className="text-sm font-semibold text-white truncate flex items-center gap-2">
                              <span>{it.exercise.name}</span>
                              {it.technique && it.technique !== "Padrão" && (
                                <span className="badge-ok text-[10px] !py-0.5">{it.technique}</span>
                              )}
                            </div>
                            <div className="text-xs text-ink-300 truncate">
                              {it.exercise.muscle_group} · {it.exercise.equipment}
                            </div>
                          </div>
                        </div>

                        {/* Controles de Séries, Reps, Carga, Descanso e Técnica */}
                        <div className="flex flex-wrap items-center gap-2">
                          {/* Seletor de Técnica */}
                          <select
                            value={it.technique || "Padrão"}
                            onChange={(e) => updateItem(itemIdx, { technique: e.target.value })}
                            className="bg-ink-850 text-xs text-lime font-semibold px-3 py-2 rounded-full border border-white/5 focus:outline-none"
                          >
                            <option value="Padrão">Padrão</option>
                            <option value="Bi-Set">Bi-Set</option>
                            <option value="Drop-Set">Drop-Set</option>
                            <option value="Rest-Pause">Rest-Pause</option>
                            <option value="Pirâmide">Pirâmide</option>
                            <option value="Pico de Contração">Pico de Contração</option>
                          </select>

                          {[
                            { k: "Séries", v: it.sets, type: "number", w: "w-8", on: (v: string) => updateItem(itemIdx, { sets: parseInt(v) || 1 }) },
                            { k: "Reps", v: it.reps, type: "text", w: "w-12", on: (v: string) => updateItem(itemIdx, { reps: v }) },
                            { k: "Kg", v: it.load_kg, type: "number", w: "w-10", on: (v: string) => updateItem(itemIdx, { load_kg: parseFloat(v) || 0 }), lime: true },
                            { k: "Desc.", v: it.rest_seconds, type: "number", w: "w-9", on: (v: string) => updateItem(itemIdx, { rest_seconds: parseInt(v) || 30 }), suffix: "s" },
                          ].map((f) => (
                            <label key={f.k} className="flex items-center gap-1.5 bg-ink-850 pl-3 pr-2 py-2 rounded-full border border-white/5">
                              <span className="text-[10px] text-ink-400 uppercase tracking-wide">{f.k}</span>
                              <input
                                type={f.type}
                                value={f.v}
                                onChange={(e) => f.on(e.target.value)}
                                className={`${f.w} bg-transparent font-semibold text-center text-sm focus:outline-none ${f.lime ? "text-lime" : "text-white"}`}
                              />
                              {f.suffix && <span className="text-[10px] text-ink-400 -ml-1">{f.suffix}</span>}
                            </label>
                          ))}

                          <button
                            onClick={() => removeExerciseFromWorkout(itemIdx)}
                            className="btn-icon w-9 h-9 text-ink-400 hover:text-[#FF7A5C] hover:bg-[#FF5A36]/10 print:hidden"
                            title="Remover exercício"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rounded-3xl border-2 border-dashed border-white/10">
                    <Empty icon={<Dumbbell className="w-6 h-6" />}>
                      Nenhum exercício neste treino ainda. Escolha no catálogo abaixo ou aplique um template.
                    </Empty>
                  </div>
                )}
              </div>

              {/* Catálogo de Exercícios */}
              <div className="card p-5 space-y-4 print:hidden">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                  <div>
                    <h3 className="section-title">Catálogo de Exercícios</h3>
                    <div className="text-xs text-ink-300">
                      {exercises.length} disponíveis · Toque para adicionar ao Treino {currentWorkout?.letter}
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <div className="relative">
                      <Search className="w-4 h-4 text-ink-400 absolute left-4 top-1/2 -translate-y-1/2" />
                      <input
                        type="text"
                        placeholder="Buscar exercício..."
                        value={searchExercise}
                        onChange={(e) => setSearchExercise(e.target.value)}
                        className="field !rounded-full !py-2.5 pl-11 sm:w-60"
                      />
                    </div>

                    <button
                      onClick={() => setShowNewExerciseModal(true)}
                      className="btn-lime !py-2.5 !px-4 text-xs shrink-0"
                    >
                      <Plus className="w-4 h-4" /> Criar
                    </button>
                  </div>
                </div>

                {/* Grupos musculares em pills */}
                <div className="flex items-center gap-2 overflow-x-auto no-scrollbar pb-1">
                  <button onClick={() => setSelectedGroup("")} className={selectedGroup === "" ? "pill-on !py-2" : "pill-off !py-2"}>
                    Todos
                  </button>
                  {muscleGroups.map((g) => (
                    <button
                      key={g.group}
                      onClick={() => setSelectedGroup(g.group)}
                      className={selectedGroup === g.group ? "pill-on !py-2" : "pill-off !py-2"}
                    >
                      {g.group}
                      <span className="text-[10px] opacity-60 ml-1">{g.count}</span>
                    </button>
                  ))}
                </div>

                {/* Grade de Exercícios Rápidos */}
                <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-2 max-h-72 overflow-y-auto -mr-2 pr-2">
                  {filteredExercises.slice(0, 36).map((ex) => (
                    <button
                      key={ex.id}
                      onClick={() => addExerciseToCurrentWorkout(ex)}
                      className="text-left p-2.5 pl-4 rounded-full bg-ink-750 hover:bg-ink-700 transition-all flex items-center justify-between gap-2 group"
                    >
                      <div className="truncate">
                        <div className="text-sm font-semibold text-white truncate flex items-center gap-1.5">
                          <span>{ex.name}</span>
                          {ex.video_url && <Video className="w-3 h-3 text-lime shrink-0" />}
                        </div>
                        <div className="text-[11px] text-ink-300 truncate">
                          {ex.muscle_group} · {ex.equipment}
                        </div>
                      </div>
                      <span className="w-8 h-8 rounded-full bg-ink-600 group-hover:bg-lime text-ink-200 group-hover:text-black flex items-center justify-center shrink-0 transition-colors">
                        <Plus className="w-4 h-4" />
                      </span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Barra Fixa de Salvar */}
              <div className="sticky bottom-4 z-30 print:hidden">
                <div className="rounded-full bg-ink-850/90 backdrop-blur-xl border border-white/[0.08] p-2.5 pl-6 flex items-center justify-between gap-3 shadow-card">
                  {saveSuccess ? (
                    <div className="text-sm text-lime font-semibold flex items-center gap-2">
                      <CheckCircle className="w-4 h-4" /> Treino salvo e ativado no app do aluno!
                    </div>
                  ) : (
                    <div className="text-xs text-ink-300 hidden sm:block">
                      O aluno receberá a atualização automaticamente no aplicativo.
                    </div>
                  )}

                  <div className="flex items-center gap-2 ml-auto">
                    <button
                      type="button"
                      onClick={shareWorkoutViaWhatsApp}
                      className="btn-dark !py-3 !px-4 text-xs font-semibold"
                      title="Enviar treino formatado no WhatsApp do aluno"
                    >
                      <Share2 className="w-4 h-4 text-lime" />
                      <span className="hidden sm:inline">Enviar WhatsApp</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => {
                        setSaveTmplTitle(planTitle);
                        setSaveTmplGoal(planGoal || "Hipertrofia");
                        setShowSaveTemplateModal(true);
                      }}
                      className="btn-dark !py-3 !px-4 text-xs font-semibold"
                      title="Salvar configuração atual como modelo reutilizável para a academia"
                    >
                      <Bookmark className="w-4 h-4 text-lime" />
                      <span className="hidden sm:inline">Salvar como Modelo</span>
                    </button>

                    <button
                      onClick={handleSaveWorkoutPlan}
                      disabled={saving || workouts[activeWorkoutTab]?.items.length === 0}
                      className="btn-lime !py-3 font-bold"
                    >
                      <Save className="w-4 h-4" />
                      {saving ? "Salvando..." : "Salvar Ficha de Treino"}
                    </button>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* MODAL: CADASTRO DE NOVO EXERCÍCIO */}
      {showNewExerciseModal && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-ink-850 border border-white/10 rounded-4xl max-w-md w-full p-6 space-y-4 shadow-card animate-fade-up">
            <div className="flex items-center justify-between pb-2 border-b border-white/5">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Dumbbell className="w-5 h-5 text-lime" /> Novo Exercício Customizado
              </h3>
              <button onClick={() => setShowNewExerciseModal(false)} className="btn-icon w-8 h-8 bg-ink-700 text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleCreateExercise} className="space-y-3">
              <div>
                <label className="label">Nome do Exercício</label>
                <input
                  type="text"
                  required
                  placeholder="Ex: Elevação Pélvica com Barra"
                  value={newExercise.name}
                  onChange={(e) => setNewExercise({ ...newExercise, name: e.target.value })}
                  className="field"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Grupo Muscular</label>
                  <select
                    value={newExercise.muscle_group}
                    onChange={(e) => setNewExercise({ ...newExercise, muscle_group: e.target.value })}
                    className="field"
                  >
                    {["Peitoral", "Costas", "Pernas", "Ombros", "Bíceps", "Tríceps", "Abdômen", "Glúteos", "Cardio"].map((g) => (
                      <option key={g} value={g}>{g}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="label">Equipamento</label>
                  <input
                    type="text"
                    placeholder="Ex: Máquina, Barra..."
                    value={newExercise.equipment}
                    onChange={(e) => setNewExercise({ ...newExercise, equipment: e.target.value })}
                    className="field"
                  />
                </div>
              </div>

              <div>
                <label className="label">Link do Vídeo (YouTube ou Vimeo)</label>
                <input
                  type="url"
                  placeholder="https://www.youtube.com/watch?v=..."
                  value={newExercise.video_url}
                  onChange={(e) => setNewExercise({ ...newExercise, video_url: e.target.value })}
                  className="field"
                />
              </div>

              <div>
                <label className="label">Instruções de Execução / Dicas</label>
                <textarea
                  rows={3}
                  placeholder="Descreva a postura, cadência e cuidados..."
                  value={newExercise.description}
                  onChange={(e) => setNewExercise({ ...newExercise, description: e.target.value })}
                  className="field resize-none"
                />
              </div>

              <div className="flex items-center gap-2 pt-2">
                <button type="button" onClick={() => setShowNewExerciseModal(false)} className="btn-dark flex-1">
                  Cancelar
                </button>
                <button type="submit" disabled={creatingExercise} className="btn-lime flex-1">
                  {creatingExercise ? "Cadastrando..." : "Cadastrar Exercício"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: SALÃO AO VIVO (TRACKER DE ALUNOS NO SALÃO) */}
      {showFloorModal && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-ink-850 border border-white/10 rounded-4xl max-w-2xl w-full p-6 space-y-5 shadow-card animate-fade-up max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-white/5">
              <div className="flex items-center gap-3">
                <span className="relative w-9 h-9 rounded-full bg-lime/15 flex items-center justify-center">
                  <span className="w-2.5 h-2.5 rounded-full bg-lime animate-ping absolute" />
                  <span className="w-2.5 h-2.5 rounded-full bg-lime" />
                </span>
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    Entradas de hoje
                  </h3>
                  <p className="text-xs text-ink-300">
                    {todayCheckins.length} aluno(s) com check-in registrado hoje
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowFloorModal(false)}
                className="btn-icon w-8 h-8 bg-ink-700 text-white"
                title="Fechar"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto space-y-2 pr-1 -mr-1">
              {todayCheckins.length === 0 ? (
                <div className="text-center py-12 text-ink-400">
                  <Users className="w-12 h-12 mx-auto mb-2 opacity-30" />
                  <p className="text-sm">Nenhum check-in registrado no dia de hoje.</p>
                </div>
              ) : (
                todayCheckins.map((chk: any) => {
                  const std = students.find((s) => s.id === chk.student_id) || chk.student;
                  const name = std?.user?.name || "Aluno";
                  const phone = std?.user?.phone || "";
                  const checkinTime = chk.created_at
                    ? new Date(chk.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
                    : "Hoje";
                  const elapsed = getElapsedTime(chk.created_at);

                  return (
                    <div
                      key={chk.id}
                      className="p-3.5 rounded-3xl bg-ink-750 hover:bg-ink-700/80 transition-all flex items-center justify-between gap-3 border border-white/5"
                    >
                      <div className="flex items-center gap-3 min-w-0">
                        <Avatar name={name} size={42} tone="dark" />
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-white text-sm truncate">{name}</span>
                            <span className="badge-ok text-[10px] !py-0.5">No salão</span>
                          </div>
                          <div className="text-xs text-ink-300 flex items-center gap-2 mt-0.5">
                            <span className="flex items-center gap-1">
                              <Clock className="w-3 h-3 text-ink-400" /> Entrou às {checkinTime} ({elapsed})
                            </span>
                            <span>·</span>
                            <span className="truncate">{std?.plan_name || "Plano Livre"}</span>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 shrink-0">
                        {phone && (
                          <a
                            href={`https://wa.me/55${phone.replace(/\D/g, "")}?text=${encodeURIComponent(`Olá ${name}! Vi que você está treinando agora na GymSync. Precisa de algum auxílio no treino de hoje? 💪`)}`}
                            target="_blank"
                            rel="noreferrer"
                            className="btn-dark !py-2 !px-3 text-xs"
                            title="Conversar no WhatsApp"
                          >
                            <MessageSquare className="w-3.5 h-3.5 text-lime" />
                            <span className="hidden sm:inline">WhatsApp</span>
                          </a>
                        )}

                        <button
                          onClick={() => {
                            if (std) {
                              const found = students.find((s) => s.id === std.id);
                              if (found) selectStudent(found);
                            }
                            setShowFloorModal(false);
                            toast.success(`Carregando ficha de ${name}`);
                          }}
                          className="btn-lime !py-2 !px-3 text-xs font-semibold"
                        >
                          Abrir Ficha
                        </button>
                      </div>
                    </div>
                  );
                })
              )}
            </div>

            <div className="pt-2 border-t border-white/5 flex items-center justify-between text-xs text-ink-300">
              <span>Atualizado em tempo real conforme os alunos realizam check-in na catraca/app.</span>
              <button
                onClick={() => setShowFloorModal(false)}
                className="btn-dark !py-1.5 !px-4"
              >
                Fechar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: BIBLIOTECA DE MODELOS DE TREINO */}
      {showTemplatesModal && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 animate-fade-in">
          <div className="bg-ink-900 border border-white/10 rounded-3xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
            {/* Cabeçalho */}
            <div className="p-5 border-b border-white/5 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-2xl bg-lime/15 text-lime flex items-center justify-center">
                  <BookOpen className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    Biblioteca de Modelos de Treino
                    <span className="text-xs px-2 py-0.5 rounded-full bg-white/10 text-ink-200 font-normal">
                      {templateList.length} disponíveis
                    </span>
                  </h3>
                  <p className="text-xs text-ink-300">Escolha uma ficha pronta para aplicar no aluno com 1 clique</p>
                </div>
              </div>
              <button
                onClick={() => setShowTemplatesModal(false)}
                className="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center text-ink-300 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Filtros e Busca */}
            <div className="p-4 border-b border-white/5 bg-ink-850/50 flex flex-wrap items-center gap-3">
              <div className="relative flex-1 min-w-[200px]">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-ink-400" />
                <input
                  type="text"
                  placeholder="Buscar por título ou foco..."
                  value={templateSearch}
                  onChange={(e) => setTemplateSearch(e.target.value)}
                  className="field !py-2 !pl-9 text-xs"
                />
              </div>

              <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar py-0.5">
                {["TODOS", "Hipertrofia", "Adaptação", "Emagrecimento"].map((g) => (
                  <button
                    key={g}
                    onClick={() => setTemplateGoalFilter(g)}
                    className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-all ${
                      templateGoalFilter === g ? "bg-lime text-black" : "bg-ink-750 text-ink-300 hover:text-white"
                    }`}
                  >
                    {g}
                  </button>
                ))}
              </div>
            </div>

            {/* Conteúdo Principal Split View */}
            <div className="flex-1 min-h-0 grid grid-cols-1 md:grid-cols-12 divide-y md:divide-y-0 md:divide-x divide-white/5 overflow-hidden">
              {/* Lista de Modelos (Esquerda) */}
              <div className="md:col-span-5 p-4 overflow-y-auto space-y-2.5 max-h-[50vh] md:max-h-none">
                {loadingTemplates ? (
                  <div className="p-8 text-center text-xs text-ink-300">Carregando modelos...</div>
                ) : (
                  (() => {
                    const filtered = templateList.filter((t) => {
                      const matchSearch =
                        !templateSearch ||
                        t.title.toLowerCase().includes(templateSearch.toLowerCase()) ||
                        t.goal.toLowerCase().includes(templateSearch.toLowerCase());
                      const matchGoal =
                        templateGoalFilter === "TODOS" ||
                        t.goal.toLowerCase().includes(templateGoalFilter.toLowerCase());
                      return matchSearch && matchGoal;
                    });

                    if (filtered.length === 0) {
                      return <Empty title="Nenhum modelo encontrado">Tente mudar o filtro ou busca.</Empty>;
                    }

                    return filtered.map((tmpl) => {
                      const isSelected = previewTemplate?.id === tmpl.id;
                      const totalExercises = tmpl.workouts?.reduce((acc, w) => acc + (w.items?.length || 0), 0) || 0;
                      return (
                        <div
                          key={tmpl.id}
                          onClick={() => setPreviewTemplate(tmpl)}
                          className={`p-3.5 rounded-2xl cursor-pointer transition-all border text-left ${
                            isSelected
                              ? "bg-lime/10 border-lime/40 text-white"
                              : "bg-ink-800/60 border-white/5 hover:border-white/10 text-ink-200"
                          }`}
                        >
                          <div className="flex items-start justify-between gap-2">
                            <h4 className="text-sm font-bold text-white leading-tight">{tmpl.title}</h4>
                            {tmpl.gym_id && (
                              <button
                                onClick={(e) => handleDeleteTemplate(tmpl.id, e)}
                                className="text-ink-400 hover:text-red-400 p-1"
                                title="Excluir este modelo"
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </button>
                            )}
                          </div>

                          <div className="flex flex-wrap items-center gap-1.5 mt-2">
                            <span className="px-2 py-0.5 rounded-lg bg-white/5 text-[10px] text-lime font-semibold">
                              {tmpl.goal}
                            </span>
                            <span className="px-2 py-0.5 rounded-lg bg-white/5 text-[10px] text-ink-300">
                              {tmpl.level}
                            </span>
                            <span className="text-[10px] text-ink-400 ml-auto">
                              {tmpl.workouts?.length || 0} divisões · {totalExercises} exs
                            </span>
                          </div>
                        </div>
                      );
                    });
                  })()
                )}
              </div>

              {/* Preview Detalhado (Direita) */}
              <div className="md:col-span-7 p-5 overflow-y-auto max-h-[50vh] md:max-h-none flex flex-col justify-between">
                {previewTemplate ? (
                  <div className="space-y-4">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="badge-lime text-[10px]">{previewTemplate.goal}</span>
                        <span className="badge-mute text-[10px]">{previewTemplate.level}</span>
                        {previewTemplate.gym_id ? (
                          <span className="badge bg-amber-500/15 text-amber-300 text-[10px]">Modelo da Academia</span>
                        ) : (
                          <span className="badge bg-blue-500/15 text-blue-300 text-[10px]">Padrão Oficial</span>
                        )}
                      </div>
                      <h3 className="text-xl font-bold text-white mt-1.5">{previewTemplate.title}</h3>
                      {previewTemplate.description && (
                        <p className="text-xs text-ink-300 mt-1">{previewTemplate.description}</p>
                      )}
                    </div>

                    {/* Divisões e Exercícios */}
                    <div className="space-y-3 pt-2">
                      {previewTemplate.workouts?.map((w) => (
                        <div key={w.letter} className="p-3.5 rounded-2xl bg-ink-800/80 border border-white/5 space-y-2">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <span className="w-6 h-6 rounded-lg bg-lime text-black font-black text-xs flex items-center justify-center">
                                {w.letter}
                              </span>
                              <span className="text-xs font-bold text-white">{w.name}</span>
                            </div>
                            <span className="text-[10px] text-ink-400">{w.items?.length || 0} exercícios</span>
                          </div>

                          <div className="space-y-1.5 pt-1">
                            {w.items?.map((it, idx) => {
                              const exName = it.exercise?.name || "Exercício";
                              return (
                                <div
                                  key={idx}
                                  className="flex items-center justify-between text-xs py-1 px-2.5 rounded-xl bg-ink-750/50"
                                >
                                  <span className="text-ink-200 truncate pr-2">
                                    {idx + 1}. {exName}
                                  </span>
                                  <span className="text-ink-400 shrink-0 text-[11px] font-mono">
                                    {it.sets}x {it.reps} · {it.rest_seconds}s
                                  </span>
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <Empty title="Selecione um modelo">Clique em um modelo à esquerda para ver os exercícios.</Empty>
                )}

                {previewTemplate && (
                  <div className="pt-4 mt-4 border-t border-white/5 flex items-center justify-end gap-3">
                    <button
                      onClick={() => setShowTemplatesModal(false)}
                      className="btn-ghost !py-2.5 !px-4 text-xs"
                    >
                      Cancelar
                    </button>
                    <button
                      onClick={() => applyTemplateFromLibrary(previewTemplate)}
                      className="btn-lime !py-2.5 !px-5 text-xs font-bold shadow-glow"
                    >
                      <Sparkles className="w-4 h-4" /> Aplicar na Ficha do Aluno
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: SALVAR TREINO COMO NOVO MODELO */}
      {showSaveTemplateModal && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 animate-fade-in">
          <div className="bg-ink-900 border border-white/10 rounded-3xl w-full max-w-lg p-6 space-y-5 shadow-2xl">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-2xl bg-lime/15 text-lime flex items-center justify-center">
                  <Bookmark className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-white">Salvar como Modelo</h3>
                  <p className="text-xs text-ink-300">Este treino ficará disponível para prescrever a outros alunos</p>
                </div>
              </div>
              <button
                onClick={() => setShowSaveTemplateModal(false)}
                className="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center text-ink-300 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="label">Título do Modelo</label>
                <input
                  type="text"
                  placeholder="Ex: Treino ABC Foco Dorsal e Bíceps"
                  value={saveTmplTitle}
                  onChange={(e) => setSaveTmplTitle(e.target.value)}
                  className="field"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Objetivo</label>
                  <select
                    value={saveTmplGoal}
                    onChange={(e) => setSaveTmplGoal(e.target.value)}
                    className="field"
                  >
                    <option value="Hipertrofia">Hipertrofia</option>
                    <option value="Adaptação">Adaptação</option>
                    <option value="Emagrecimento">Emagrecimento</option>
                    <option value="Força">Força</option>
                    <option value="Funcional">Funcional</option>
                  </select>
                </div>
                <div>
                  <label className="label">Nível</label>
                  <select
                    value={saveTmplLevel}
                    onChange={(e) => setSaveTmplLevel(e.target.value)}
                    className="field"
                  >
                    <option value="Iniciante">Iniciante</option>
                    <option value="Intermediário">Intermediário</option>
                    <option value="Avançado">Avançado</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="label">Descrição / Observações (opcional)</label>
                <textarea
                  rows={2}
                  placeholder="Explicação do método, divisão de dias recomendada..."
                  value={saveTmplDesc}
                  onChange={(e) => setSaveTmplDesc(e.target.value)}
                  className="field"
                />
              </div>

              <div className="p-3 rounded-2xl bg-ink-800 border border-white/5 text-xs text-ink-300 flex items-center justify-between">
                <span>Total de divisões: <strong className="text-white">{workouts.length}</strong></span>
                <span>Total de exercícios: <strong className="text-white">{workouts.reduce((acc, w) => acc + w.items.length, 0)}</strong></span>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setShowSaveTemplateModal(false)}
                className="btn-ghost !py-2.5 !px-4 text-xs"
              >
                Cancelar
              </button>
              <button
                type="button"
                onClick={handleSaveAsTemplate}
                disabled={savingTemplate || !saveTmplTitle.trim()}
                className="btn-lime !py-2.5 !px-5 text-xs font-bold"
              >
                {savingTemplate ? "Salvando..." : "Salvar na Biblioteca"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: AVALIAÇÃO FÍSICA & MEDIDAS DO ALUNO */}
      {showAssessmentModal && selectedStudent && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4 animate-fade-in">
          <div className="bg-ink-900 border border-white/10 rounded-3xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
            <div className="p-5 border-b border-white/5 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-2xl bg-lime/15 text-lime flex items-center justify-center">
                  <Activity className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-white">
                    Avaliação Física · {selectedStudent.user?.name}
                  </h3>
                  <p className="text-xs text-ink-300">Histórico de composição corporal e circunferências</p>
                </div>
              </div>
              <button
                onClick={() => setShowAssessmentModal(false)}
                className="w-8 h-8 rounded-full bg-white/5 hover:bg-white/10 flex items-center justify-center text-ink-300 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-6 overflow-y-auto space-y-6">
              {!showNewAssessmentForm ? (
                <>
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-ink-300 uppercase tracking-wider">
                      {assessments.length} avaliação(ões) registrada(s)
                    </span>
                    <button
                      onClick={() => setShowNewAssessmentForm(true)}
                      className="btn-lime !py-2 !px-4 text-xs font-bold"
                    >
                      <Plus className="w-4 h-4" /> Nova Avaliação
                    </button>
                  </div>

                  {loadingAssessments ? (
                    <div className="p-8 text-center text-xs text-ink-300">Carregando avaliações...</div>
                  ) : assessments.length === 0 ? (
                    <Empty title="Nenhuma avaliação física cadastrada">
                      Clique em "Nova Avaliação" para cadastrar o peso e medidas deste aluno.
                    </Empty>
                  ) : (
                    <div className="space-y-4">
                      {assessments.map((a, idx) => (
                        <div key={a.id} className="p-4 rounded-2xl bg-ink-800 border border-white/5 space-y-3">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-bold text-white">
                                {new Date(a.date).toLocaleDateString("pt-BR")}
                              </span>
                              {idx === 0 && <span className="badge-ok text-[10px]">Atual</span>}
                            </div>
                            <div className="text-xs text-ink-300">
                              {a.evaluator?.name ? `Avaliador(a): ${a.evaluator.name}` : "Auto-avaliação"}
                            </div>
                          </div>

                          <div className="grid grid-cols-4 gap-2 pt-1 text-center">
                            <div className="p-2.5 rounded-xl bg-ink-750">
                              <div className="text-[10px] text-ink-300">Peso</div>
                              <div className="text-sm font-bold text-white mt-0.5">{a.weight} kg</div>
                            </div>
                            <div className="p-2.5 rounded-xl bg-ink-750">
                              <div className="text-[10px] text-ink-300">Altura</div>
                              <div className="text-sm font-bold text-white mt-0.5">{a.height} cm</div>
                            </div>
                            <div className="p-2.5 rounded-xl bg-ink-750">
                              <div className="text-[10px] text-ink-300">% Gordura</div>
                              <div className="text-sm font-bold text-lime mt-0.5">{a.body_fat ? `${a.body_fat}%` : "—"}</div>
                            </div>
                            <div className="p-2.5 rounded-xl bg-ink-750">
                              <div className="text-[10px] text-ink-300">Massa Magra</div>
                              <div className="text-sm font-bold text-white mt-0.5">{a.muscle_mass ? `${a.muscle_mass} kg` : "—"}</div>
                            </div>
                          </div>

                          {(a.waist || a.chest || a.hips || a.right_arm) && (
                            <div className="pt-2 border-t border-white/5 grid grid-cols-3 sm:grid-cols-6 gap-1.5 text-center text-xs">
                              <div className="p-1.5 rounded-lg bg-ink-750/60">
                                <span className="text-[10px] text-ink-400 block">Tórax</span>
                                <span className="font-semibold text-white">{a.chest || "—"} cm</span>
                              </div>
                              <div className="p-1.5 rounded-lg bg-ink-750/60">
                                <span className="text-[10px] text-ink-400 block">Cintura</span>
                                <span className="font-semibold text-white">{a.waist || "—"} cm</span>
                              </div>
                              <div className="p-1.5 rounded-lg bg-ink-750/60">
                                <span className="text-[10px] text-ink-400 block">Abdômen</span>
                                <span className="font-semibold text-white">{a.abdomen || "—"} cm</span>
                              </div>
                              <div className="p-1.5 rounded-lg bg-ink-750/60">
                                <span className="text-[10px] text-ink-400 block">Quadril</span>
                                <span className="font-semibold text-white">{a.hips || "—"} cm</span>
                              </div>
                              <div className="p-1.5 rounded-lg bg-ink-750/60">
                                <span className="text-[10px] text-ink-400 block">Braço D</span>
                                <span className="font-semibold text-white">{a.right_arm || "—"} cm</span>
                              </div>
                              <div className="p-1.5 rounded-lg bg-ink-750/60">
                                <span className="text-[10px] text-ink-400 block">Coxa D</span>
                                <span className="font-semibold text-white">{a.right_thigh || "—"} cm</span>
                              </div>
                            </div>
                          )}

                          {a.notes && (
                            <p className="text-xs text-ink-300 italic pt-1">"{a.notes}"</p>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </>
              ) : (
                <div className="space-y-4 animate-fade-in">
                  <div className="flex items-center justify-between pb-2 border-b border-white/5">
                    <h4 className="text-sm font-bold text-white">Nova Medição / Avaliação</h4>
                    <button
                      onClick={() => setShowNewAssessmentForm(false)}
                      className="text-xs text-ink-300 hover:text-white"
                    >
                      ← Voltar ao histórico
                    </button>
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    <div>
                      <label className="label">Peso (kg)</label>
                      <input
                        type="number"
                        step="0.1"
                        value={newAssessment.weight}
                        onChange={(e) => setNewAssessment({ ...newAssessment, weight: parseFloat(e.target.value) || 0 })}
                        className="field"
                      />
                    </div>
                    <div>
                      <label className="label">Altura (cm)</label>
                      <input
                        type="number"
                        value={newAssessment.height}
                        onChange={(e) => setNewAssessment({ ...newAssessment, height: parseFloat(e.target.value) || 0 })}
                        className="field"
                      />
                    </div>
                    <div>
                      <label className="label">% Gordura</label>
                      <input
                        type="number"
                        step="0.1"
                        value={newAssessment.body_fat}
                        onChange={(e) => setNewAssessment({ ...newAssessment, body_fat: parseFloat(e.target.value) || 0 })}
                        className="field"
                      />
                    </div>
                    <div>
                      <label className="label">Massa Magra (kg)</label>
                      <input
                        type="number"
                        step="0.1"
                        value={newAssessment.muscle_mass}
                        onChange={(e) => setNewAssessment({ ...newAssessment, muscle_mass: parseFloat(e.target.value) || 0 })}
                        className="field"
                      />
                    </div>
                  </div>

                  <div className="pt-2">
                    <div className="text-xs font-semibold text-ink-300 uppercase tracking-wider mb-2">
                      Circunferências Corporais (cm)
                    </div>
                    <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                      <div>
                        <label className="label">Tórax</label>
                        <input
                          type="number"
                          step="0.5"
                          value={newAssessment.chest}
                          onChange={(e) => setNewAssessment({ ...newAssessment, chest: parseFloat(e.target.value) || 0 })}
                          className="field"
                        />
                      </div>
                      <div>
                        <label className="label">Cintura</label>
                        <input
                          type="number"
                          step="0.5"
                          value={newAssessment.waist}
                          onChange={(e) => setNewAssessment({ ...newAssessment, waist: parseFloat(e.target.value) || 0 })}
                          className="field"
                        />
                      </div>
                      <div>
                        <label className="label">Abdômen</label>
                        <input
                          type="number"
                          step="0.5"
                          value={newAssessment.abdomen}
                          onChange={(e) => setNewAssessment({ ...newAssessment, abdomen: parseFloat(e.target.value) || 0 })}
                          className="field"
                        />
                      </div>
                      <div>
                        <label className="label">Quadril</label>
                        <input
                          type="number"
                          step="0.5"
                          value={newAssessment.hips}
                          onChange={(e) => setNewAssessment({ ...newAssessment, hips: parseFloat(e.target.value) || 0 })}
                          className="field"
                        />
                      </div>
                      <div>
                        <label className="label">Braço Direito</label>
                        <input
                          type="number"
                          step="0.5"
                          value={newAssessment.right_arm}
                          onChange={(e) => setNewAssessment({ ...newAssessment, right_arm: parseFloat(e.target.value) || 0, left_arm: parseFloat(e.target.value) || 0 })}
                          className="field"
                        />
                      </div>
                      <div>
                        <label className="label">Coxa Direita</label>
                        <input
                          type="number"
                          step="0.5"
                          value={newAssessment.right_thigh}
                          onChange={(e) => setNewAssessment({ ...newAssessment, right_thigh: parseFloat(e.target.value) || 0, left_thigh: parseFloat(e.target.value) || 0 })}
                          className="field"
                        />
                      </div>
                    </div>
                  </div>

                  <div>
                    <label className="label">Observações da Avaliação</label>
                    <textarea
                      rows={2}
                      placeholder="Ex: Aluno em fase de perda de gordura, sem dores articulares..."
                      value={newAssessment.notes}
                      onChange={(e) => setNewAssessment({ ...newAssessment, notes: e.target.value })}
                      className="field"
                    />
                  </div>

                  <div className="flex items-center justify-end gap-3 pt-2">
                    <button
                      type="button"
                      onClick={() => setShowNewAssessmentForm(false)}
                      className="btn-ghost !py-2.5 !px-4 text-xs"
                    >
                      Cancelar
                    </button>
                    <button
                      type="button"
                      onClick={handleCreateAssessment}
                      disabled={savingAssessment}
                      className="btn-lime !py-2.5 !px-5 text-xs font-bold"
                    >
                      {savingAssessment ? "Salvando..." : "Salvar Avaliação"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function Prescription() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-ink-300">Carregando treinos...</div>}>
      <ProfessorContent />
    </Suspense>
  );
}
