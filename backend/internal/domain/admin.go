package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* --------------------------- Permissões --------------------------- */

// Permission chave de acesso a uma área da gestão
type Permission string

const (
	PermDashboard     Permission = "dashboard.view"
	PermStudentsView  Permission = "students.view"
	PermStudentsEdit  Permission = "students.manage"
	PermCheckin       Permission = "checkin.manage"
	PermAgendaView    Permission = "agenda.view"
	PermAgendaManage  Permission = "agenda.manage"
	PermWorkouts      Permission = "workouts.manage"
	PermLeads         Permission = "leads.manage"
	PermSales         Permission = "sales.manage"
	PermGoals         Permission = "goals.manage"
	PermFinanceView   Permission = "finance.view"
	PermFinanceManage Permission = "finance.manage"
	PermCatalog       Permission = "catalog.manage"
	PermStaff         Permission = "staff.manage"
	PermReports       Permission = "reports.view"
	PermAnnounce      Permission = "announcements.send"
	PermAudit         Permission = "audit.view"
	PermEquipment     Permission = "equipment.manage"
)

// PermissionInfo descrição exibida na tela de permissões
type PermissionInfo struct {
	Key   Permission `json:"key"`
	Label string     `json:"label"`
	Group string     `json:"group"`
	Hint  string     `json:"hint"`
}

// PermissionCatalog todas as permissões, na ordem em que aparecem na tela
var PermissionCatalog = []PermissionInfo{
	{PermDashboard, "Ver painel", "Geral", "Números do dia e aulas de hoje"},
	{PermReports, "Ver relatórios", "Geral", "Frequência, ocupação, evolução mensal"},
	{PermStudentsView, "Ver alunos", "Alunos", "Lista e ficha dos alunos"},
	{PermStudentsEdit, "Cadastrar e editar alunos", "Alunos", "Matrícula, troca de plano, cancelamento, senha, importação"},
	{PermCheckin, "Recepção / check-in", "Alunos", "QR de entrada e check-in manual"},
	{PermWorkouts, "Prescrever treinos", "Alunos", "Criar fichas e exercícios"},
	{PermAgendaView, "Ver agenda e chamada", "Agenda", "Lista de presença das aulas"},
	{PermAgendaManage, "Gerenciar agenda", "Agenda", "Grade, aulas avulsas, cancelamentos e substituições"},
	{PermLeads, "Leads e aulas experimentais", "Vendas", "Funil de interessados e matrícula"},
	{PermSales, "Registrar vendas", "Vendas", "Matrículas e renovações com vendedor(a)"},
	{PermGoals, "Metas e comissões", "Vendas", "Definir metas e ver comissões de todos"},
	{PermFinanceView, "Ver financeiro", "Financeiro", "Mensalidades, recebimentos e inadimplência"},
	{PermFinanceManage, "Receber e ajustar cobranças", "Financeiro", "Dar baixa, desconto, cancelar e lançar cobrança"},
	{PermCatalog, "Planos, modalidades e academia", "Configurações", "Cadastros e regras da academia"},
	{PermAnnounce, "Enviar avisos", "Configurações", "Avisos e eventos para os alunos"},
	{PermEquipment, "Equipamentos", "Configurações", "Manutenção dos aparelhos"},
	{PermStaff, "Equipe e folha", "Equipe", "Funcionários, permissões, horas/aula e pagamento"},
	{PermAudit, "Histórico de ações", "Equipe", "Quem fez o quê no sistema"},
}

// AllPermissions lista de todas as chaves
func AllPermissions() []Permission {
	out := make([]Permission, len(PermissionCatalog))
	for i, p := range PermissionCatalog {
		out[i] = p.Key
	}
	return out
}

