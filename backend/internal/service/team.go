package service

import (
	"sort"
	"strings"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeamService vendas/comissões, horas/aula e fechamento da equipe
type TeamService struct {
	db *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{db: db}
}

/* ------------------------------ Vendas ------------------------------ */

// CommissionFor comissão de uma venda pelo perfil de quem vendeu
func CommissionFor(p *domain.StaffProfile, amount float64) float64 {
	if p == nil || amount <= 0 {
		return 0
	}
	return round2(amount*p.CommissionPercent/100 + p.CommissionFixed)
}

// ContractValue valor do contrato do plano (mensal × meses)
func ContractValue(p *domain.Plan) float64 {
	if p == nil {
		return 0
	}
	months := p.DurationMonths
	if months < 1 {
		months = 1
	}
	return round2(p.Price * float64(months))
}

// SaleInput dados de uma venda
type SaleInput struct {
	SellerID    uuid.UUID
	StudentID   *uuid.UUID
	PlanID      *uuid.UUID
	Kind        domain.SaleKind
	Amount      float64 // 0 = valor do contrato do plano
	Description string
	CreatedBy   uuid.UUID
}

var saleKinds = map[domain.SaleKind]bool{domain.SaleNew: true, domain.SaleRenewal: true, domain.SaleUpgrade: true, domain.SaleExtra: true}

// RegisterSale grava a venda com a comissão calculada no momento (mudar a regra depois não altera o histórico)
func (t *TeamService) RegisterSale(tx *gorm.DB, gymID uuid.UUID, in SaleInput) (*domain.Sale, error) {
	if !saleKinds[in.Kind] {
		return nil, bizErr("Tipo de venda inválido")
	}
	var seller domain.User
	if err := tx.First(&seller, "id = ? AND gym_id = ? AND active = ?", in.SellerID, gymID, true).Error; err != nil || !domain.IsStaffRole(seller.Role) {
		return nil, bizErr("Vendedor(a) não encontrado(a) na equipe")
	}
	var plan *domain.Plan
	if in.PlanID != nil && *in.PlanID != uuid.Nil {
		var p domain.Plan
		if err := tx.First(&p, "id = ? AND gym_id = ?", *in.PlanID, gymID).Error; err != nil {
			return nil, bizErr("Plano não encontrado")
		}
		plan = &p
	} else {
		in.PlanID = nil
	}
	if in.StudentID != nil {
		var n int64
		tx.Model(&domain.Student{}).Where("id = ? AND gym_id = ?", *in.StudentID, gymID).Count(&n)
		if n == 0 {
			return nil, bizErr("Aluno não encontrado")
		}
	}
	amount := in.Amount
	if amount <= 0 {
		amount = ContractValue(plan)
	}
	if amount <= 0 {
		return nil, bizErr("Informe o valor da venda")
	}
	desc := strings.TrimSpace(in.Description)
	if desc == "" && plan != nil {
		desc = plan.Name
	}
	var profile domain.StaffProfile
	var pp *domain.StaffProfile
	if tx.Where("user_id = ?", seller.ID).First(&profile).Error == nil {
		pp = &profile
	}
	sale := domain.Sale{
		GymID: gymID, SellerID: seller.ID, StudentID: in.StudentID, PlanID: in.PlanID, Kind: in.Kind,
		Description: desc, Amount: round2(amount), Commission: CommissionFor(pp, amount),
		Status: domain.SaleActive, CreatedByID: in.CreatedBy,
	}
	if err := tx.Create(&sale).Error; err != nil {
		return nil, err
	}
	return &sale, nil
}

// RankingRow desempenho de vendas de uma pessoa no mês
type RankingRow struct {
	UserID         uuid.UUID   `json:"user_id"`
	Name           string      `json:"name"`
	Role           domain.Role `json:"role"`
	Sales          int         `json:"sales"`
	NewSales       int         `json:"new_sales"`
	Renewals       int         `json:"renewals"`
	Revenue        float64     `json:"revenue"`
	Commission     float64     `json:"commission"`
	GoalSales      int         `json:"goal_sales"`
	GoalRevenue    float64     `json:"goal_revenue"`
	Progress       float64     `json:"progress"` // 0..100+ (pelo faturamento, ou pela quantidade se não houver meta de valor)
	LeadsAssigned  int64       `json:"leads_assigned"`
	LeadsConverted int64       `json:"leads_converted"`
	Conversion     float64     `json:"conversion"` // %
}

// RankingResult ranking + meta geral da academia
type RankingResult struct {
	Month       string       `json:"month"`
	Rows        []RankingRow `json:"rows"`
	TotalSales  int          `json:"total_sales"`
	TotalAmount float64      `json:"total_amount"`
	TotalComm   float64      `json:"total_commission"`
	GymGoal     *domain.Goal `json:"gym_goal"`
	GymProgress float64      `json:"gym_progress"`
}

func progress(goalSales int, goalRevenue float64, sales int, revenue float64) float64 {
	if goalRevenue > 0 {
		return round2(revenue / goalRevenue * 100)
	}
	if goalSales > 0 {
		return round2(float64(sales) / float64(goalSales) * 100)
	}
	return 0
}

// Ranking vendas por pessoa no mês, com metas e conversão de leads
func (t *TeamService) Ranking(gymID uuid.UUID, month time.Time) RankingResult {
	end := month.AddDate(0, 1, 0)
	ref := month.Format("2006-01")
	res := RankingResult{Month: ref, Rows: []RankingRow{}}

	var sales []domain.Sale
	t.db.Where("gym_id = ? AND status = ? AND created_at >= ? AND created_at < ?", gymID, domain.SaleActive, month, end).Find(&sales)

	var goals []domain.Goal
	t.db.Where("gym_id = ? AND month = ?", gymID, ref).Find(&goals)
	goalBy := map[uuid.UUID]domain.Goal{}
	for i, g := range goals {
		if g.UserID == nil {
			res.GymGoal = &goals[i]
		} else {
			goalBy[*g.UserID] = g
		}
	}

	// Participam: quem vendeu, quem tem meta e quem pode registrar vendas (consultores/gerentes ativos)
	var users []domain.User
	t.db.Where("gym_id = ? AND role IN ?", gymID, []domain.Role{domain.RoleAdmin, domain.RoleGerente, domain.RoleConsultor, domain.RoleRecepcao, domain.RoleProfessor}).Find(&users)
	rows := map[uuid.UUID]*RankingRow{}
	include := map[uuid.UUID]bool{}
	for _, s := range sales {
		include[s.SellerID] = true
	}
	for id := range goalBy {
		include[id] = true
	}
	for _, u := range users {
		if u.Active && u.Role == domain.RoleConsultor {
			include[u.ID] = true
		}
	}
	for _, u := range users {
		if include[u.ID] {
			g := goalBy[u.ID]
			rows[u.ID] = &RankingRow{UserID: u.ID, Name: u.Name, Role: u.Role, GoalSales: g.TargetSales, GoalRevenue: g.TargetRevenue}
		}
	}
	for _, s := range sales {
		r := rows[s.SellerID]
		if r == nil {
			continue
		}
		r.Sales++
		r.Revenue += s.Amount
		r.Commission += s.Commission
		switch s.Kind {
		case domain.SaleNew:
			r.NewSales++
		case domain.SaleRenewal, domain.SaleUpgrade:
			r.Renewals++
		}
		res.TotalSales++
		res.TotalAmount += s.Amount
		res.TotalComm += s.Commission
	}

	// Conversão: leads atribuídos criados no mês × quantos viraram matrícula
	type leadRow struct {
		AssignedToID uuid.UUID
		Status       domain.LeadStatus
		N            int64
	}
	var lr []leadRow
	t.db.Model(&domain.Lead{}).Select("assigned_to_id, status, count(*) as n").
		Where("gym_id = ? AND assigned_to_id IS NOT NULL AND created_at >= ? AND created_at < ?", gymID, month, end).
		Group("assigned_to_id, status").Scan(&lr)
	for _, l := range lr {
		if r := rows[l.AssignedToID]; r != nil {
			r.LeadsAssigned += l.N
			if l.Status == domain.LeadEnrolled {
				r.LeadsConverted += l.N
			}
		}
	}

	for _, r := range rows {
		r.Revenue, r.Commission = round2(r.Revenue), round2(r.Commission)
		r.Progress = progress(r.GoalSales, r.GoalRevenue, r.Sales, r.Revenue)
		if r.LeadsAssigned > 0 {
			r.Conversion = round2(float64(r.LeadsConverted) / float64(r.LeadsAssigned) * 100)
		}
		res.Rows = append(res.Rows, *r)
	}
	sort.Slice(res.Rows, func(i, j int) bool {
		if res.Rows[i].Revenue != res.Rows[j].Revenue {
			return res.Rows[i].Revenue > res.Rows[j].Revenue
		}
		return res.Rows[i].Name < res.Rows[j].Name
	})
	res.TotalAmount, res.TotalComm = round2(res.TotalAmount), round2(res.TotalComm)
	if res.GymGoal != nil {
		res.GymProgress = progress(res.GymGoal.TargetSales, res.GymGoal.TargetRevenue, res.TotalSales, res.TotalAmount)
	}
	return res
}

/* ---------------------------- Horas/aula ---------------------------- */

// TeacherHours aulas dadas por professor(a) no mês
type TeacherHours struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Active      bool      `json:"active"`
	Given       int       `json:"given"`     // aulas já realizadas
	Hours       float64   `json:"hours"`     // horas dadas
	Upcoming    int       `json:"upcoming"`  // aulas ainda por vir no mês
	Covered     int       `json:"covered"`   // aulas que cobriu de outra pessoa (substituição)
	Capacity    int       `json:"capacity"`  // vagas das aulas dadas
	Attended    int64     `json:"attended"`  // presenças nas aulas dadas
	Occupancy   float64   `json:"occupancy"` // presenças / vagas (%)
	HourlyRate  float64   `json:"hourly_rate"`
	HoursAmount float64   `json:"hours_amount"`
}

