"use client";

import React, { useEffect, useState, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { 
  Dumbbell, UserCheck, Plus, Trash2, Search, CheckCircle, 
  Layers, Clock, Sparkles, Filter, ChevronRight, Save, User 
} from "lucide-react";
import { api } from "@/services/api";
import { Exercise, Student, WorkoutPlan } from "@/types";

function ProfessorContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialStudentId = searchParams.get("studentId");

  const [students, setStudents] = useState<Student[]>([]);
  const [selectedStudent, setSelectedStudent] = useState<Student | null>(null);
  const [exercises, setExercises] = useState<Exercise[]>([]);
  const [muscleGroups, setMuscleGroups] = useState<{ group: string; count: number }[]>([]);
  const [selectedGroup, setSelectedGroup] = useState("");
  const [searchExercise, setSearchExercise] = useState("");
  const [activePlans, setActivePlans] = useState<WorkoutPlan[]>([]);
  const [todayCheckins, setTodayCheckins] = useState<any[]>([]);

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

  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      const [stdRes, exRes, chkRes] = await Promise.all([
        api.listStudents("", "", 1, 100),
        api.listExercises(""),
        api.getTodayCheckIns(),
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
      if (err.message?.includes("401") || err.message?.includes("Acesso negado")) {
        router.push("/");
      }
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

      // Pega exercícios de peito, costas e pernas da base
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
            { exercise: supino, sets: 4, reps: "8-10", load_kg: 60, rest_seconds: 90, notes: "Cadência controlada" },
            { exercise: crossover, sets: 3, reps: "12-15", load_kg: 25, rest_seconds: 60, notes: "Pico de contração" },
            { exercise: triceps, sets: 4, reps: "12", load_kg: 35, rest_seconds: 45, notes: "" },
          ],
        },
        {
          letter: "B",
          name: "Costas, Bíceps e Trapézio",
          description: "Pull Day",
          items: [
            { exercise: puxada, sets: 4, reps: "10-12", load_kg: 50, rest_seconds: 75, notes: "" },
            { exercise: remada, sets: 4, reps: "8-10", load_kg: 55, rest_seconds: 75, notes: "" },
            { exercise: rosca, sets: 3, reps: "10-12", load_kg: 28, rest_seconds: 60, notes: "" },
          ],
        },
        {
          letter: "C",
          name: "Pernas Completo e Abdômen",
          description: "Legs & Core Day",
          items: [
            { exercise: agachamento, sets: 4, reps: "8-10", load_kg: 80, rest_seconds: 120, notes: "Amplitude completa" },
            { exercise: leg, sets: 4, reps: "12", load_kg: 180, rest_seconds: 90, notes: "" },
          ],
        },
      ]);
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
            { exercise: supino, sets: 3, reps: "12-15", load_kg: 30, rest_seconds: 60, notes: "" },
            { exercise: puxada, sets: 3, reps: "12-15", load_kg: 35, rest_seconds: 60, notes: "" },
          ],
        },
        {
          letter: "B",
          name: "Membros Inferiores & Core",
          description: "Pernas e Abdômen",
          items: [
            { exercise: leg, sets: 3, reps: "12-15", load_kg: 80, rest_seconds: 60, notes: "" },
          ],
        },
      ]);
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
      notes: "",
    });
    setWorkouts(updated);
  };

  const removeExerciseFromWorkout = (itemIndex: number) => {
    const updated = [...workouts];
    updated[activeWorkoutTab].items.splice(itemIndex, 1);
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
            notes: it.notes,
          })),
        })),
      };

      await api.createWorkoutPlan(selectedStudent.id, payload);
      setSaveSuccess(true);
      const plans = await api.getStudentWorkouts(selectedStudent.id);
      setActivePlans(plans);
    } catch (err: any) {
      alert("Erro ao salvar treino: " + err.message);
    } finally {
      setSaving(false);
    }
  };

  const filteredExercises = exercises.filter((ex) => {
    const matchesGroup = selectedGroup ? ex.muscle_group === selectedGroup : true;
    const matchesSearch = searchExercise
      ? ex.name.toLowerCase().includes(searchExercise.toLowerCase())
      : true;
    return matchesGroup && matchesSearch;
  });

  return (
    <div className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      {/* Cabeçalho */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold text-white flex items-center gap-2">
            Portal do Professor / Treinador
            <span className="text-xs px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
              Personal
            </span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Prescrição ágil com templates prontos, catálogo de mais de 110 exercícios e acompanhamento do salão.
          </p>
        </div>

        {/* Live Salão Bar */}
        <div className="bg-slate-900 border border-slate-800 px-4 py-2 rounded-xl flex items-center gap-3">
          <div className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse" />
          <div className="text-xs">
            <span className="text-slate-400">Treinando Agora:</span>{" "}
            <b className="text-white">{todayCheckins.length} alunos</b>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
        {/* Coluna 1: Seleção de Aluno */}
        <div className="lg:col-span-1 space-y-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4 space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-slate-800">
              <h3 className="text-sm font-bold text-white flex items-center gap-2">
                <User className="w-4 h-4 text-emerald-400" /> Selecionar Aluno
              </h3>
              <span className="text-xs text-slate-500">{students.length}</span>
            </div>

            <div className="max-h-[600px] overflow-y-auto space-y-2 pr-1">
              {students.map((std) => (
                <button
                  key={std.id}
                  onClick={() => selectStudent(std)}
                  className={`w-full text-left p-3 rounded-xl border transition-all ${selectedStudent?.id === std.id ? "bg-emerald-500/10 border-emerald-500/50 text-white" : "bg-slate-800/40 border-slate-700/50 text-slate-300 hover:bg-slate-800"}`}
                >
                  <div className="font-semibold text-xs text-white">{std.user?.name}</div>
                  <div className="text-[10px] text-slate-400 flex items-center justify-between mt-1">
                    <span>{std.plan_name}</span>
                    <span className={std.status === "ATIVO" ? "text-emerald-400 font-semibold" : "text-red-400"}>
                      {std.status}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Colunas 2 e 3: Montador de Ficha */}
        <div className="lg:col-span-3 space-y-6">
          {selectedStudent && (
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-6">
              {/* Header do Aluno Selecionado e Ações de Template */}
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-slate-800 pb-4">
                <div>
                  <div className="text-xs text-slate-400">Prescrevendo treino para:</div>
                  <h2 className="text-lg font-bold text-white flex items-center gap-2">
                    {selectedStudent.user?.name}
                    <span className="text-xs font-normal text-slate-400">
                      (CPF: {selectedStudent.user?.document || "N/A"})
                    </span>
                  </h2>
                </div>

                {/* Templates Rápidos */}
                <div className="flex items-center gap-2">
                  <span className="text-xs text-slate-400">Templates:</span>
                  <button
                    onClick={() => applyTemplate("ABC")}
                    className="px-3 py-1 rounded-lg bg-blue-600/20 hover:bg-blue-600/30 text-blue-400 border border-blue-500/30 text-xs font-semibold flex items-center gap-1 transition-colors"
                  >
                    <Sparkles className="w-3 h-3" /> Ficha ABC
                  </button>
                  <button
                    onClick={() => applyTemplate("AB")}
                    className="px-3 py-1 rounded-lg bg-purple-600/20 hover:bg-purple-600/30 text-purple-400 border border-purple-500/30 text-xs font-semibold flex items-center gap-1 transition-colors"
                  >
                    <Sparkles className="w-3 h-3" /> Ficha AB
                  </button>
                </div>
              </div>

              {/* Informações da Ficha */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1">Título da Ficha</label>
                  <input
                    type="text"
                    value={planTitle}
                    onChange={(e) => setPlanTitle(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1">Objetivo / Foco</label>
                  <input
                    type="text"
                    value={planGoal}
                    onChange={(e) => setPlanGoal(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white"
                  />
                </div>
              </div>

              {/* Abas das Divisões de Treino (A, B, C...) */}
              <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
                {workouts.map((w, idx) => (
                  <button
                    key={w.letter}
                    onClick={() => setActiveWorkoutTab(idx)}
                    className={`px-4 py-2 rounded-xl text-xs font-bold transition-all flex items-center gap-2 ${activeWorkoutTab === idx ? "bg-emerald-600 text-white shadow-lg shadow-emerald-600/20" : "bg-slate-800 text-slate-400 hover:text-white"}`}
                  >
                    <span>Treino {w.letter}</span>
                    <span className="text-[10px] opacity-70">({w.items.length} ex)</span>
                  </button>
                ))}

                <button
                  onClick={() => {
                    const letters = ["A", "B", "C", "D", "E", "F"];
                    const nextLetter = letters[workouts.length] || "X";
                    setWorkouts([
                      ...workouts,
                      { letter: nextLetter, name: `Divisão ${nextLetter}`, description: "", items: [] },
                    ]);
                  }}
                  className="p-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white text-xs transition-colors"
                  title="Adicionar mais uma divisão (Ex: Treino C ou D)"
                >
                  <Plus className="w-4 h-4" />
                </button>
              </div>

              {/* Itens do Treino Ativo */}
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-xs font-bold text-white uppercase tracking-wider">
                    Exercícios do Treino {workouts[activeWorkoutTab]?.letter} — {workouts[activeWorkoutTab]?.name}
                  </h3>
                  <span className="text-xs text-slate-400">
                    {workouts[activeWorkoutTab]?.items.length} exercícios configurados
                  </span>
                </div>

                {workouts[activeWorkoutTab]?.items.length > 0 ? (
                  <div className="space-y-3">
                    {workouts[activeWorkoutTab].items.map((it: any, itemIdx: number) => (
                      <div
                        key={itemIdx}
                        className="bg-slate-800/60 border border-slate-700/60 rounded-xl p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-7 h-7 rounded-lg bg-emerald-500/20 text-emerald-400 flex items-center justify-center font-bold text-xs">
                            {itemIdx + 1}
                          </div>
                          <div>
                            <div className="text-xs font-bold text-white">{it.exercise.name}</div>
                            <div className="text-[10px] text-slate-400">
                              {it.exercise.muscle_group} • {it.exercise.equipment}
                            </div>
                          </div>
                        </div>

                        {/* Controles de Séries, Reps, Carga e Descanso */}
                        <div className="flex items-center gap-2 text-xs">
                          <div className="flex items-center gap-1 bg-slate-900 px-2 py-1 rounded-lg border border-slate-700">
                            <span className="text-slate-400 text-[10px]">Séries:</span>
                            <input
                              type="number"
                              value={it.sets}
                              onChange={(e) => {
                                const val = parseInt(e.target.value) || 1;
                                const updated = [...workouts];
                                updated[activeWorkoutTab].items[itemIdx].sets = val;
                                setWorkouts(updated);
                              }}
                              className="w-10 bg-transparent text-white font-bold text-center focus:outline-none"
                            />
                          </div>

                          <div className="flex items-center gap-1 bg-slate-900 px-2 py-1 rounded-lg border border-slate-700">
                            <span className="text-slate-400 text-[10px]">Reps:</span>
                            <input
                              type="text"
                              value={it.reps}
                              onChange={(e) => {
                                const updated = [...workouts];
                                updated[activeWorkoutTab].items[itemIdx].reps = e.target.value;
                                setWorkouts(updated);
                              }}
                              className="w-14 bg-transparent text-white font-bold text-center focus:outline-none"
                            />
                          </div>

                          <div className="flex items-center gap-1 bg-slate-900 px-2 py-1 rounded-lg border border-slate-700">
                            <span className="text-slate-400 text-[10px]">Carga:</span>
                            <input
                              type="number"
                              value={it.load_kg}
                              onChange={(e) => {
                                const val = parseFloat(e.target.value) || 0;
                                const updated = [...workouts];
                                updated[activeWorkoutTab].items[itemIdx].load_kg = val;
                                setWorkouts(updated);
                              }}
                              className="w-12 bg-transparent text-emerald-400 font-bold text-center focus:outline-none"
                            />
                            <span className="text-[10px] text-slate-500">kg</span>
                          </div>

                          <div className="flex items-center gap-1 bg-slate-900 px-2 py-1 rounded-lg border border-slate-700">
                            <Clock className="w-3 h-3 text-amber-400" />
                            <input
                              type="number"
                              value={it.rest_seconds}
                              onChange={(e) => {
                                const val = parseInt(e.target.value) || 30;
                                const updated = [...workouts];
                                updated[activeWorkoutTab].items[itemIdx].rest_seconds = val;
                                setWorkouts(updated);
                              }}
                              className="w-10 bg-transparent text-amber-400 font-bold text-center focus:outline-none"
                            />
                            <span className="text-[10px] text-slate-500">s</span>
                          </div>

                          <button
                            onClick={() => removeExerciseFromWorkout(itemIdx)}
                            className="p-1.5 text-slate-500 hover:text-red-400 transition-colors"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="p-8 border border-dashed border-slate-800 rounded-xl text-center text-xs text-slate-500">
                    Nenhum exercício adicionado a este treino ainda. Escolha no catálogo abaixo ou clique em um template acima.
                  </div>
                )}
              </div>

              {/* Catálogo de 110+ Exercícios */}
              <div className="pt-4 border-t border-slate-800 space-y-3">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                  <div className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
                    <Dumbbell className="w-3.5 h-3.5 text-blue-400" />
                    Ampla Base de Exercícios ({exercises.length} disponíveis)
                  </div>

                  <div className="flex items-center gap-2">
                    <div className="relative">
                      <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-2.5" />
                      <input
                        type="text"
                        placeholder="Buscar exercício..."
                        value={searchExercise}
                        onChange={(e) => setSearchExercise(e.target.value)}
                        className="pl-8 pr-3 py-1.5 bg-slate-800 border border-slate-700 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none w-48"
                      />
                    </div>

                    <select
                      value={selectedGroup}
                      onChange={(e) => setSelectedGroup(e.target.value)}
                      className="px-2.5 py-1.5 bg-slate-800 border border-slate-700 rounded-xl text-xs text-slate-300 focus:outline-none"
                    >
                      <option value="">Todos os Grupos</option>
                      {muscleGroups.map((g) => (
                        <option key={g.group} value={g.group}>
                          {g.group} ({g.count})
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Grade de Exercícios Rápidos */}
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2 max-h-56 overflow-y-auto pr-1">
                  {filteredExercises.slice(0, 30).map((ex) => (
                    <button
                      key={ex.id}
                      onClick={() => addExerciseToCurrentWorkout(ex)}
                      className="text-left p-2.5 rounded-xl bg-slate-800/40 hover:bg-slate-800 border border-slate-700/50 hover:border-emerald-500/50 transition-all flex items-center justify-between group"
                    >
                      <div className="truncate pr-2">
                        <div className="text-xs font-semibold text-white group-hover:text-emerald-400 truncate">
                          {ex.name}
                        </div>
                        <div className="text-[10px] text-slate-500">
                          {ex.muscle_group} • {ex.equipment}
                        </div>
                      </div>
                      <Plus className="w-4 h-4 text-slate-500 group-hover:text-emerald-400 shrink-0" />
                    </button>
                  ))}
                </div>
              </div>

              {/* Botão de Salvar Ficha */}
              <div className="pt-4 border-t border-slate-800 flex items-center justify-between">
                {saveSuccess ? (
                  <div className="text-xs text-emerald-400 font-semibold flex items-center gap-1.5">
                    <CheckCircle className="w-4 h-4" /> Treino salvo e ativado no app do aluno com sucesso!
                  </div>
                ) : (
                  <div className="text-xs text-slate-500">
                    O aluno receberá a atualização automaticamente no aplicativo.
                  </div>
                )}

                <button
                  onClick={handleSaveWorkoutPlan}
                  disabled={saving || workouts[activeWorkoutTab]?.items.length === 0}
                  className="px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition-colors shadow-lg shadow-emerald-600/30 flex items-center gap-2 disabled:opacity-50"
                >
                  <Save className="w-4 h-4" />
                  {saving ? "Salvando..." : "Salvar Ficha de Treino"}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function ProfessorPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-xs text-slate-400">Carregando portal do professor...</div>}>
      <ProfessorContent />
    </Suspense>
  );
}
