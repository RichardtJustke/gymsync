export type Role = "ADMIN" | "GERENTE" | "RECEPCAO" | "CONSULTOR" | "PROFESSOR" | "ALUNO";
export type StudentStatus = "ATIVO" | "INADIMPLENTE" | "CANCELADO" | "PENDENTE";
export type InvoiceStatus = "PENDENTE" | "PAGO" | "VENCIDO" | "CANCELADO";
export type PaymentMethod = "PIX" | "BOLETO" | "CARTAO" | "DINHEIRO" | "TRANSFERENCIA";

export type Permission =
  | "dashboard.view"
  | "students.view"
  | "students.manage"
  | "checkin.manage"
  | "agenda.view"
  | "agenda.manage"
  | "workouts.manage"
  | "leads.manage"
  | "sales.manage"
  | "goals.manage"
  | "finance.view"
  | "finance.manage"
  | "catalog.manage"
  | "staff.manage"
  | "reports.view"
  | "announcements.send"
  | "audit.view"
  | "equipment.manage";

export interface Gym {
  id: string;
  name: string;
  document: string;
  phone?: string;
  email?: string;
  address?: string;
  theme_hex: string;
  booking_window_days: number;
  cancel_deadline_hours: number;
  overdue_block_days: number;
}

export interface User {
  id: string;
  gym_id: string;
  name: string;
  email: string;
  document: string;
  phone: string;
  role: Role;
  job_title?: string;
  active: boolean;
  first_access: boolean;
  created_at?: string;
  gym?: Gym;
}

export interface Plan {
  id: string;
  name: string;
  description: string;
  price: number;
  duration_months: number;
  classes_per_week: number;
  active: boolean;
}

export interface Modality {
  id: string;
  name: string;
  description: string;
  color: string;
  allow_trial: boolean;
  active: boolean;
}

export interface Student {
  id: string;
  gym_id: string;
  user_id: string;
  user: User;
  plan_id?: string | null;
  plan?: Plan | null;
  plan_name: string;
  plan_price: number;
  due_day: number;
  status: StudentStatus;
  weight?: number;
  height?: number;
  notes?: string;
  import_source?: string;
  canceled_at?: string | null;
  cancel_reason?: string;
  created_at: string;
}

export interface Exercise {
  id: string;
  gym_id?: string;
  name: string;
  muscle_group: string;
  equipment: string;
  description: string;
  video_url?: string;
}

export interface WorkoutExercise {
  id: string;
  workout_id: string;
  exercise_id: string;
  exercise?: Exercise;
  sets: number;
  reps: string;
  load_kg: number;
  rest_seconds: number;
  notes?: string;
  order_index: number;
}

export interface Workout {
  id: string;
  workout_plan_id: string;
  letter: string;
  name: string;
  description?: string;
  order_index: number;
  items: WorkoutExercise[];
}

export interface WorkoutPlan {
  id: string;
  gym_id: string;
  student_id: string;
  teacher_id: string;
  teacher?: User;
  title: string;
  goal: string;
  active: boolean;
  start_date: string;
  workouts: Workout[];
}

export interface WorkoutTemplateItem {
  id?: string;
  workout_template_div_id?: string;
  exercise_id: string;
  exercise?: Exercise;
  sets: number;
  reps: string;
  load_kg: number;
  rest_seconds: number;
  notes?: string;
  order_index?: number;
}

export interface WorkoutTemplateDiv {
  id?: string;
  workout_template_id?: string;
  letter: string;
  name: string;
  description?: string;
  order_index?: number;
  items: WorkoutTemplateItem[];
}

export interface WorkoutTemplate {
  id: string;
  gym_id?: string | null;
  created_by_id?: string | null;
  created_by?: User;
  title: string;
  goal: string;
  level: string;
  description?: string;
  workouts: WorkoutTemplateDiv[];
  created_at?: string;
}

export interface PhysicalAssessment {
  id: string;
  gym_id: string;
  student_id: string;
  student?: Student;
  evaluator_id?: string | null;
  evaluator?: User;
  date: string;
  weight: number;
  height: number;
  body_fat: number;
  muscle_mass: number;
  chest: number;
  waist: number;
  abdomen: number;
  hips: number;
  right_arm: number;
  left_arm: number;
  right_thigh: number;
  left_thigh: number;
  right_calf: number;
  left_calf: number;
  notes?: string;
  created_at?: string;
}