// Hours calcula horas/aula do mês por professor(a) (aulas canceladas não contam)
func (t *TeamService) Hours(gymID uuid.UUID, month time.Time) []TeacherHours {
	end := month.AddDate(0, 1, 0)
	now := time.Now()

	var sessions []domain.ClassSession
	t.db.Where("gym_id = ? AND status = ? AND teacher_id IS NOT NULL AND starts_at >= ? AND starts_at < ?",
		gymID, domain.SessionScheduled, month.UTC(), end.UTC()).Find(&sessions)

	scheduleTeacher := map[uuid.UUID]*uuid.UUID{}
	var schedules []domain.ClassSchedule
	t.db.Unscoped().Select("id", "teacher_id").Where("gym_id = ?", gymID).Find(&schedules)
	for _, s := range schedules {
		scheduleTeacher[s.ID] = s.TeacherID
	}

	var given []uuid.UUID
	for _, s := range sessions {
		if s.EndsAt.Before(now) {
			given = append(given, s.ID)
		}
	}
	attended := map[uuid.UUID]int64{}
	if len(given) > 0 {
		type row struct {
			SessionID uuid.UUID
			N         int64
		}
		var rows []row
		t.db.Model(&domain.Booking{}).Select("session_id, count(*) as n").
			Where("session_id IN ? AND status = ?", given, domain.BookingAttended).Group("session_id").Scan(&rows)
		for _, r := range rows {
			attended[r.SessionID] = r.N
		}
	}

	var users []domain.User
	t.db.Unscoped().Where("gym_id = ? AND (role = ? OR id IN (?))", gymID, domain.RoleProfessor,
		t.db.Model(&domain.ClassSession{}).Select("teacher_id").Where("gym_id = ? AND teacher_id IS NOT NULL AND starts_at >= ? AND starts_at < ?", gymID, month.UTC(), end.UTC())).
		Order("name asc").Find(&users)
	profiles := t.profiles(gymID)

	byUser := map[uuid.UUID]*TeacherHours{}
	out := []TeacherHours{}
	for _, u := range users {
		if u.DeletedAt.Valid && !hasTeacher(sessions, u.ID) {
			continue
		}
		th := &TeacherHours{UserID: u.ID, Name: u.Name, Active: u.Active && !u.DeletedAt.Valid}
		if p, ok := profiles[u.ID]; ok {
			th.HourlyRate = p.HourlyRate
		}
		byUser[u.ID] = th
	}
	for _, s := range sessions {
		th := byUser[*s.TeacherID]
		if th == nil {
			continue
		}
		if s.EndsAt.Before(now) {
			th.Given++
			th.Hours += s.EndsAt.Sub(s.StartsAt).Hours()
			th.Capacity += s.Capacity
			th.Attended += attended[s.ID]
		} else {
			th.Upcoming++
		}
		if s.ScheduleID != nil {
			if orig, ok := scheduleTeacher[*s.ScheduleID]; ok && (orig == nil || *orig != *s.TeacherID) {
				th.Covered++
			}
		}
	}
	for _, u := range users {
		th := byUser[u.ID]
		if th == nil {
			continue
		}
		th.Hours = round2(th.Hours)
		if th.Capacity > 0 {
			th.Occupancy = round2(float64(th.Attended) / float64(th.Capacity) * 100)
		}
		th.HoursAmount = round2(th.Hours * th.HourlyRate)
		out = append(out, *th)
	}
	return out
}

