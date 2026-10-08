package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role representa os níveis de acesso na plataforma
type Role string

const (
	RoleAdmin     Role = "ADMIN"     // Dono / gestão total (permissões fixas)
	RoleGerente   Role = "GERENTE"   // Gerente da unidade
	RoleRecepcao  Role = "RECEPCAO"  // Recepção / atendimento
	RoleConsultor Role = "CONSULTOR" // Consultor(a) de vendas
	RoleProfessor Role = "PROFESSOR" // Personal / Instrutor
	RoleAluno     Role = "ALUNO"     // Aluno / Cliente
)

// StaffRoles papéis de equipe (todos menos aluno)
var StaffRoles = []Role{RoleAdmin, RoleGerente, RoleRecepcao, RoleConsultor, RoleProfessor}

// IsStaffRole indica se o papel é de equipe
func IsStaffRole(r Role) bool {
	for _, s := range StaffRoles {
		if s == r {
			return true
		}
	}
	return false
}

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
	ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name     string    `gorm:"size:255;not null" json:"name"`
	Document string    `gorm:"size:30;not null" json:"document"` // CNPJ/CPF
	Phone    string    `gorm:"size:30" json:"phone"`
	Email    string    `gorm:"size:255" json:"email"`
	Address  string    `gorm:"size:255" json:"address"`
	LogoURL  string    `gorm:"size:500" json:"logo_url"`
	ThemeHex string    `gorm:"size:20;default:'#3b82f6'" json:"theme_hex"` // Suporte para cor personalizada
	// Regras da agenda
	BookingWindowDays   int `gorm:"default:7" json:"booking_window_days"`   // Quantos dias antes a reserva abre
	CancelDeadlineHours int `gorm:"default:2" json:"cancel_deadline_hours"` // Antecedência mínima para cancelar reserva
	// Cobrança: dias de atraso para bloquear a entrada (0 = nunca bloqueia automaticamente)
	OverdueBlockDays int            `gorm:"default:0" json:"overdue_block_days"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
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
	JobTitle     string         `gorm:"size:100" json:"job_title"`  // Cargo exibido (ex.: "Coordenadora de Pilates")
	Active       bool           `gorm:"default:true" json:"active"` // Equipe desativada não entra no sistema
	FirstAccess  bool           `gorm:"default:true" json:"first_access"`
	TokenVersion int            `gorm:"default:0" json:"-"` // Incrementado para invalidar sessões (troca de senha, exclusão)
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
	PlanID        *uuid.UUID     `gorm:"type:uuid;index" json:"plan_id"`
	Plan          *Plan          `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
	PlanName      string         `gorm:"size:100;default:'Mensal'" json:"plan_name"`
	PlanPrice     float64        `gorm:"default:0.0" json:"plan_price"`
	DueDay        int            `gorm:"default:10" json:"due_day"`
	Status        StudentStatus  `gorm:"size:30;default:'ATIVO'" json:"status"`
	Weight        float64        `json:"weight"` // Peso em kg
	Height        float64        `json:"height"` // Altura em cm
	Notes         string         `gorm:"type:text" json:"notes"`
	ImportSource  string         `gorm:"size:100" json:"import_source"` // ex: "planilha-excel"
	LeadID        *uuid.UUID     `gorm:"type:uuid;index" json:"lead_id"`
	CanceledAt    *time.Time     `gorm:"index" json:"canceled_at"`
	CancelReason  string         `gorm:"size:255" json:"cancel_reason"`
	LastDueNotice string         `gorm:"size:10" json:"-"` // Data (AAAA-MM-DD) do último lembrete de vencimento enviado
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
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID uuid.UUID      `gorm:"type:uuid;index;not null" json:"student_id"`
	Student   *Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	TeacherID uuid.UUID      `gorm:"type:uuid;index;not null" json:"teacher_id"`
	Teacher   *User          `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Title     string         `gorm:"size:255;not null" json:"title"` // Ex: "Treino ABC - Foco em Hipertrofia"
	Goal      string         `gorm:"size:100" json:"goal"`
	Active    bool           `gorm:"default:true" json:"active"`
	StartDate time.Time      `json:"start_date"`
	EndDate   *time.Time     `json:"end_date"`
	Workouts  []Workout      `gorm:"foreignKey:WorkoutPlanID" json:"workouts"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
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
	Name          string            `gorm:"size:255;not null" json:"name"`  // Ex: "Peito e Tríceps"
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