export interface Invoice {
  id: string;
  gym_id: string;
  student_id: string;
  student?: Student;
  kind: "MENSALIDADE" | "AVULSA";
  reference: string;
  amount: number;
  discount: number;
  paid_amount: number;
  due_date: string;
  paid_at?: string | null;
  payment_method?: PaymentMethod | "";
  status: InvoiceStatus;
  cancel_reason?: string;
  boleto_digits: string;
  pix_copy_paste: string;
  description: string;
}

export interface CheckIn {
  id: string;
  gym_id: string;
  student_id: string;
  student?: Student;
  method?: string;
  created_at: string;
}

export interface ClassSession {
  id: string;
  schedule_id?: string | null;
  modality_id: string;
  modality?: Modality;
  teacher_id?: string | null;
  teacher?: { id: string; name: string } | null;
  title: string;
  starts_at: string;
  ends_at: string;
  capacity: number;
  room: string;
  status: "AGENDADA" | "CANCELADA";
  notes: string;
  booked?: number;
  waitlist?: number;
  my_status?: string;
}

export interface ClassSchedule {
  id: string;
  modality_id: string;
  modality?: Modality;
  teacher_id?: string | null;
  teacher?: User | null;
  weekday: number;
  start_time: string;
  duration_min: number;
  capacity: number;
  room: string;
  active: boolean;
}

export type BookingStatus = "CONFIRMADA" | "ESPERA" | "CANCELADA" | "PRESENTE" | "FALTA";

export interface Booking {
  id: string;
  session_id: string;
  session?: ClassSession;
  student_id?: string | null;
  student?: Student | null;
  lead_id?: string | null;
  lead?: Lead | null;
  status: BookingStatus;
  created_at: string;
}

export type LeadStatus = "NOVO" | "AULA_AGENDADA" | "COMPARECEU" | "MATRICULADA" | "PERDIDA";

export interface Lead {
  id: string;
  name: string;
  email: string;
  phone: string;
  source: string;
  message: string;
  status: LeadStatus;
  trial_session?: ClassSession | null;
  student_id?: string | null;
  assigned_to_id?: string | null;
  assigned_to?: { id: string; name: string } | null;
  notes: string;
  created_at: string;
}

export interface TodaySession {
  id: string;
  title: string;
  color: string;
  teacher: string;
  starts_at: string;
  capacity: number;
  booked: number;
  attended: number;
  status: string;
}

export interface DashboardStats {
  total_students: number;
  active_students: number;
  inactive_students: number;
  today_checkins: number;
  new_leads: number;
  leads_last30: number;
  enrolled_last30: number;
  without_workout_count: number;
  recent_checkins: CheckIn[];
  today_sessions: TodaySession[];
  show_finance: boolean;
  monthly_revenue: number;
  expected_revenue: number;
  overdue_total: number;
  delinquent_students: number;
  pending_invoices: Invoice[];
  sales_month: number;
  uncovered_sessions: number;
  equipment_issues: number;
  staff_active: number;
}

export interface FinanceSummary {
  month: string;
  gateway: string;
  expected: number;
  received: number;
  received_count: number;
  pending: number;
  pending_count: number;
  overdue_total: number;
  overdue_count: number;
  delinquent_students: number;
  discounts: number;
  by_method: Record<string, number>;
  daily_received: { day: number; value: number }[];
}

export type PayType = "FIXO" | "HORA_AULA" | "MISTO";

export interface StaffProfile {
  id: string;
  user_id: string;
  hire_date?: string | null;
  pay_type: PayType;
  base_salary: number;
  hourly_rate: number;
  commission_percent: number;
  commission_fixed: number;
  pix_key: string;
  notes: string;
}

export interface StaffMember extends User {
  profile: StaffProfile | null;
  role_label: string;
  classes_month: number;
  sales_month: number;
  last_action?: string | null;
}

export interface PermissionInfo {
  key: Permission;
  label: string;
  group: string;
  hint: string;
}

export interface RolePermissions {
  role: Role;
  label: string;
  editable: boolean;
  custom: boolean;
  permissions: Permission[];
  members: number;
}