func hasTeacher(sessions []domain.ClassSession, id uuid.UUID) bool {
	for _, s := range sessions {
		if s.TeacherID != nil && *s.TeacherID == id {
			return true
		}
	}
	return false
}

func (t *TeamService) profiles(gymID uuid.UUID) map[uuid.UUID]domain.StaffProfile {
	var list []domain.StaffProfile
	t.db.Where("gym_id = ?", gymID).Find(&list)
	m := map[uuid.UUID]domain.StaffProfile{}
	for _, p := range list {
		m[p.UserID] = p
	}
	return m
}

/* ------------------------------- Folha ------------------------------- */

// PayrollRow valores do mês de uma pessoa da equipe
type PayrollRow struct {
	UserID      uuid.UUID            `json:"user_id"`
	Name        string               `json:"name"`
	Role        domain.Role          `json:"role"`
	JobTitle    string               `json:"job_title"`
	PayType     domain.PayType       `json:"pay_type"`
	PixKey      string               `json:"pix_key"`
	BaseSalary  float64              `json:"base_salary"`
	Hours       float64              `json:"hours"`
	HourlyRate  float64              `json:"hourly_rate"`
	HoursAmount float64              `json:"hours_amount"`
	Commission  float64              `json:"commission"`
	Bonus       float64              `json:"bonus"`
	Deductions  float64              `json:"deductions"`
	Total       float64              `json:"total"`
	Status      domain.PayrollStatus `json:"status"`
	PaidAt      *time.Time           `json:"paid_at"`
	Notes       string               `json:"notes"`
}