type InvoiceKind string

const (
	InvoiceMonthly InvoiceKind = "MENSALIDADE"
	InvoiceExtra   InvoiceKind = "AVULSA"
)

// PaymentMethod forma de pagamento registrada na baixa
type PaymentMethod string

const (
	PayPix      PaymentMethod = "PIX"
	PayBoleto   PaymentMethod = "BOLETO"
	PayCard     PaymentMethod = "CARTAO"
	PayCash     PaymentMethod = "DINHEIRO"
	PayTransfer PaymentMethod = "TRANSFERENCIA"
)

// Invoice cobrança de mensalidade (Boleto / PIX simulados até integrar um gateway)
type Invoice struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"student_id"`
	Student       *Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Kind          InvoiceKind    `gorm:"size:20;default:'MENSALIDADE'" json:"kind"`
	Reference     string         `gorm:"size:7;index" json:"reference"` // Competência AAAA-MM (mensalidades)
	Amount        float64        `gorm:"not null" json:"amount"`
	Discount      float64        `gorm:"default:0" json:"discount"`
	PaidAmount    float64        `gorm:"default:0" json:"paid_amount"`
	DueDate       time.Time      `gorm:"not null;index" json:"due_date"`
	PaidAt        *time.Time     `gorm:"index" json:"paid_at"`
	PaymentMethod PaymentMethod  `gorm:"size:20" json:"payment_method"`
	ReceivedByID  *uuid.UUID     `gorm:"type:uuid" json:"received_by_id"`
	Status        InvoiceStatus  `gorm:"size:30;default:'PENDENTE';index" json:"status"`
	CancelReason  string         `gorm:"size:255" json:"cancel_reason"`
	BoletoDigits  string         `gorm:"size:100" json:"boleto_digits"`   // Linha digitável simulada
	PixCopyPaste  string         `gorm:"type:text" json:"pix_copy_paste"` // Código Copia e Cola simulado
	Description   string         `gorm:"size:255" json:"description"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (inv *Invoice) BeforeCreate(tx *gorm.DB) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	return nil
}

// CheckIn registro de presença e acesso na academia
type CheckIn struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	GymID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID uuid.UUID  `gorm:"type:uuid;index;not null" json:"student_id"`
	Student   *Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	BookingID *uuid.UUID `gorm:"type:uuid;index" json:"booking_id"`
	Method    string     `gorm:"size:20;default:'QR'" json:"method"` // QR ou MANUAL
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
}

func (ci *CheckIn) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == uuid.Nil {
		ci.ID = uuid.New()
	}
	return nil
}