export interface TeacherHours {
  user_id: string;
  name: string;
  active: boolean;
  given: number;
  hours: number;
  upcoming: number;
  covered: number;
  capacity: number;
  attended: number;
  occupancy: number;
  hourly_rate: number;
  hours_amount: number;
}

export interface PayrollRow {
  user_id: string;
  name: string;
  role: Role;
  job_title: string;
  pay_type: PayType | "";
  pix_key: string;
  base_salary: number;
  hours: number;
  hourly_rate: number;
  hours_amount: number;
  commission: number;
  bonus: number;
  deductions: number;
  total: number;
  status: "ABERTO" | "PAGO";
  paid_at?: string | null;
  notes: string;
}

export type SaleKind = "MATRICULA" | "RENOVACAO" | "UPGRADE" | "AVULSA";

export interface Sale {
  id: string;
  seller_id: string;
  seller?: { id: string; name: string; role: Role } | null;
  student_id?: string | null;
  student?: Student | null;
  plan_id?: string | null;
  kind: SaleKind;
  description: string;
  amount: number;
  commission: number;
  status: "ATIVA" | "ESTORNADA";
  created_at: string;
}

export interface Goal {
  id: string;
  user_id?: string | null;
  month: string;
  target_sales: number;
  target_revenue: number;
}

export interface RankingRow {
  user_id: string;
  name: string;
  role: Role;
  sales: number;
  new_sales: number;
  renewals: number;
  revenue: number;
  commission: number;
  goal_sales: number;
  goal_revenue: number;
  progress: number;
  leads_assigned: number;
  leads_converted: number;
  conversion: number;
}

export interface Ranking {
  month: string;
  rows: RankingRow[];
  total_sales: number;
  total_amount: number;
  total_commission: number;
  gym_goal: Goal | null;
  gym_progress: number;
}

export interface AuditLog {
  id: string;
  user_id?: string | null;
  user_name: string;
  user_role: Role;
  action: string;
  entity: string;
  entity_id?: string | null;
  summary: string;
  created_at: string;
}

export interface Equipment {
  id: string;
  name: string;
  sector: string;
  status: "OPERACIONAL" | "ATENCAO" | "MANUTENCAO";
  notes: string;
  last_maintenance?: string | null;
  next_maintenance?: string | null;
}

export interface Announcement {
  id: string;
  kind: "AVISO" | "EVENTO";
  title: string;
  body: string;
  audience: string;
  sent_count: number;
  created_at: string;
}

export interface Notification {
  id: string;
  kind: string;
  title: string;
  body: string;
  link: string;
  read_at?: string | null;
  created_at: string;
}

export interface Overview {
  months: {
    month: string;
    new_students: number;
    canceled: number;
    revenue: number;
    expected: number;
    sales: number;
    sales_amount: number;
    leads: number;
    checkins: number;
  }[];
  active: number;
  churn_rate: number;
  ticket: number;
  payers: number;
  by_teacher: { name: string; sessions: number; capacity: number; attended: number; occupancy: number }[];
  by_slot: { weekday: number; hour: number; sessions: number; capacity: number; booked: number; occupancy: number }[];
  plan_mix: { plan: string; students: number; monthly: number }[];
}

export interface FrequencyReport {
  days: number;
  total: number;
  heatmap: number[][];
  daily: { date: string; count: number }[];
  attendance: { modality: string; color: string; sessions: number; booked: number; attended: number; no_show: number; capacity: number }[];
  absent: { student_id: string; name: string; phone: string; plan_name: string; last_checkin_at?: string | null; days_absent: number }[];
}

export interface ParsedStudentRow {
  row_index: number;
  name: string;
  document: string;
  email: string;
  phone: string;
  plan_name: string;
  plan_price: number;
  due_day: number;
  status: string;
  is_valid: boolean;
  error?: string;
}

export interface ImportPreviewResult {
  total_rows: number;
  valid_rows: number;
  invalid_rows: number;
  detected_fields: Record<string, string>;
  rows: ParsedStudentRow[];
}

export interface StudentDetail {
  student: Student;
  workout_plans: WorkoutPlan[];
  checkins: CheckIn[];
  bookings: Booking[];
  invoices: Invoice[];
  sales: Sale[];
  checkins_last30: number;
}
