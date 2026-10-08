package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db      *gorm.DB
	billing *service.BillingService
	perms   *service.Permissions
}

func NewDashboardHandler(db *gorm.DB, billing *service.BillingService, perms *service.Permissions) *DashboardHandler {
	return &DashboardHandler{db: db, billing: billing, perms: perms}
}

type TodaySession struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Color    string    `json:"color"`
	Teacher  string    `json:"teacher"`
	StartsAt time.Time `json:"starts_at"`
	Capacity int       `json:"capacity"`
	Booked   int64     `json:"booked"`
	Attended int64     `json:"attended"`
	Status   string    `json:"status"`
}

type DashboardStatsResponse struct {
	TotalStudents       int64            `json:"total_students"`
	ActiveStudents      int64            `json:"active_students"`
	InactiveStudents    int64            `json:"inactive_students"`
	TodayCheckins       int64            `json:"today_checkins"`
	NewLeads            int64            `json:"new_leads"`
	Leads30             int64            `json:"leads_last30"`
	Enrolled30          int64            `json:"enrolled_last30"`
	WithoutWorkoutCount int64            `json:"without_workout_count"`
	RecentCheckins      []domain.CheckIn `json:"recent_checkins"`
	TodaySessions       []TodaySession   `json:"today_sessions"`

	// Financeiro (só para quem tem acesso ao financeiro; senão vem zerado)
	ShowFinance        bool             `json:"show_finance"`
	MonthlyRevenue     float64          `json:"monthly_revenue"`
	ExpectedRevenue    float64          `json:"expected_revenue"`
	OverdueTotal       float64          `json:"overdue_total"`
	DelinquentStudents int64            `json:"delinquent_students"`
	PendingInvoices    []domain.Invoice `json:"pending_invoices"`

	// Equipe e operação
	SalesMonth        int64 `json:"sales_month"`
	UncoveredSessions int64 `json:"uncovered_sessions"` // aulas dos próximos 7 dias sem professor(a)
	EquipmentIssues   int64 `json:"equipment_issues"`   // aparelhos em atenção/manutenção
	StaffActive       int64 `json:"staff_active"`
}

