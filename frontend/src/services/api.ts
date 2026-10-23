import type {
  Announcement, AuditLog, Booking, BookingStatus, CheckIn, ClassSchedule, ClassSession, DashboardStats, Equipment, Exercise,
  FinanceSummary, FrequencyReport, Gym, Goal, ImportPreviewResult, Invoice, Lead, LeadStatus, Modality, Notification, Overview,
  ParsedStudentRow, PayrollRow, Permission, PermissionInfo, Plan, Ranking, Role, RolePermissions, Sale, SaleKind, StaffMember,
  Student, StudentDetail, TeacherHours, User, WorkoutPlan, PaymentMethod,
  WorkoutTemplate, PhysicalAssessment,
} from "@/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

const TOKEN_KEY = "gymsync_token";
const USER_KEY = "gymsync_user";
const PERMS_KEY = "gymsync_perms";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string) {
  if (typeof window !== "undefined") localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  if (typeof window !== "undefined") {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(PERMS_KEY);
  }
}

export function saveSession(user: User, permissions: Permission[]) {
  if (typeof window === "undefined") return;
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  localStorage.setItem(PERMS_KEY, JSON.stringify(permissions || []));
  window.dispatchEvent(new Event("gymsync:session"));
}

export function loadSession(): { user: User | null; permissions: Permission[] } {
  if (typeof window === "undefined") return { user: null, permissions: [] };
  try {
    return {
      user: JSON.parse(localStorage.getItem(USER_KEY) || "null"),
      permissions: JSON.parse(localStorage.getItem(PERMS_KEY) || "[]"),
    };
  } catch {
    return { user: null, permissions: [] };
  }
}

/** Página inicial de cada perfil depois do login */
export function homeFor(role?: Role) {
  if (role === "ALUNO") return "/aluno";
  if (role === "PROFESSOR") return "/professor";
  return "/admin";
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers = new Headers(options.headers || {});
  if (!headers.has("Content-Type") && options.body && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`${API_BASE_URL}${endpoint}`, { ...options, headers });

  if (!res.ok) {
    let msg = `Erro ${res.status}`;
    try {
      const j = await res.json();
      msg = j.error || msg;
    } catch {
      /* corpo vazio */
    }
    // Sessão encerrada (desativado, senha trocada): volta para o login
    if (res.status === 401 && typeof window !== "undefined" && endpoint !== "/auth/login") {
      clearToken();
      window.location.href = "/";
    }
    throw new ApiError(msg, res.status);
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : null) as T;
}

const get = <T,>(url: string) => request<T>(url);
const send = <T,>(method: string, url: string, body?: unknown) =>
  request<T>(url, { method, body: body === undefined ? undefined : JSON.stringify(body) });

function qs(params: Record<string, string | number | undefined | null>) {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") q.set(k, String(v));
  });
  const s = q.toString();
  return s ? `?${s}` : "";
}

export type TempCredentials = { login: string; temp_password: string };

