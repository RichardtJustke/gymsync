import { DashboardStats, Exercise, ImportPreviewResult, Invoice, ParsedStudentRow, Student, WorkoutPlan } from "@/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("gymsync_token");
}

export function setToken(token: string) {
  if (typeof window !== "undefined") {
    localStorage.setItem("gymsync_token", token);
  }
}

export function clearToken() {
  if (typeof window !== "undefined") {
    localStorage.removeItem("gymsync_token");
    localStorage.removeItem("gymsync_user");
  }
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers = new Headers(options.headers || {});

  if (!headers.has("Content-Type") && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    let errorMsg = `Erro ${res.status}`;
    try {
      const errJson = await res.json();
      errorMsg = errJson.error || errorMsg;
    } catch {
      // ignora
    }
    throw new Error(errorMsg);
  }

  return res.json();
}

export const api = {
  // Auth
  async login(identifier: string, password: string) {
    const data = await request<{ token: string; user: any; gym: any }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ identifier, password }),
    });
    setToken(data.token);
    if (typeof window !== "undefined") {
      localStorage.setItem("gymsync_user", JSON.stringify(data.user));
    }
    return data;
  },

  async registerGym(payload: any) {
    const data = await request<{ token: string; user: any; gym: any }>("/auth/register-gym", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    setToken(data.token);
    return data;
  },

  async getMe() {
    return request<{ user: any; gym: any; student: any }>("/auth/me");
  },

  // Dashboard
  async getDashboardStats() {
    return request<DashboardStats>("/dashboard/stats");
  },

  // Alunos
  async listStudents(search = "", status = "", page = 1, limit = 20) {
    const query = new URLSearchParams({
      q: search,
      status,
      page: String(page),
      limit: String(limit),
    });
    return request<{ data: Student[]; total: number; page: number; limit: number }>(`/students?${query}`);
  },

  async createStudent(payload: any) {
    return request<Student>("/students", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  async getStudent(id: string) {
    return request<{ student: Student; workout_plans: WorkoutPlan[]; invoices: Invoice[]; checkins: any[] }>(`/students/${id}`);
  },

  // Importador
  async uploadImportPreview(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    return request<ImportPreviewResult>("/students/import/preview", {
      method: "POST",
      body: formData,
    });
  },

  async executeImport(rows: ParsedStudentRow[]) {
    return request<{ total_processed: number; created_count: number; updated_count: number; skipped_count: number; errors: string[] }>("/students/import/execute", {
      method: "POST",
      body: JSON.stringify({ rows }),
    });
  },

  // Exercícios
  async listExercises(group = "") {
    const query = group ? `?group=${encodeURIComponent(group)}` : "";
    return request<{ data: Exercise[]; groups: { group: string; count: number }[]; total: number }>(`/exercises${query}`);
  },

  async createExercise(payload: any) {
    return request<Exercise>("/exercises", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  // Treinos
  async getStudentWorkouts(studentId: string) {
    return request<WorkoutPlan[]>(`/students/${studentId}/workouts`);
  },

  async createWorkoutPlan(studentId: string, payload: any) {
    return request<WorkoutPlan>(`/students/${studentId}/workouts`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  async getMyWorkouts() {
    return request<{ has_plan: boolean; student: Student; plan: WorkoutPlan | null }>("/workouts/my");
  },

  async updateExerciseLoad(itemId: string, loadKg: number) {
    return request<{ success: boolean; load_kg: number }>(`/workouts/items/${itemId}/load`, {
      method: "POST",
      body: JSON.stringify({ load_kg: loadKg }),
    });
  },

  // Faturamento
  async listInvoices(status = "") {
    const query = status ? `?status=${status}` : "";
    return request<Invoice[]>(`/invoices${query}`);
  },

  async getMyInvoices() {
    return request<Invoice[]>("/invoices/my");
  },

  async mockPay(invoiceId: string) {
    return request<{ success: boolean; message: string; invoice: Invoice }>(`/invoices/${invoiceId}/mock-pay`, {
      method: "POST",
    });
  },

  getBoletoUrl(invoiceId: string) {
    return `${API_BASE_URL}/invoices/${invoiceId}/boleto`;
  },

  // Check-in
  async doCheckIn(studentId?: string) {
    return request<{ success: boolean; message: string; check_in: any }>("/checkins", {
      method: "POST",
      body: JSON.stringify({ student_id: studentId }),
    });
  },

  async getTodayCheckIns() {
    return request<any[]>("/checkins/today");
  },
};