func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var stats DashboardStatsResponse
	now := config.Now()
	today := config.StartOfDay(now)
	d30 := now.AddDate(0, 0, -30)

	h.db.Model(&domain.Student{}).Where("gym_id = ?", gymID).Count(&stats.TotalStudents)
	h.db.Model(&domain.Student{}).Where("gym_id = ? AND status = ?", gymID, domain.StatusAtivo).Count(&stats.ActiveStudents)
	stats.InactiveStudents = stats.TotalStudents - stats.ActiveStudents

	h.db.Model(&domain.CheckIn{}).Where("gym_id = ? AND created_at >= ?", gymID, today).Count(&stats.TodayCheckins)
	h.db.Model(&domain.Lead{}).Where("gym_id = ? AND status = ?", gymID, domain.LeadNew).Count(&stats.NewLeads)
	h.db.Model(&domain.Lead{}).Where("gym_id = ? AND created_at >= ?", gymID, d30).Count(&stats.Leads30)
	h.db.Model(&domain.Lead{}).Where("gym_id = ? AND status = ? AND updated_at >= ?", gymID, domain.LeadEnrolled, d30).Count(&stats.Enrolled30)

	h.db.Model(&domain.Student{}).
		Where("gym_id = ? AND status = ? AND id NOT IN (?)", gymID, domain.StatusAtivo,
			h.db.Model(&domain.WorkoutPlan{}).Select("student_id").Where("gym_id = ? AND active = ?", gymID, true)).
		Count(&stats.WithoutWorkoutCount)

	h.db.Preload("Student.User").Where("gym_id = ?", gymID).Order("created_at desc").Limit(8).Find(&stats.RecentCheckins)

	var sessions []domain.ClassSession
	h.db.Preload("Modality").Preload("Teacher", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Where("gym_id = ? AND starts_at >= ? AND starts_at < ?", gymID, today.UTC(), today.AddDate(0, 0, 1).UTC()).
		Order("starts_at asc").Find(&sessions)
	stats.TodaySessions = []TodaySession{}
	for _, s := range sessions {
		ts := TodaySession{ID: s.ID, StartsAt: s.StartsAt, Capacity: s.Capacity, Status: string(s.Status), Title: s.Title}
		if s.Modality != nil {
			ts.Color = s.Modality.Color
			if ts.Title == "" {
				ts.Title = s.Modality.Name
			}
		}
		if s.Teacher != nil {
			ts.Teacher = s.Teacher.Name
		}
		h.db.Model(&domain.Booking{}).Where("session_id = ? AND status IN ?", s.ID, []domain.BookingStatus{domain.BookingConfirmed, domain.BookingAttended}).Count(&ts.Booked)
		h.db.Model(&domain.Booking{}).Where("session_id = ? AND status = ?", s.ID, domain.BookingAttended).Count(&ts.Attended)
		stats.TodaySessions = append(stats.TodaySessions, ts)
	}

	claims := middleware.GetUserFromContext(r.Context())
	stats.PendingInvoices = []domain.Invoice{}
	if h.perms.Has(gymID, claims.Role, domain.PermFinanceView) {
		stats.ShowFinance = true
		month, _ := service.ParseMonth("")
		sum := h.billing.Summary(gymID, month)
		stats.MonthlyRevenue, stats.ExpectedRevenue = sum.Received, sum.Expected
		stats.OverdueTotal, stats.DelinquentStudents = sum.OverdueTotal, sum.DelinquentStudents
		h.db.Preload("Student.User").
			Where("gym_id = ? AND (status = ? OR (status = ? AND due_date < ?))", gymID, domain.InvoiceVencido, domain.InvoicePendente, today.AddDate(0, 0, 4)).
			Order("due_date asc").Limit(8).Find(&stats.PendingInvoices)
	}
	month, _ := service.ParseMonth("")
	h.db.Model(&domain.Sale{}).Where("gym_id = ? AND status = ? AND created_at >= ?", gymID, domain.SaleActive, month).Count(&stats.SalesMonth)
	h.db.Model(&domain.ClassSession{}).Where("gym_id = ? AND status = ? AND teacher_id IS NULL AND starts_at > ? AND starts_at < ?",
		gymID, domain.SessionScheduled, time.Now(), time.Now().AddDate(0, 0, 7)).Count(&stats.UncoveredSessions)
	h.db.Model(&domain.Equipment{}).Where("gym_id = ? AND status <> ?", gymID, domain.EquipOK).Count(&stats.EquipmentIssues)
	h.db.Model(&domain.User{}).Where("gym_id = ? AND role IN ? AND active = ?", gymID, domain.StaffRoles, true).Count(&stats.StaffActive)

	respondJSON(w, http.StatusOK, stats)
}

type AbsentStudent struct {
	StudentID     uuid.UUID  `json:"student_id"`
	Name          string     `json:"name"`
	Phone         string     `json:"phone"`
	PlanName      string     `json:"plan_name"`
	LastCheckinAt *time.Time `json:"last_checkin_at"`
	DaysAbsent    int        `json:"days_absent"` // -1 = sem check-in no último ano
}

type ModalityAttendance struct {
	Modality string `json:"modality"`
	Color    string `json:"color"`
	Sessions int64  `json:"sessions"`
	Booked   int64  `json:"booked"`
	Attended int64  `json:"attended"`
	NoShow   int64  `json:"no_show"`
	Capacity int64  `json:"capacity"`
}

