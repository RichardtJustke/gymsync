package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role representa os níveis de acesso na plataforma
type Role string

const (
	RoleAdmin     Role = "ADMIN"     // Gestor da Academia
	RoleProfessor Role = "PROFESSOR" // Personal / Instrutor
	RoleAluno     Role = "ALUNO"     // Aluno / Cliente
)

// StudentStatus status da matrícula do aluno
type StudentStatus string

const (
	StatusAtivo        StudentStatus = "ATIVO"
	StatusInadimplente StudentStatus = "INADIMPLENTE"
	StatusCancelado    StudentStatus = "CANCELADO"
	StatusPendente     StudentStatus = "PENDENTE"
)

// InvoiceStatus status de fatura/mensalidade
type InvoiceStatus string

const (
	InvoicePendente InvoiceStatus = "PENDENTE"
	InvoicePago     InvoiceStatus = "PAGO"
	InvoiceVencido  InvoiceStatus = "VENCIDO"
	InvoiceCancel   InvoiceStatus = "CANCELADO"
)

// Gym representa a academia (Tenant isolado)
type Gym struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	Document  string         `gorm:"size:30;not null" json:"document"` // CNPJ/CPF
	Phone     string         `gorm:"size:30" json:"phone"`
	Email     string         `gorm:"size:255" json:"email"`
	Address   string         `gorm:"size:255" json:"address"`
	LogoURL   string         `gorm:"size:500" json:"logo_url"`
	ThemeHex  string         `gorm:"size:20;default:'#3b82f6'" json:"theme_hex"` // Suporte para cor personalizada
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (g *Gym) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

// User usuário autenticável do sistema
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	Gym          *Gym           `gorm:"foreignKey:GymID" json:"gym,omitempty"`
	Name         string         `gorm:"size:255;not null" json:"name"`
	Email        string         `gorm:"size:255;index" json:"email"`
	Document     string         `gorm:"size:30;index" json:"document"` // CPF
	Phone        string         `gorm:"size:30" json:"phone"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Role         Role           `gorm:"size:30;not null;default:'ALUNO'" json:"role"`
	FirstAccess  bool           `gorm:"default:true" json:"first_access"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// Student perfil detalhado do aluno
type Student struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	UserID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`
	User          User           `gorm:"foreignKey:UserID" json:"user"`
	PlanName      string         `gorm:"size:100;default:'Mensal'" json:"plan_name"`
	PlanPrice     float64        `gorm:"default:0.0" json:"plan_price"`
	DueDay        int            `gorm:"default:10" json:"due_day"`
	Status        StudentStatus  `gorm:"size:30;default:'ATIVO'" json:"status"`
	Weight        float64        `json:"weight"` // Peso em kg
	Height        float64        `json:"height"` // Altura em cm
	Notes         string         `gorm:"type:text" json:"notes"`
	ImportSource  string         `gorm:"size:100" json:"import_source"` // ex: "planilha-excel"
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (s *Student) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// Exercise exercício de musculação ou funcional (ampla base pré-cadastrada)
type Exercise struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       *uuid.UUID     `gorm:"type:uuid;index" json:"gym_id"` // Nulo se for exercício global padrão
	Name        string         `gorm:"size:255;not null" json:"name"`
	MuscleGroup string         `gorm:"size:100;not null;index" json:"muscle_group"` // Peito, Costas, etc.
	Equipment   string         `gorm:"size:100" json:"equipment"`                   // Barra, Halteres, Máquina, Cabo, Peso Corporal
	Description string         `gorm:"type:text" json:"description"`
	VideoURL    string         `gorm:"size:500" json:"video_url"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (e *Exercise) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// WorkoutPlan conjunto de fichas prescrito para um aluno (Ex: Ficha de Hipertrofia 2026)
type WorkoutPlan struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID   uuid.UUID      `gorm:"type:uuid;index;not null" json:"student_id"`
	Student     *Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	TeacherID   uuid.UUID      `gorm:"type:uuid;index;not null" json:"teacher_id"`
	Teacher     *User          `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Title       string         `gorm:"size:255;not null" json:"title"` // Ex: "Treino ABC - Foco em Hipertrofia"
	Goal        string         `gorm:"size:100" json:"goal"`
	Active      bool           `gorm:"default:true" json:"active"`
	StartDate   time.Time      `json:"start_date"`
	EndDate     *time.Time     `json:"end_date"`
	Workouts    []Workout      `gorm:"foreignKey:WorkoutPlanID" json:"workouts"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (wp *WorkoutPlan) BeforeCreate(tx *gorm.DB) error {
	if wp.ID == uuid.Nil {
		wp.ID = uuid.New()
	}
	return nil
}

// Workout ficha individual (Ex: Treino A - Peito e Tríceps)
type Workout struct {
	ID            uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	WorkoutPlanID uuid.UUID         `gorm:"type:uuid;index;not null" json:"workout_plan_id"`
	Letter        string            `gorm:"size:10;not null" json:"letter"` // A, B, C, D
	Name          string            `gorm:"size:255;not null" json:"name"`   // Ex: "Peito e Tríceps"
	Description   string            `gorm:"type:text" json:"description"`
	OrderIndex    int               `gorm:"default:0" json:"order_index"`
	Items         []WorkoutExercise `gorm:"foreignKey:WorkoutID" json:"items"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"-"`
}

func (w *Workout) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// WorkoutExercise item da ficha com séries, repetições e carga
type WorkoutExercise struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	WorkoutID   uuid.UUID `gorm:"type:uuid;index;not null" json:"workout_id"`
	ExerciseID  uuid.UUID `gorm:"type:uuid;index;not null" json:"exercise_id"`
	Exercise    *Exercise `gorm:"foreignKey:ExerciseID" json:"exercise,omitempty"`
	Sets        int       `gorm:"default:4" json:"sets"`
	Reps        string    `gorm:"size:50;default:'10-12'" json:"reps"`
	LoadKg      float64   `gorm:"default:0" json:"load_kg"`
	RestSeconds int       `gorm:"default:60" json:"rest_seconds"`
	Notes       string    `gorm:"size:255" json:"notes"`
	OrderIndex  int       `gorm:"default:0" json:"order_index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (we *WorkoutExercise) BeforeCreate(tx *gorm.DB) error {
	if we.ID == uuid.Nil {
		we.ID = uuid.New()
	}
	return nil
}

// Invoice cobrança mockada de mensalidade (Boleto / PIX)
type Invoice struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID    uuid.UUID      `gorm:"type:uuid;index;not null" json:"student_id"`
	Student      *Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Amount       float64        `gorm:"not null" json:"amount"`
	DueDate      time.Time      `gorm:"not null" json:"due_date"`
	PaidAt       *time.Time     `json:"paid_at"`
	Status       InvoiceStatus  `gorm:"size:30;default:'PENDENTE'" json:"status"`
	BoletoDigits string         `gorm:"size:100" json:"boleto_digits"` // Linha digitável simulada
	PixCopyPaste string         `gorm:"type:text" json:"pix_copy_paste"` // Código Copia e Cola simulado
	Description  string         `gorm:"size:255" json:"description"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (inv *Invoice) BeforeCreate(tx *gorm.DB) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	return nil
}

// CheckIn registro de presença e acesso na academia
type CheckIn struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GymID     uuid.UUID `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID uuid.UUID `gorm:"type:uuid;index;not null" json:"student_id"`
	Student   *Student  `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (ci *CheckIn) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == uuid.Nil {
		ci.ID = uuid.New()
	}
	return nil
}
