package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// newID garante UUID antes de inserir
func newID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

// Plan plano comercial da academia (Mensal, Trimestral...)
type Plan struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID          uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Description    string         `gorm:"type:text" json:"description"`
	Price          float64        `gorm:"not null;default:0" json:"price"`
	DurationMonths int            `gorm:"default:1" json:"duration_months"`
	ClassesPerWeek int            `gorm:"default:0" json:"classes_per_week"` // 0 = ilimitado
	Active         bool           `gorm:"default:true" json:"active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Plan) BeforeCreate(tx *gorm.DB) error { newID(&p.ID); return nil }

// Modality modalidade de aula (Pilates, Funcional, Dança...)
type Modality struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Color       string         `gorm:"size:20;default:'#D7FF3A'" json:"color"`
	AllowTrial  bool           `gorm:"default:true" json:"allow_trial"` // Aparece no site para aula experimental
	Active      bool           `gorm:"default:true" json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *Modality) BeforeCreate(tx *gorm.DB) error { newID(&m.ID); return nil }

// ClassSchedule horário recorrente semanal de uma turma
type ClassSchedule struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	ModalityID  uuid.UUID      `gorm:"type:uuid;index;not null" json:"modality_id"`
	Modality    *Modality      `gorm:"foreignKey:ModalityID" json:"modality,omitempty"`
	TeacherID   *uuid.UUID     `gorm:"type:uuid;index" json:"teacher_id"`
	Teacher     *User          `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Weekday     int            `gorm:"not null" json:"weekday"`           // 0 = domingo ... 6 = sábado
	StartTime   string         `gorm:"size:5;not null" json:"start_time"` // "HH:MM" no fuso da academia
	DurationMin int            `gorm:"default:60" json:"duration_min"`
	Capacity    int            `gorm:"default:15" json:"capacity"`
	Room        string         `gorm:"size:100" json:"room"`
	Active      bool           `gorm:"default:true" json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *ClassSchedule) BeforeCreate(tx *gorm.DB) error { newID(&c.ID); return nil }

type SessionStatus string

const (
	SessionScheduled SessionStatus = "AGENDADA"
	SessionCanceled  SessionStatus = "CANCELADA"
)

// ClassSession ocorrência concreta de uma aula (gerada a partir do horário recorrente ou avulsa)
type ClassSession struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	ScheduleID *uuid.UUID     `gorm:"type:uuid;uniqueIndex:idx_session_schedule_start" json:"schedule_id"`
	ModalityID uuid.UUID      `gorm:"type:uuid;index;not null" json:"modality_id"`
	Modality   *Modality      `gorm:"foreignKey:ModalityID" json:"modality,omitempty"`
	TeacherID  *uuid.UUID     `gorm:"type:uuid;index" json:"teacher_id"`
	Teacher    *User          `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
	Title      string         `gorm:"size:255" json:"title"` // Opcional: evento/aulão
	StartsAt   time.Time      `gorm:"index;uniqueIndex:idx_session_schedule_start;not null" json:"starts_at"`
	EndsAt     time.Time      `gorm:"not null" json:"ends_at"`
	Capacity   int            `gorm:"default:15" json:"capacity"`
	Room       string         `gorm:"size:100" json:"room"`
	Status     SessionStatus  `gorm:"size:20;default:'AGENDADA'" json:"status"`
	Notes      string         `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *ClassSession) BeforeCreate(tx *gorm.DB) error { newID(&c.ID); return nil }

type BookingStatus string

const (
	BookingConfirmed BookingStatus = "CONFIRMADA"
	BookingWaitlist  BookingStatus = "ESPERA"
	BookingCanceled  BookingStatus = "CANCELADA"
	BookingAttended  BookingStatus = "PRESENTE"
	BookingNoShow    BookingStatus = "FALTA"
)

// Booking reserva de vaga numa aula (aluna ou lead em aula experimental)
type Booking struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	GymID          uuid.UUID     `gorm:"type:uuid;index;not null" json:"gym_id"`
	SessionID      uuid.UUID     `gorm:"type:uuid;index;not null" json:"session_id"`
	Session        *ClassSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	StudentID      *uuid.UUID    `gorm:"type:uuid;index" json:"student_id"`
	Student        *Student      `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	LeadID         *uuid.UUID    `gorm:"type:uuid;index" json:"lead_id"`
	Lead           *Lead         `gorm:"foreignKey:LeadID" json:"lead,omitempty"`
	Status         BookingStatus `gorm:"size:20;index;not null" json:"status"`
	ReminderSentAt *time.Time    `json:"-"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (b *Booking) BeforeCreate(tx *gorm.DB) error { newID(&b.ID); return nil }

