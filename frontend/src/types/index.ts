export type Role = 'ADMIN' | 'PROFESSOR' | 'ALUNO';
export type StudentStatus = 'ATIVO' | 'INADIMPLENTE' | 'CANCELADO' | 'PENDENTE';
export type InvoiceStatus = 'PENDENTE' | 'PAGO' | 'VENCIDO' | 'CANCELADO';

export interface Gym {
  id: string;
  name: string;
  document: string;
  phone?: string;
  email?: string;
  address?: string;
  theme_hex: string;
}

export interface User {
  id: string;
  gym_id: string;
  name: string;
  email: string;
  document: string;
  phone: string;
  role: Role;
  first_access: boolean;
  gym?: Gym;
}

export interface Student {
  id: string;
  gym_id: string;
  user_id: string;
  user: User;
  plan_name: string;
  plan_price: number;
  due_day: number;
  status: StudentStatus;
  weight?: number;
  height?: number;
  notes?: string;
  import_source?: string;
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

export interface Invoice {
  id: string;
  gym_id: string;
  student_id: string;
  student?: Student;
  amount: number;
  due_date: string;
  paid_at?: string;
  status: InvoiceStatus;
  boleto_digits: string;
  pix_copy_paste: string;
  description: string;
}

export interface CheckIn {
  id: string;
  gym_id: string;
  student_id: string;
  student?: Student;
  created_at: string;
}

export interface DashboardStats {
  total_students: number;
  active_students: number;
  delinquent_students: number;
  monthly_revenue: number;
  today_checkins: number;
  without_workout_count: number;
  recent_checkins: CheckIn[];
  pending_invoices: Invoice[];
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