// DefaultPermissions permissões iniciais de cada cargo (a gestão pode ajustar)
var DefaultPermissions = map[Role][]Permission{
	RoleGerente: {
		PermDashboard, PermReports, PermStudentsView, PermStudentsEdit, PermCheckin, PermWorkouts,
		PermAgendaView, PermAgendaManage, PermLeads, PermSales, PermGoals, PermFinanceView, PermFinanceManage,
		PermCatalog, PermAnnounce, PermEquipment, PermAudit,
	},
	RoleRecepcao: {
		PermDashboard, PermStudentsView, PermStudentsEdit, PermCheckin, PermAgendaView, PermLeads,
		PermFinanceView, PermFinanceManage,
	},
	RoleConsultor: {
		PermDashboard, PermStudentsView, PermLeads, PermSales, PermAgendaView,
	},
	RoleProfessor: {
		PermStudentsView, PermWorkouts, PermAgendaView, PermCheckin,
	},
}

// RolePermission permissões configuradas de um cargo na academia (sem linha = padrão)
type RolePermission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_role_perm;not null" json:"gym_id"`
	Role        Role      `gorm:"size:30;uniqueIndex:idx_role_perm;not null" json:"role"`
	Permissions string    `gorm:"type:text" json:"permissions"` // lista separada por vírgula
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *RolePermission) BeforeCreate(tx *gorm.DB) error { newID(&p.ID); return nil }

/* ----------------------------- Equipe ----------------------------- */

type PayType string

const (
	PayFixed  PayType = "FIXO"      // Salário fixo
	PayHourly PayType = "HORA_AULA" // Por hora/aula dada
	PayMixed  PayType = "MISTO"     // Fixo + hora/aula
)

// StaffProfile dados de contratação e remuneração de um membro da equipe
type StaffProfile struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	GymID             uuid.UUID  `gorm:"type:uuid;index;not null" json:"gym_id"`
	UserID            uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	HireDate          *time.Time `json:"hire_date"`
	PayType           PayType    `gorm:"size:20;default:'FIXO'" json:"pay_type"`
	BaseSalary        float64    `gorm:"default:0" json:"base_salary"`
	HourlyRate        float64    `gorm:"default:0" json:"hourly_rate"`        // Valor da hora/aula
	CommissionPercent float64    `gorm:"default:0" json:"commission_percent"` // % sobre o valor da venda
	CommissionFixed   float64    `gorm:"default:0" json:"commission_fixed"`   // Valor fixo por venda
	PixKey            string     `gorm:"size:120" json:"pix_key"`
	Notes             string     `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (s *StaffProfile) BeforeCreate(tx *gorm.DB) error { newID(&s.ID); return nil }

/* ---------------------------- Auditoria ---------------------------- */

// AuditLog registro de uma ação feita por alguém da equipe
type AuditLog struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	GymID     uuid.UUID  `gorm:"type:uuid;index;not null" json:"gym_id"`
	UserID    *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	UserName  string     `gorm:"size:255" json:"user_name"`
	UserRole  Role       `gorm:"size:30" json:"user_role"`
	Action    string     `gorm:"size:60;index;not null" json:"action"` // ex.: student.cancel
	Entity    string     `gorm:"size:40;index" json:"entity"`          // ex.: student
	EntityID  *uuid.UUID `gorm:"type:uuid;index" json:"entity_id"`
	Summary   string     `gorm:"type:text" json:"summary"`
	IP        string     `gorm:"size:64" json:"ip"`
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error { newID(&a.ID); return nil }

/* ------------------------- Vendas e metas ------------------------- */

type SaleKind string

const (
	SaleNew     SaleKind = "MATRICULA"
	SaleRenewal SaleKind = "RENOVACAO"
	SaleUpgrade SaleKind = "UPGRADE"
	SaleExtra   SaleKind = "AVULSA"
)

type SaleStatus string

const (
	SaleActive   SaleStatus = "ATIVA"
	SaleReversed SaleStatus = "ESTORNADA"
)

// Sale venda atribuída a alguém da equipe (base das comissões e metas)
type Sale struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID  `gorm:"type:uuid;index;not null" json:"gym_id"`
	SellerID    uuid.UUID  `gorm:"type:uuid;index;not null" json:"seller_id"`
	Seller      *User      `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	StudentID   *uuid.UUID `gorm:"type:uuid;index" json:"student_id"`
	Student     *Student   `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	PlanID      *uuid.UUID `gorm:"type:uuid" json:"plan_id"`
	Kind        SaleKind   `gorm:"size:20;not null" json:"kind"`
	Description string     `gorm:"size:255" json:"description"`
	Amount      float64    `gorm:"not null" json:"amount"`      // Valor do contrato
	Commission  float64    `gorm:"default:0" json:"commission"` // Calculada na hora da venda (histórico não muda)
	Status      SaleStatus `gorm:"size:20;default:'ATIVA';index" json:"status"`
	CreatedByID uuid.UUID  `gorm:"type:uuid" json:"created_by_id"`
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (s *Sale) BeforeCreate(tx *gorm.DB) error { newID(&s.ID); return nil }

// Goal meta mensal de vendas (UserID nulo = meta da academia)
type Goal struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	GymID         uuid.UUID  `gorm:"type:uuid;index;not null" json:"gym_id"`
	UserID        *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Month         string     `gorm:"size:7;index;not null" json:"month"` // AAAA-MM
	TargetSales   int        `gorm:"default:0" json:"target_sales"`
	TargetRevenue float64    `gorm:"default:0" json:"target_revenue"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (g *Goal) BeforeCreate(tx *gorm.DB) error { newID(&g.ID); return nil }