export const api = {
  registerDevice: (token: string, platform: string) => send<{ success: boolean }>("POST", "/devices", { token, platform }),
  /* ------------------------------ Público ------------------------------ */
  publicGym: () => get<{ name: string; phone: string; address: string; email: string }>("/public/gym"),
  /* ------------------------------ Conta ------------------------------ */
  async login(identifier: string, password: string) {
    const data = await send<{ token: string; user: User; gym: Gym; permissions: Permission[] }>("POST", "/auth/login", { identifier, password });
    setToken(data.token);
    saveSession(data.user, data.permissions);
    return data;
  },
  async getMe() {
    const data = await get<{ user: User; gym: Gym; student: Student | null; permissions: Permission[] }>("/auth/me");
    saveSession(data.user, data.permissions);
    return data;
  },
  async changePassword(current_password: string, new_password: string) {
    const data = await send<{ token: string; user: User }>("POST", "/auth/password", { current_password, new_password });
    setToken(data.token);
    return data;
  },

  deleteMyAccount: (password: string) => send("DELETE", "/auth/me", { password }),

  /* ---------------------- LGPD & Privacidade ---------------------- */
  exportMyData: async () => {
    const res = await fetch(`${API_BASE_URL}/me/export-data`, {
      headers: { Authorization: `Bearer ${getToken()}` },
    });
    if (!res.ok) throw new Error("Falha ao exportar dados da conta");
    return res.json();
  },
  getLGPDConsent: () => get<{ accepted: boolean; policy_version: string; accepted_at?: string }>("/me/lgpd-consent"),
  registerLGPDConsent: (policy_version = "v1.0-2026", analytics_opt_in = true, marketing_opt_in = false) =>
    send("POST", "/me/lgpd-consent", { policy_version, analytics_opt_in, marketing_opt_in }),
  anonymizeMyData: (password: string) => send("POST", "/me/lgpd-anonymize", { password }),

  /* --------------------------- Notificações --------------------------- */
  myNotifications: () => get<{ data: Notification[]; unread: number }>("/notifications/my"),
  markNotificationsRead: (ids: string[] = []) => send("POST", "/notifications/read", { ids }),

  /* ----------------------------- Painel ----------------------------- */
  getDashboardStats: () => get<DashboardStats>("/dashboard/stats"),
  getOverview: (months = 6) => get<Overview>(`/reports/overview${qs({ months })}`),
  getFrequency: (days = 30) => get<FrequencyReport>(`/reports/frequency${qs({ days })}`),

  /* ----------------------------- Alunos ----------------------------- */
  listStudents(search = "", status = "", page = 1, limit = 20) {
    return get<{ data: Student[]; total: number; page: number; limit: number }>(`/students${qs({ q: search, status, page, limit })}`);
  },
  createStudent: (payload: Record<string, unknown>) => send<{ student: Student; temp_password: string }>("POST", "/students", payload),
  getStudent: (id: string) => get<StudentDetail>(`/students/${id}`),
  updateStudent: (id: string, payload: Record<string, unknown>) => send<Student>("PUT", `/students/${id}`, payload),
  resetStudentPassword: (id: string) => send<TempCredentials>("POST", `/students/${id}/reset-password`),
  updateMyProfile: (payload: Record<string, unknown>) => send<Student>("PUT", "/me/profile", payload),
  uploadImportPreview(file: File) {
    const fd = new FormData();
    fd.append("file", file);
    return request<ImportPreviewResult>("/students/import/preview", { method: "POST", body: fd });
  },
  executeImport: (rows: ParsedStudentRow[]) =>
    send<{ total_processed: number; created_count: number; updated_count: number; skipped_count: number; errors: string[]; credentials?: { name: string; login: string; temp_password: string }[] }>(
      "POST", "/students/import/execute", { rows }),

  /* ----------------------------- Treinos ----------------------------- */
  listExercises(group = "") {
    return get<{ data: Exercise[]; groups: { group: string; count: number }[]; total: number }>(`/exercises${qs({ group })}`);
  },
  createExercise: (payload: Record<string, unknown>) => send<Exercise>("POST", "/exercises", payload),
  getStudentWorkouts: (studentId: string) => get<WorkoutPlan[]>(`/students/${studentId}/workouts`),
  createWorkoutPlan: (studentId: string, payload: Record<string, unknown>) => send<WorkoutPlan>("POST", `/students/${studentId}/workouts`, payload),
  getMyWorkouts: () => get<{ has_plan: boolean; student: Student; plan: WorkoutPlan | null }>("/workouts/my"),
  updateExerciseLoad: (itemId: string, loadKg: number) =>
    send<{ success: boolean; load_kg: number }>("POST", `/workouts/items/${itemId}/load`, { load_kg: loadKg }),

  /* ---------------------------- Recepção ---------------------------- */
  getQRCode: () => get<{ code: string; qr_payload: string; expires_in: number }>("/checkins/qr"),
  getTodayCheckIns: () => get<CheckIn[]>("/checkins/today"),
  manualCheckIn: (student_id: string) => send<{ success: boolean; message: string; already?: boolean }>("POST", "/checkins/manual", { student_id }),
  doCheckIn: (code: string) => send<{ success: boolean; message: string; already?: boolean }>("POST", "/checkins", { code }),
  myCheckIns: (month = "") =>
    get<{ month: string; days: number[]; month_count: number; last30_count: number; week_count: number; streak_weeks: number }>(`/checkins/my${qs({ month })}`),

  /* ------------------------------ Agenda ------------------------------ */
  listSessions: (params: { from?: string; days?: number; teacher_id?: string; modality_id?: string } = {}) =>
    get<ClassSession[]>(`/sessions${qs(params)}`),
  sessionAttendees: (id: string) => get<Booking[]>(`/sessions/${id}/attendees`),
  markAttendance: (bookingId: string, status: Extract<BookingStatus, "PRESENTE" | "FALTA">) => send("PATCH", `/bookings/${bookingId}`, { status }),
  createSession: (payload: Record<string, unknown>) => send<ClassSession>("POST", "/sessions", payload),
  updateSession: (id: string, payload: { teacher_id?: string | null; capacity?: number; room?: string; notes?: string }) =>
    send<ClassSession>("PUT", `/sessions/${id}`, payload),
  cancelSession: (id: string, reason: string) => send<{ success: boolean; notified: number }>("POST", `/sessions/${id}/cancel`, { reason }),
  listSchedules: () => get<ClassSchedule[]>("/schedules"),
  createSchedule: (payload: Record<string, unknown>) => send<ClassSchedule>("POST", "/schedules", payload),
  updateSchedule: (id: string, payload: Record<string, unknown>) => send<ClassSchedule>("PUT", `/schedules/${id}`, payload),
  deleteSchedule: (id: string) => send("DELETE", `/schedules/${id}`),
  myBookings: () => get<Booking[]>("/bookings/my"),
  book: (sessionId: string) => send<{ booking: Booking; message: string }>("POST", `/sessions/${sessionId}/book`),
  cancelBooking: (sessionId: string) => send<{ message: string }>("DELETE", `/sessions/${sessionId}/book`),

  /* ------------------------------ Leads ------------------------------ */
  listLeads: (params: { status?: string; assigned_to?: string } = {}) =>
    get<{ data: Lead[]; counts: Partial<Record<LeadStatus, number>> }>(`/leads${qs(params)}`),
  createLead: (payload: Record<string, unknown>) => send<Lead>("POST", "/leads", payload),
  updateLead: (id: string, payload: Record<string, unknown>) => send<Lead>("PUT", `/leads/${id}`, payload),
  convertLead: (id: string, payload: Record<string, unknown>) => send<{ student: Student } & TempCredentials>("POST", `/leads/${id}/convert`, payload),

  /* ------------------------- Vendas e metas ------------------------- */
  listSales: (month = "", seller_id = "") => get<Sale[]>(`/sales${qs({ month, seller_id })}`),
  createSale: (payload: { seller_id?: string; student_id?: string; plan_id?: string; kind: SaleKind; amount?: number; description?: string; apply_plan?: boolean }) =>
    send<Sale>("POST", "/sales", payload),
  reverseSale: (id: string, reason: string) => send("POST", `/sales/${id}/reverse`, { reason }),
  ranking: (month = "") => get<Ranking>(`/sales/ranking${qs({ month })}`),
  setGoal: (payload: { user_id?: string | null; month: string; target_sales: number; target_revenue: number }) => send<Goal>("PUT", "/goals", payload),

  /* ---------------------------- Financeiro ---------------------------- */
  listInvoices: (params: { status?: string; month?: string; q?: string; student_id?: string } = {}) => get<Invoice[]>(`/invoices${qs(params)}`),
  financeSummary: (month = "") => get<FinanceSummary>(`/finance/summary${qs({ month })}`),
  createInvoice: (payload: { student_id: string; amount: number; due_date: string; description: string }) => send<Invoice>("POST", "/invoices", payload),
  payInvoice: (id: string, payload: { method: PaymentMethod; amount?: number; discount?: number }) => send<Invoice>("POST", `/invoices/${id}/pay`, payload),
  cancelInvoice: (id: string, reason: string) => send<Invoice>("POST", `/invoices/${id}/cancel`, { reason }),
  generateInvoices: (month = "") => send<{ created: number }>("POST", "/invoices/generate", { month }),
  getMyInvoices: () => get<Invoice[]>("/invoices/my"),
  mockPay: (id: string) => send<{ success: boolean; message: string; invoice: Invoice }>("POST", `/invoices/${id}/mock-pay`),
  /** HTML do boleto (precisa do token, então vem por fetch e é exibido num iframe isolado) */
  async getBoletoHtml(id: string) {
    const res = await fetch(`${API_BASE_URL}/invoices/${id}/boleto`, { headers: { Authorization: `Bearer ${getToken() || ""}` } });
    if (!res.ok) throw new ApiError("Não foi possível abrir o boleto", res.status);
    return res.text();
  },

  /* ------------------------------ Equipe ------------------------------ */
  listStaff: (inactive = false) => get<StaffMember[]>(`/staff${inactive ? "?inactive=1" : ""}`),
  createStaff: (payload: Record<string, unknown>) => send<{ user: User } & TempCredentials>("POST", "/staff", payload),
  updateStaff: (id: string, payload: Record<string, unknown>) => send<User>("PUT", `/staff/${id}`, payload),
  setStaffActive: (id: string, active: boolean) =>
    send<{ success: boolean; uncovered_classes?: number }>("POST", `/staff/${id}/${active ? "activate" : "deactivate"}`),
  resetStaffPassword: (id: string) => send<TempCredentials>("POST", `/staff/${id}/reset-password`),
  getPermissions: () => get<{ catalog: PermissionInfo[]; roles: RolePermissions[] }>("/permissions"),
  setPermissions: (role: Role, permissions: Permission[]) => send("PUT", `/permissions/${role}`, { permissions }),
  resetPermissions: (role: Role) => send("PUT", `/permissions/${role}`, { reset: true }),
  teacherHours: (month = "") => get<{ month: string; rows: TeacherHours[] }>(`/staff/hours${qs({ month })}`),
  payroll: (month = "") => get<{ month: string; rows: PayrollRow[]; total: number; paid: number }>(`/payroll${qs({ month })}`),
  savePayroll: (userId: string, payload: { month: string; bonus: number; deductions: number; notes?: string; mark_paid?: boolean; reopen?: boolean }) =>
    send<PayrollRow>("PUT", `/staff/${userId}/payroll`, payload),
  listTeachers: () => get<User[]>("/teachers"),
  audit: (params: { entity?: string; user_id?: string; q?: string; from?: string; to?: string; page?: number } = {}) =>
    get<{ data: AuditLog[]; total: number; page: number; limit: number }>(`/audit${qs(params)}`),

  /* --------------------------- Configurações --------------------------- */
  listPlans: (all = false) => get<{ plan: Plan; students: number }[]>(`/plans${all ? "?all=1" : ""}`),
  createPlan: (payload: Record<string, unknown>) => send<Plan>("POST", "/plans", payload),
  updatePlan: (id: string, payload: Record<string, unknown>) => send<Plan>("PUT", `/plans/${id}`, payload),
  listModalities: (all = false) => get<Modality[]>(`/modalities${all ? "?all=1" : ""}`),
  createModality: (payload: Record<string, unknown>) => send<Modality>("POST", "/modalities", payload),
  updateModality: (id: string, payload: Record<string, unknown>) => send<Modality>("PUT", `/modalities/${id}`, payload),
  getGym: () => get<Gym>("/gym"),
  updateGym: (payload: Record<string, unknown>) => send<Gym>("PUT", "/gym", payload),
  listAnnouncements: () => get<Announcement[]>("/announcements"),
  createAnnouncement: (payload: { kind: string; title: string; body: string; modality_id?: string }) => send<Announcement>("POST", "/announcements", payload),
  listEquipment: () => get<Equipment[]>("/equipment"),
  createEquipment: (payload: Record<string, unknown>) => send<Equipment>("POST", "/equipment", payload),
  updateEquipment: (id: string, payload: Record<string, unknown>) => send<Equipment>("PUT", `/equipment/${id}`, payload),
  deleteEquipment: (id: string) => send("DELETE", `/equipment/${id}`),

  /* ---------------------- Biblioteca de Modelos ---------------------- */
  listWorkoutTemplates: (params: { goal?: string; level?: string; q?: string } = {}) =>
    get<WorkoutTemplate[]>(`/workout-templates${qs(params)}`),
  getWorkoutTemplate: (id: string) => get<WorkoutTemplate>(`/workout-templates/${id}`),
  createWorkoutTemplate: (payload: Record<string, unknown>) => send<WorkoutTemplate>("POST", "/workout-templates", payload),
  deleteWorkoutTemplate: (id: string) => send<{ message: string }>("DELETE", `/workout-templates/${id}`),

  /* ------------------------- Avaliação Física ------------------------- */
  listStudentAssessments: (studentId: string) => get<PhysicalAssessment[]>(`/students/${studentId}/assessments`),
  createStudentAssessment: (studentId: string, payload: Record<string, unknown>) =>
    send<PhysicalAssessment>("POST", `/students/${studentId}/assessments`, payload),
  deleteStudentAssessment: (studentId: string, assessmentId: string) =>
    send<{ message: string }>("DELETE", `/students/${studentId}/assessments/${assessmentId}`),
  getMyAssessments: () => get<{ student: Student; assessments: PhysicalAssessment[] }>("/assessments/my"),
  createMyAssessment: (payload: Record<string, unknown>) => send<PhysicalAssessment>("POST", "/assessments/my", payload),
};