// WorkoutTemplate modelo de ficha reutilizável (Biblioteca de Modelos para professores e academia)
type WorkoutTemplate struct {
	ID          uuid.UUID            `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       *uuid.UUID           `gorm:"type:uuid;index" json:"gym_id"` // Nulo se for modelo global do sistema
	CreatedByID *uuid.UUID           `gorm:"type:uuid;index" json:"created_by_id"`
	CreatedBy   *User                `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Title       string               `gorm:"size:255;not null" json:"title"`
	Goal        string               `gorm:"size:100;not null" json:"goal"`       // Hipertrofia, Emagrecimento, Força, Adaptação, Funcional
	Level       string               `gorm:"size:50;default:'Intermediário'" json:"level"` // Iniciante, Intermediário, Avançado
	Description string               `gorm:"type:text" json:"description"`
	Workouts    []WorkoutTemplateDiv `gorm:"foreignKey:WorkoutTemplateID;constraint:OnDelete:CASCADE" json:"workouts"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	DeletedAt   gorm.DeletedAt       `gorm:"index" json:"-"`
}

func (wt *WorkoutTemplate) BeforeCreate(tx *gorm.DB) error {
	if wt.ID == uuid.Nil {
		wt.ID = uuid.New()
	}
	return nil
}

// WorkoutTemplateDiv divisão do treino no modelo (Ex: Treino A - Push)
type WorkoutTemplateDiv struct {
	ID                uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	WorkoutTemplateID uuid.UUID             `gorm:"type:uuid;index;not null" json:"workout_template_id"`
	Letter            string                `gorm:"size:10;not null" json:"letter"`
	Name              string                `gorm:"size:255;not null" json:"name"`
	Description       string                `gorm:"type:text" json:"description"`
	OrderIndex        int                   `gorm:"default:0" json:"order_index"`
	Items             []WorkoutTemplateItem `gorm:"foreignKey:WorkoutTemplateDivID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

func (wtd *WorkoutTemplateDiv) BeforeCreate(tx *gorm.DB) error {
	if wtd.ID == uuid.Nil {
		wtd.ID = uuid.New()
	}
	return nil
}

// WorkoutTemplateItem item do modelo com séries, repetições e descanso sugerido
type WorkoutTemplateItem struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	WorkoutTemplateDivID uuid.UUID `gorm:"type:uuid;index;not null" json:"workout_template_div_id"`
	ExerciseID           uuid.UUID `gorm:"type:uuid;index;not null" json:"exercise_id"`
	Exercise             *Exercise `gorm:"foreignKey:ExerciseID" json:"exercise,omitempty"`
	Sets                 int       `gorm:"default:4" json:"sets"`
	Reps                 string    `gorm:"size:50;default:'10-12'" json:"reps"`
	LoadKg               float64   `gorm:"default:0" json:"load_kg"`
	RestSeconds          int       `gorm:"default:60" json:"rest_seconds"`
	Notes                string    `gorm:"size:255" json:"notes"`
	OrderIndex           int       `gorm:"default:0" json:"order_index"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (wti *WorkoutTemplateItem) BeforeCreate(tx *gorm.DB) error {
	if wti.ID == uuid.Nil {
		wti.ID = uuid.New()
	}
	return nil
}

// PhysicalAssessment avaliação física e acompanhamento de medidas corporais
type PhysicalAssessment struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	StudentID   uuid.UUID      `gorm:"type:uuid;index;not null" json:"student_id"`
	Student     *Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	EvaluatorID *uuid.UUID     `gorm:"type:uuid;index" json:"evaluator_id"`
	Evaluator   *User          `gorm:"foreignKey:EvaluatorID" json:"evaluator,omitempty"`
	Date        time.Time      `gorm:"index;not null" json:"date"`
	Weight      float64        `json:"weight"`       // Peso em kg
	Height      float64        `json:"height"`       // Altura em cm
	BodyFat     float64        `json:"body_fat"`     // % Gordura
	MuscleMass  float64        `json:"muscle_mass"`  // Massa muscular (kg ou %)
	Chest       float64        `json:"chest"`        // Tórax / Peitoral (cm)
	Waist       float64        `json:"waist"`        // Cintura (cm)
	Abdomen     float64        `json:"abdomen"`      // Abdômen (cm)
	Hips        float64        `json:"hips"`         // Quadril (cm)
	RightArm    float64        `json:"right_arm"`    // Braço D (cm)
	LeftArm     float64        `json:"left_arm"`     // Braço E (cm)
	RightThigh  float64        `json:"right_thigh"`  // Coxa D (cm)
	LeftThigh   float64        `json:"left_thigh"`   // Coxa E (cm)
	RightCalf   float64        `json:"right_calf"`   // Panturrilha D (cm)
	LeftCalf    float64        `json:"left_calf"`    // Panturrilha E (cm)
	Notes       string         `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (pa *PhysicalAssessment) BeforeCreate(tx *gorm.DB) error {
	if pa.ID == uuid.Nil {
		pa.ID = uuid.New()
	}
	return nil
}

// LGPDConsent registro comprobatório de consentimento e termos de privacidade (LGPD Art. 7º e 8º)
type LGPDConsent struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GymID          uuid.UUID `gorm:"type:uuid;index;not null" json:"gym_id"`
	UserID         uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	PolicyVersion  string    `gorm:"size:30;not null" json:"policy_version"`
	AcceptedAt     time.Time `gorm:"not null" json:"accepted_at"`
	IPAddress      string    `gorm:"size:64" json:"ip_address"`
	UserAgent      string    `gorm:"size:255" json:"user_agent"`
	AnalyticsOptIn bool      `gorm:"default:true" json:"analytics_opt_in"`
	MarketingOptIn bool      `gorm:"default:false" json:"marketing_opt_in"`
	CreatedAt      time.Time `json:"created_at"`
}

func (c *LGPDConsent) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
