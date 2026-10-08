package service

import (
	"fmt"
	"log"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Scheduler rotinas periódicas: lembretes de aula, lembretes de vencimento,
// faltas automáticas e geração antecipada da agenda.
type Scheduler struct {
	db       *gorm.DB
	notifier *Notifier
	agenda   *AgendaService
	billing  *BillingService
	lastBill map[uuid.UUID]string // data (AAAA-MM-DD) da última rotina de cobrança por academia
}

func NewScheduler(db *gorm.DB, notifier *Notifier, agenda *AgendaService, billing *BillingService) *Scheduler {
	return &Scheduler{db: db, notifier: notifier, agenda: agenda, billing: billing, lastBill: map[uuid.UUID]string{}}
}

func (s *Scheduler) Start() {
	go func() {
		s.tick()
		t := time.NewTicker(5 * time.Minute)
		for range t.C {
			s.tick()
		}
	}()
}

func (s *Scheduler) tick() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SCHEDULER] pânico recuperado: %v", r)
		}
	}()
	var gyms []domain.Gym
	s.db.Find(&gyms)
	for _, g := range gyms {
		window := g.BookingWindowDays
		if window <= 0 {
			window = 7
		}
		now := config.Now()
		if err := s.agenda.EnsureSessions(g.ID, now, now.AddDate(0, 0, window+1)); err != nil {
			log.Printf("[SCHEDULER] falha ao gerar agenda: %v", err)
		}
		s.classReminders(g.ID)
		s.markNoShows(g.ID)
		s.dueReminders(g.ID)
		// Cobrança: uma vez por dia (gera mensalidades e atualiza atrasos)
		if today := now.Format("2006-01-02"); s.lastBill[g.ID] != today {
			if err := s.billing.RunDaily(g.ID); err != nil {
				log.Printf("[SCHEDULER] falha na rotina de cobrança: %v", err)
			} else {
				s.lastBill[g.ID] = today
			}
		}
	}
}

// classReminders avisa ~1h antes da aula reservada
func (s *Scheduler) classReminders(gymID uuid.UUID) {
	now := time.Now()
	var bookings []domain.Booking
	s.db.Preload("Session.Modality").
		Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
		Where("bookings.gym_id = ? AND bookings.status = ? AND bookings.reminder_sent_at IS NULL AND bookings.student_id IS NOT NULL", gymID, domain.BookingConfirmed).
		Where("class_sessions.status = ? AND class_sessions.starts_at > ? AND class_sessions.starts_at <= ?", domain.SessionScheduled, now, now.Add(75*time.Minute)).
		Find(&bookings)

	for _, b := range bookings {
		t := b.Session.StartsAt.In(config.Loc())
		s.notifier.NotifyStudents(gymID, []uuid.UUID{*b.StudentID}, domain.NotifyReminder,
			"Sua aula está chegando ⏰",
			fmt.Sprintf("%s hoje às %02d:%02d. Não esqueça de fazer o check-in na recepção!", modalityName(b.Session), t.Hour(), t.Minute()),
			"/aluno/?tab=agenda")
		s.db.Model(&domain.Booking{}).Where("id = ?", b.ID).Update("reminder_sent_at", now)
	}
}

// markNoShows marca falta em reservas confirmadas sem check-in depois que a aula acabou
func (s *Scheduler) markNoShows(gymID uuid.UUID) {
	cutoff := time.Now().Add(-30 * time.Minute)
	var ids []uuid.UUID
	s.db.Model(&domain.Booking{}).
		Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
		Where("bookings.gym_id = ? AND bookings.status = ? AND class_sessions.ends_at < ?", gymID, domain.BookingConfirmed, cutoff).
		Pluck("bookings.id", &ids)
	if len(ids) > 0 {
		s.db.Model(&domain.Booking{}).Where("id IN ?", ids).Update("status", domain.BookingNoShow)
	}
}

// dueReminders lembra o vencimento do plano 3 dias antes e no dia (uma vez por data, após as 9h)
func (s *Scheduler) dueReminders(gymID uuid.UUID) {
	now := config.Now()
	if now.Hour() < 9 {
		return
	}
	todayKey := now.Format("2006-01-02")
	in3 := now.AddDate(0, 0, 3)

	var students []domain.Student
	s.db.Where("gym_id = ? AND status = ? AND due_day IN ? AND (last_due_notice IS NULL OR last_due_notice <> ?)",
		gymID, domain.StatusAtivo, []int{now.Day(), in3.Day()}, todayKey).Find(&students)

	for _, st := range students {
		title, body := "Seu plano vence em 3 dias", fmt.Sprintf("O vencimento do seu plano é dia %d. Qualquer dúvida, fale com a recepção.", st.DueDay)
		if st.DueDay == now.Day() {
			title, body = "Seu plano vence hoje", "Hoje é o dia do vencimento do seu plano. Regularize na recepção para continuar treinando."
		}
		s.notifier.NotifyStudents(gymID, []uuid.UUID{st.ID}, domain.NotifyDue, title, body, "/aluno/?tab=perfil")
		s.db.Model(&domain.Student{}).Where("id = ?", st.ID).Update("last_due_notice", todayKey)
	}
}