// PayrollTotal soma do fechamento
func PayrollTotal(base, hoursAmount, commission, bonus, deductions float64) float64 {
	return round2(base + hoursAmount + commission + bonus - deductions)
}

// ComputeRow valores calculados (sem ajustes) a partir do perfil, horas e comissões
func ComputeRow(p *domain.StaffProfile, hours, commission float64) (base, hoursAmount float64) {
	if p == nil {
		return 0, 0
	}
	switch p.PayType {
	case domain.PayHourly:
		return 0, round2(hours * p.HourlyRate)
	case domain.PayMixed:
		return p.BaseSalary, round2(hours * p.HourlyRate)
	default:
		return p.BaseSalary, 0
	}
}

// Payroll fechamento do mês: valores ao vivo para quem está em aberto, congelados para quem já foi pago
func (t *TeamService) Payroll(gymID uuid.UUID, month time.Time) []PayrollRow {
	ref := month.Format("2006-01")
	end := month.AddDate(0, 1, 0)

	var entries []domain.PayrollEntry
	t.db.Where("gym_id = ? AND month = ?", gymID, ref).Find(&entries)
	entryBy := map[uuid.UUID]domain.PayrollEntry{}
	for _, e := range entries {
		entryBy[e.UserID] = e
	}

	var users []domain.User
	t.db.Unscoped().Where("gym_id = ? AND role IN ?", gymID, domain.StaffRoles).Order("name asc").Find(&users)
	profiles := t.profiles(gymID)

	hoursBy := map[uuid.UUID]float64{}
	for _, h := range t.Hours(gymID, month) {
		hoursBy[h.UserID] = h.Hours
	}
	type commRow struct {
		SellerID uuid.UUID
		Total    float64
	}
	var comms []commRow
	t.db.Model(&domain.Sale{}).Select("seller_id, sum(commission) as total").
		Where("gym_id = ? AND status = ? AND created_at >= ? AND created_at < ?", gymID, domain.SaleActive, month, end).
		Group("seller_id").Scan(&comms)
	commBy := map[uuid.UUID]float64{}
	for _, c := range comms {
		commBy[c.SellerID] = round2(c.Total)
	}

	rows := []PayrollRow{}
	for _, u := range users {
		entry, hasEntry := entryBy[u.ID]
		active := u.Active && !u.DeletedAt.Valid
		if !active && !hasEntry && hoursBy[u.ID] == 0 && commBy[u.ID] == 0 {
			continue
		}
		var pp *domain.StaffProfile
		if p, ok := profiles[u.ID]; ok {
			pp = &p
		}
		row := PayrollRow{UserID: u.ID, Name: u.Name, Role: u.Role, JobTitle: u.JobTitle, Status: domain.PayrollOpen}
		if pp != nil {
			row.PayType, row.PixKey, row.HourlyRate = pp.PayType, pp.PixKey, pp.HourlyRate
		}
		if hasEntry && entry.Status == domain.PayrollPaid {
			row.BaseSalary, row.Hours, row.HoursAmount, row.Commission = entry.BaseSalary, entry.Hours, entry.HoursAmount, entry.Commission
			row.Bonus, row.Deductions, row.Total, row.Status, row.PaidAt, row.Notes = entry.Bonus, entry.Deductions, entry.Total, entry.Status, entry.PaidAt, entry.Notes
			rows = append(rows, row)
			continue
		}
		row.Hours = hoursBy[u.ID]
		row.Commission = commBy[u.ID]
		row.BaseSalary, row.HoursAmount = ComputeRow(pp, row.Hours, row.Commission)
		if hasEntry {
			row.Bonus, row.Deductions, row.Notes = entry.Bonus, entry.Deductions, entry.Notes
		}
		row.Total = PayrollTotal(row.BaseSalary, row.HoursAmount, row.Commission, row.Bonus, row.Deductions)
		rows = append(rows, row)
	}
	return rows
}