// GetFrequency relatório de frequência: mapa de calor, presença por modalidade, série diária e alunas sumidas
func (h *DashboardHandler) GetFrequency(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 7 || days > 180 {
		days = 30
	}
	loc := config.Loc()
	now := config.Now()
	since := config.StartOfDay(now).AddDate(0, 0, -days+1)

	// Mapa de calor dia da semana × hora (calculado em Go para funcionar em Postgres e SQLite)
	var times []time.Time
	h.db.Model(&domain.CheckIn{}).Where("gym_id = ? AND created_at >= ?", gymID, since).Pluck("created_at", &times)
	heat := make([][]int, 7)
	for i := range heat {
		heat[i] = make([]int, 24)
	}
	daily := map[string]int{}
	for _, t := range times {
		lt := t.In(loc)
		heat[int(lt.Weekday())][lt.Hour()]++
		daily[lt.Format("2006-01-02")]++
	}
	series := make([]map[string]interface{}, 0, days)
	for d := since; !d.After(now); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		series = append(series, map[string]interface{}{"date": k, "count": daily[k]})
	}

	// Presença por modalidade (aulas já realizadas no período)
	type row struct {
		ModalityID uuid.UUID
		Status     domain.BookingStatus
		N          int64
	}
	var rows []row
	h.db.Model(&domain.Booking{}).
		Select("class_sessions.modality_id as modality_id, bookings.status as status, count(*) as n").
		Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
		Where("bookings.gym_id = ? AND class_sessions.starts_at >= ? AND class_sessions.starts_at < ? AND class_sessions.status = ?", gymID, since.UTC(), time.Now(), domain.SessionScheduled).
		Group("class_sessions.modality_id, bookings.status").Scan(&rows)

	type capRow struct {
		ModalityID uuid.UUID
		Sessions   int64
		Capacity   int64
	}
	var caps []capRow
	h.db.Model(&domain.ClassSession{}).Select("modality_id, count(*) as sessions, sum(capacity) as capacity").
		Where("gym_id = ? AND starts_at >= ? AND starts_at < ? AND status = ?", gymID, since.UTC(), time.Now(), domain.SessionScheduled).
		Group("modality_id").Scan(&caps)

	var modalities []domain.Modality
	h.db.Where("gym_id = ?", gymID).Find(&modalities)
	byMod := map[uuid.UUID]*ModalityAttendance{}
	for _, m := range modalities {
		byMod[m.ID] = &ModalityAttendance{Modality: m.Name, Color: m.Color}
	}
	for _, c := range caps {
		if m := byMod[c.ModalityID]; m != nil {
			m.Sessions, m.Capacity = c.Sessions, c.Capacity
		}
	}
	for _, rr := range rows {
		m := byMod[rr.ModalityID]
		if m == nil {
			continue
		}
		switch rr.Status {
		case domain.BookingAttended:
			m.Attended += rr.N
			m.Booked += rr.N
		case domain.BookingNoShow:
			m.NoShow += rr.N
			m.Booked += rr.N
		case domain.BookingConfirmed:
			m.Booked += rr.N
		}
	}
	attendance := []ModalityAttendance{}
	for _, m := range byMod {
		if m.Sessions > 0 || m.Booked > 0 {
			attendance = append(attendance, *m)
		}
	}
	sort.Slice(attendance, func(i, j int) bool { return attendance[i].Booked > attendance[j].Booked })

	// Alunas ativas sem check-in há 7+ dias
	var students []domain.Student
	h.db.Preload("User").Where("gym_id = ? AND status = ?", gymID, domain.StatusAtivo).Find(&students)
	// Último check-in de cada aluna no último ano (max calculado em Go: portável entre Postgres e SQLite)
	var recent []domain.CheckIn
	h.db.Select("student_id", "created_at").Where("gym_id = ? AND created_at >= ?", gymID, now.AddDate(-1, 0, 0)).Find(&recent)
	lastBy := map[uuid.UUID]time.Time{}
	for _, c := range recent {
		if c.CreatedAt.After(lastBy[c.StudentID]) {
			lastBy[c.StudentID] = c.CreatedAt
		}
	}
	absent := []AbsentStudent{}
	for _, s := range students {
		a := AbsentStudent{StudentID: s.ID, Name: s.User.Name, Phone: s.User.Phone, PlanName: s.PlanName, DaysAbsent: -1}
		if t, ok := lastBy[s.ID]; ok {
			tt := t
			a.LastCheckinAt = &tt
			a.DaysAbsent = int(now.Sub(t).Hours() / 24)
			if a.DaysAbsent < 7 {
				continue
			}
		} else if now.Sub(s.CreatedAt) < 7*24*time.Hour {
			continue // matrícula recente, ainda não conta como sumida
		}
		absent = append(absent, a)
	}
	sort.Slice(absent, func(i, j int) bool {
		di, dj := absent[i].DaysAbsent, absent[j].DaysAbsent
		if di == -1 {
			di = 1 << 30
		}
		if dj == -1 {
			dj = 1 << 30
		}
		return di > dj
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"days":       days,
		"total":      len(times),
		"heatmap":    heat, // [domingo..sábado][0..23h]
		"daily":      series,
		"attendance": attendance,
		"absent":     absent,
	})
}