type LeadStatus string

const (
	LeadNew       LeadStatus = "NOVO"
	LeadScheduled LeadStatus = "AULA_AGENDADA"
	LeadAttended  LeadStatus = "COMPARECEU"
	LeadEnrolled  LeadStatus = "MATRICULADA"
	LeadLost      LeadStatus = "PERDIDA"
)

// Lead interessada vinda do site (aula experimental) ou cadastrada na recepção
type Lead struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GymID          uuid.UUID      `gorm:"type:uuid;index;not null" json:"gym_id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Email          string         `gorm:"size:255" json:"email"`
	Phone          string         `gorm:"size:30" json:"phone"`
	Source         string         `gorm:"size:50;default:'site'" json:"source"`
	Message        string         `gorm:"type:text" json:"message"`
	Status         LeadStatus     `gorm:"size:30;index;default:'NOVO'" json:"status"`
	TrialSessionID *uuid.UUID     `gorm:"type:uuid" json:"trial_session_id"`
	TrialSession   *ClassSession  `gorm:"foreignKey:TrialSessionID" json:"trial_session,omitempty"`
	StudentID      *uuid.UUID     `gorm:"type:uuid" json:"student_id"`
	AssignedToID   *uuid.UUID     `gorm:"type:uuid;index" json:"assigned_to_id"` // Consultor(a) responsável
	AssignedTo     *User          `gorm:"foreignKey:AssignedToID" json:"assigned_to,omitempty"`
	Notes          string         `gorm:"type:text" json:"notes"`
	ConsentAt      *time.Time     `json:"consent_at"` // Aceite LGPD no formulário do site
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (l *Lead) BeforeCreate(tx *gorm.DB) error { newID(&l.ID); return nil }

type NotificationKind string

const (
	NotifyClass    NotificationKind = "AULA"
	NotifyWorkout  NotificationKind = "FICHA"
	NotifyReminder NotificationKind = "LEMBRETE"
	NotifyDue      NotificationKind = "VENCIMENTO"
	NotifyNotice   NotificationKind = "AVISO"
	NotifyEvent    NotificationKind = "EVENTO"
	NotifyLead     NotificationKind = "LEAD"
)

// Notification item da caixa de entrada de um usuário (também enviado por push)
type Notification struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	GymID     uuid.UUID        `gorm:"type:uuid;index;not null" json:"gym_id"`
	UserID    uuid.UUID        `gorm:"type:uuid;index;not null" json:"user_id"`
	Kind      NotificationKind `gorm:"size:20;not null" json:"kind"`
	Title     string           `gorm:"size:255;not null" json:"title"`
	Body      string           `gorm:"type:text" json:"body"`
	Link      string           `gorm:"size:255" json:"link"` // Rota interna do app, ex: "/aluno?tab=agenda"
	ReadAt    *time.Time       `json:"read_at"`
	CreatedAt time.Time        `gorm:"index" json:"created_at"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error { newID(&n.ID); return nil }

// Announcement aviso/evento disparado pela gestão (histórico do que foi enviado)
type Announcement struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	GymID     uuid.UUID        `gorm:"type:uuid;index;not null" json:"gym_id"`
	Kind      NotificationKind `gorm:"size:20;not null" json:"kind"`
	Title     string           `gorm:"size:255;not null" json:"title"`
	Body      string           `gorm:"type:text" json:"body"`
	Audience  string           `gorm:"size:100;default:'ALL'" json:"audience"` // ALL ou MODALITY:<id>
	SentCount int              `json:"sent_count"`
	CreatedBy uuid.UUID        `gorm:"type:uuid" json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
}

func (a *Announcement) BeforeCreate(tx *gorm.DB) error { newID(&a.ID); return nil }

// DeviceToken token de push (FCM) de um aparelho
type DeviceToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Token     string    `gorm:"size:500;uniqueIndex;not null" json:"token"`
	Platform  string    `gorm:"size:20" json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *DeviceToken) BeforeCreate(tx *gorm.DB) error { newID(&d.ID); return nil }