// PayrollAdjust ajustes do fechamento de uma pessoa
type PayrollAdjust struct {
	Bonus      float64
	Deductions float64
	Notes      string
	MarkPaid   bool
	Reopen     bool
}

// SavePayroll grava bônus/descontos e, se pedido, congela os valores e marca como pago
func (t *TeamService) SavePayroll(gymID, userID uuid.UUID, month time.Time, adj PayrollAdjust) (*PayrollRow, error) {
	if adj.Bonus < 0 || adj.Deductions < 0 {
		return nil, bizErr("Bônus e descontos não podem ser negativos")
	}
	ref := month.Format("2006-01")
	var entry domain.PayrollEntry
	exists := t.db.Where("gym_id = ? AND user_id = ? AND month = ?", gymID, userID, ref).First(&entry).Error == nil
	if exists && entry.Status == domain.PayrollPaid && !adj.Reopen {
		return nil, bizErr("Este fechamento já foi pago. Reabra para alterar.")
	}

	if adj.Reopen {
		if !exists {
			return nil, bizErr("Nada para reabrir")
		}
		entry.Status = domain.PayrollOpen
		entry.PaidAt = nil
		if err := t.db.Save(&entry).Error; err != nil {
			return nil, err
		}
	} else {
		entry.GymID, entry.UserID, entry.Month = gymID, userID, ref
		entry.Bonus, entry.Deductions, entry.Notes = round2(adj.Bonus), round2(adj.Deductions), adj.Notes
		entry.Status = domain.PayrollOpen
		// Grava primeiro como aberto para o cálculo abaixo considerar os ajustes
		var err error
		if exists {
			err = t.db.Save(&entry).Error
		} else {
			err = t.db.Create(&entry).Error
		}
		if err != nil {
			return nil, err
		}
	}

	var row *PayrollRow
	for _, r := range t.Payroll(gymID, month) {
		if r.UserID == userID {
			rr := r
			row = &rr
			break
		}
	}
	if row == nil {
		return nil, bizErr("Pessoa não encontrada na equipe")
	}
	if adj.MarkPaid && !adj.Reopen {
		now := time.Now()
		entry.BaseSalary, entry.Hours, entry.HoursAmount, entry.Commission = row.BaseSalary, row.Hours, row.HoursAmount, row.Commission
		entry.Total, entry.Status, entry.PaidAt = row.Total, domain.PayrollPaid, &now
		if err := t.db.Save(&entry).Error; err != nil {
			return nil, err
		}
		row.Status, row.PaidAt = domain.PayrollPaid, &now
	}
	return row, nil
}