/* ------------------------------ Folha ------------------------------ */

type PayrollStatus string

const (
	PayrollOpen PayrollStatus = "ABERTO"
	PayrollPaid PayrollStatus = "PAGO"
)

// PayrollEntry fechamento do mês de uma pessoa da equipe (valores congelados ao pagar)
type PayrollEntry struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	GymID       uuid.UUID     `gorm:"type:uuid;uniqueIndex:idx_payroll_user_month;not null" json:"gym_id"`
	UserID      uuid.UUID     `gorm:"type:uuid;uniqueIndex:idx_payroll_user_month;not null" json:"user_id"`
	Month       string        `gorm:"size:7;uniqueIndex:idx_payroll_user_month;not null" json:"month"`
	BaseSalary  float64       `json:"base_salary"`
	Hours       float64       `json:"hours"`
	HoursAmount float64       `json:"hours_amount"`
	Commission  float64       `json:"commission"`
	Bonus       float64       `json:"bonus"`
	Deductions  float64       `json:"deductions"`
	Total       float64       `json:"total"`
	Status      PayrollStatus `gorm:"size:20;default:'ABERTO'" json:"status"`
	PaidAt      *time.Time    `json:"paid_at"`
	Notes       string        `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (p *PayrollEntry) BeforeCreate(tx *gorm.DB) error { newID(&p.ID); return nil }

/* --------------------------- Equipamentos --------------------------- */

type EquipmentStatus string

const (
	EquipOK          EquipmentStatus = "OPERACIONAL"
	EquipWarning     EquipmentStatus = "ATENCAO"
	EquipMaintenance EquipmentStatus = "MANUTENCAO"
)

// Equipment aparelho da academia e sua manutenção
type Equipment struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	GymID           uuid.UUID       `gorm:"type:uuid;index;not null" json:"gym_id"`
	Name            string          `gorm:"size:255;not null" json:"name"`
	Sector          string          `gorm:"size:100" json:"sector"`
	Status          EquipmentStatus `gorm:"size:20;default:'OPERACIONAL'" json:"status"`
	Notes           string          `gorm:"type:text" json:"notes"`
	LastMaintenance *time.Time      `json:"last_maintenance"`
	NextMaintenance *time.Time      `json:"next_maintenance"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (e *Equipment) BeforeCreate(tx *gorm.DB) error { newID(&e.ID); return nil }

// WebhookEvent evento de gateway já processado; o EventID único garante idempotência
type WebhookEvent struct {
	EventID   string    `gorm:"size:120;primaryKey" json:"event_id"`
	Type      string    `gorm:"size:50" json:"type"`
	CreatedAt time.Time `json:"created_at"`
}
