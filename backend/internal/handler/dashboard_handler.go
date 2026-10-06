package handler

import (
	"net/http"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

type DashboardStatsResponse struct {
	TotalStudents       int64             `json:"total_students"`
	ActiveStudents      int64             `json:"active_students"`
	DelinquentStudents  int64             `json:"delinquent_students"`
	MonthlyRevenue      float64           `json:"monthly_revenue"`
	TodayCheckins       int64             `json:"today_checkins"`
	WithoutWorkoutCount int64             `json:"without_workout_count"`
	RecentCheckins      []domain.CheckIn  `json:"recent_checkins"`
	PendingInvoices     []domain.Invoice  `json:"pending_invoices"`
}

func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	if gymID == uuid.Nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Tenant não identificado"})
		return
	}

	var stats DashboardStatsResponse

	// Alunos totais, ativos e inadimplentes
	h.db.Model(&domain.Student{}).Where("gym_id = ?", gymID).Count(&stats.TotalStudents)
	h.db.Model(&domain.Student{}).Where("gym_id = ? AND status = ?", gymID, domain.StatusAtivo).Count(&stats.ActiveStudents)
	h.db.Model(&domain.Student{}).Where("gym_id = ? AND status = ?", gymID, domain.StatusInadimplente).Count(&stats.DelinquentStudents)

	// Faturamento mensal estimado (soma dos planos dos ativos)
	type RevenueSum struct {
		Total float64
	}
	var revSum RevenueSum
	h.db.Model(&domain.Student{}).
		Select("COALESCE(SUM(plan_price), 0) as total").
		Where("gym_id = ? AND status = ?", gymID, domain.StatusAtivo).
		Scan(&revSum)
	stats.MonthlyRevenue = revSum.Total

	// Presenças de hoje
	startOfDay := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&domain.CheckIn{}).
		Where("gym_id = ? AND created_at >= ?", gymID, startOfDay).
		Count(&stats.TodayCheckins)

	// Últimos 5 check-ins
	h.db.Preload("Student.User").
		Where("gym_id = ?", gymID).
		Order("created_at desc").
		Limit(5).
		Find(&stats.RecentCheckins)

	// Últimas faturas pendentes ou atrasadas
	h.db.Preload("Student.User").
		Where("gym_id = ? AND status IN (?, ?)", gymID, domain.InvoicePendente, domain.InvoiceVencido).
		Order("due_date asc").
		Limit(5).
		Find(&stats.PendingInvoices)

	// Alunos sem ficha ativa
	var studentsWithoutWorkout int64
	h.db.Model(&domain.Student{}).
		Where("gym_id = ? AND id NOT IN (?)", gymID, h.db.Model(&domain.WorkoutPlan{}).Select("student_id").Where("gym_id = ? AND active = true", gymID)).
		Count(&studentsWithoutWorkout)
	stats.WithoutWorkoutCount = studentsWithoutWorkout

	respondJSON(w, http.StatusOK, stats)
}