/* ------------------------- Visão do dono ------------------------- */

type MonthOverview struct {
	Month       string  `json:"month"`
	NewStudents int64   `json:"new_students"`
	Canceled    int64   `json:"canceled"`
	Revenue     float64 `json:"revenue"`
	Expected    float64 `json:"expected"`
	Sales       int64   `json:"sales"`
	SalesAmount float64 `json:"sales_amount"`
	Leads       int64   `json:"leads"`
	Checkins    int64   `json:"checkins"`
}

type TeacherOccupancy struct {
	Name      string  `json:"name"`
	Sessions  int     `json:"sessions"`
	Capacity  int     `json:"capacity"`
	Attended  int64   `json:"attended"`
	Occupancy float64 `json:"occupancy"`
}

type SlotOccupancy struct {
	Weekday   int     `json:"weekday"`
	Hour      int     `json:"hour"`
	Sessions  int     `json:"sessions"`
	Capacity  int     `json:"capacity"`
	Booked    int64   `json:"booked"`
	Occupancy float64 `json:"occupancy"`
}

type PlanMix struct {
	Plan     string  `json:"plan"`
	Students int64   `json:"students"`
	Monthly  float64 `json:"monthly"`
}

// GetOverview evolução mensal, retenção, ocupação por professor(a) e por horário, mix de planos
func (h *DashboardHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	n, _ := strconv.Atoi(r.URL.Query().Get("months"))
	if n < 3 || n > 24 {
		n = 6
	}
	current, _ := service.ParseMonth("")
	months := []MonthOverview{}
	for i := n - 1; i >= 0; i-- {
		start := current.AddDate(0, -i, 0)
		end := start.AddDate(0, 1, 0)
		m := MonthOverview{Month: start.Format("2006-01")}
		h.db.Unscoped().Model(&domain.Student{}).Where("gym_id = ? AND created_at >= ? AND created_at < ?", gymID, start, end).Count(&m.NewStudents)
		h.db.Unscoped().Model(&domain.Student{}).Where("gym_id = ? AND canceled_at >= ? AND canceled_at < ?", gymID, start, end).Count(&m.Canceled)
		h.db.Model(&domain.Invoice{}).Select("COALESCE(SUM(paid_amount),0)").Where("gym_id = ? AND status = ? AND paid_at >= ? AND paid_at < ?", gymID, domain.InvoicePago, start, end).Scan(&m.Revenue)
		h.db.Model(&domain.Invoice{}).Select("COALESCE(SUM(amount - discount),0)").Where("gym_id = ? AND status <> ? AND due_date >= ? AND due_date < ?", gymID, domain.InvoiceCancel, start, end).Scan(&m.Expected)
		h.db.Model(&domain.Sale{}).Where("gym_id = ? AND status = ? AND created_at >= ? AND created_at < ?", gymID, domain.SaleActive, start, end).Count(&m.Sales)
		h.db.Model(&domain.Sale{}).Select("COALESCE(SUM(amount),0)").Where("gym_id = ? AND status = ? AND created_at >= ? AND created_at < ?", gymID, domain.SaleActive, start, end).Scan(&m.SalesAmount)
		h.db.Model(&domain.Lead{}).Where("gym_id = ? AND created_at >= ? AND created_at < ?", gymID, start, end).Count(&m.Leads)
		h.db.Model(&domain.CheckIn{}).Where("gym_id = ? AND created_at >= ? AND created_at < ?", gymID, start, end).Count(&m.Checkins)
		months = append(months, m)
	}

	var active, payers int64
	h.db.Model(&domain.Student{}).Where("gym_id = ? AND status IN ?", gymID, []domain.StudentStatus{domain.StatusAtivo, domain.StatusInadimplente}).Count(&active)
	h.db.Model(&domain.Invoice{}).Where("gym_id = ? AND status = ? AND paid_at >= ?", gymID, domain.InvoicePago, current).Distinct("student_id").Count(&payers)
	last := months[len(months)-1]
	churn, ticket := 0.0, 0.0
	if active+last.Canceled > 0 {
		churn = float64(last.Canceled) / float64(active+last.Canceled) * 100
	}
	if payers > 0 {
		ticket = last.Revenue / float64(payers)
	}

	// Ocupação nos últimos 30 dias (aulas já realizadas)
	since := time.Now().AddDate(0, 0, -30)
	var sessions []domain.ClassSession
	h.db.Preload("Teacher", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Select("id", "name") }).
		Where("gym_id = ? AND status = ? AND starts_at >= ? AND ends_at < ?", gymID, domain.SessionScheduled, since, time.Now()).Find(&sessions)
	ids := make([]uuid.UUID, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	type cnt struct {
		SessionID uuid.UUID
		Status    domain.BookingStatus
		N         int64
	}
	var counts []cnt
	if len(ids) > 0 {
		h.db.Model(&domain.Booking{}).Select("session_id, status, count(*) as n").Where("session_id IN ?", ids).Group("session_id, status").Scan(&counts)
	}
	attended, booked := map[uuid.UUID]int64{}, map[uuid.UUID]int64{}
	for _, c := range counts {
		switch c.Status {
		case domain.BookingAttended:
			attended[c.SessionID] += c.N
			booked[c.SessionID] += c.N
		case domain.BookingNoShow, domain.BookingConfirmed:
			booked[c.SessionID] += c.N
		}
	}
	byTeacher := map[string]*TeacherOccupancy{}
	bySlot := map[[2]int]*SlotOccupancy{}
	loc := config.Loc()
	for _, s := range sessions {
		name := "Sem professor(a)"
		if s.Teacher != nil {
			name = s.Teacher.Name
		}
		t := byTeacher[name]
		if t == nil {
			t = &TeacherOccupancy{Name: name}
			byTeacher[name] = t
		}
		t.Sessions++
		t.Capacity += s.Capacity
		t.Attended += attended[s.ID]

		lt := s.StartsAt.In(loc)
		key := [2]int{int(lt.Weekday()), lt.Hour()}
		sl := bySlot[key]
		if sl == nil {
			sl = &SlotOccupancy{Weekday: key[0], Hour: key[1]}
			bySlot[key] = sl
		}
		sl.Sessions++
		sl.Capacity += s.Capacity
		sl.Booked += booked[s.ID]
	}
	teachers := []TeacherOccupancy{}
	for _, t := range byTeacher {
		if t.Capacity > 0 {
			t.Occupancy = float64(t.Attended) / float64(t.Capacity) * 100
		}
		teachers = append(teachers, *t)
	}
	sort.Slice(teachers, func(i, j int) bool { return teachers[i].Occupancy > teachers[j].Occupancy })
	slots := []SlotOccupancy{}
	for _, s := range bySlot {
		if s.Capacity > 0 {
			s.Occupancy = float64(s.Booked) / float64(s.Capacity) * 100
		}
		slots = append(slots, *s)
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].Weekday != slots[j].Weekday {
			return slots[i].Weekday < slots[j].Weekday
		}
		return slots[i].Hour < slots[j].Hour
	})

	var mix []PlanMix
	h.db.Model(&domain.Student{}).Select("COALESCE(NULLIF(plan_name, ''), 'Sem plano') as plan, count(*) as students, COALESCE(SUM(plan_price),0) as monthly").
		Where("gym_id = ? AND status IN ?", gymID, []domain.StudentStatus{domain.StatusAtivo, domain.StatusInadimplente}).
		Group("COALESCE(NULLIF(plan_name, ''), 'Sem plano')").Order("students desc").Scan(&mix)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"months":     months,
		"active":     active,
		"churn_rate": churn,
		"ticket":     ticket,
		"payers":     payers,
		"by_teacher": teachers,
		"by_slot":    slots,
		"plan_mix":   mix,
	})
}
