package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BusinessError erro de regra de negócio, exibido como está para a usuária
type BusinessError struct{ Msg string }

func (e *BusinessError) Error() string { return e.Msg }

func bizErr(format string, a ...any) error { return &BusinessError{Msg: fmt.Sprintf(format, a...)} }

// IsBusiness indica se o erro deve ser devolvido como 4xx com a mensagem original
func IsBusiness(err error) bool {
	var be *BusinessError
	return errors.As(err, &be)
}

type AgendaService struct {
	db       *gorm.DB
	notifier *Notifier
}

func NewAgendaService(db *gorm.DB, notifier *Notifier) *AgendaService {
	return &AgendaService{db: db, notifier: notifier}
}

// ParseHHMM converte "HH:MM" em horas e minutos
func ParseHHMM(s string) (int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, bizErr("Horário inválido, use HH:MM")
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, bizErr("Horário inválido, use HH:MM")
	}
	return h, m, nil
}

// EnsureSessions materializa as aulas dos horários recorrentes entre from e to (a partir de hoje)
func (s *AgendaService) EnsureSessions(gymID uuid.UUID, from, to time.Time) error {
	loc := config.Loc()
	today := config.StartOfDay(config.Now())
	if from.Before(today) {
		from = today
	}
	if !to.After(from) {
		return nil
	}

	var schedules []domain.ClassSchedule
	if err := s.db.Where("gym_id = ? AND active = ?", gymID, true).Find(&schedules).Error; err != nil {
		return err
	}
	if len(schedules) == 0 {
		return nil
	}

	var toCreate []domain.ClassSession
	for d := config.StartOfDay(from); d.Before(to); d = d.AddDate(0, 0, 1) {
		for _, sc := range schedules {
			if int(d.Weekday()) != sc.Weekday {
				continue
			}
			h, m, err := ParseHHMM(sc.StartTime)
			if err != nil {
				continue
			}
			start := time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, loc)
			scID := sc.ID
			toCreate = append(toCreate, domain.ClassSession{
				GymID:      gymID,
				ScheduleID: &scID,
				ModalityID: sc.ModalityID,
				TeacherID:  sc.TeacherID,
				StartsAt:   start.UTC(),
				EndsAt:     start.Add(time.Duration(sc.DurationMin) * time.Minute).UTC(),
				Capacity:   sc.Capacity,
				Room:       sc.Room,
				Status:     domain.SessionScheduled,
			})
		}
	}
	if len(toCreate) == 0 {
		return nil
	}
	// O índice único (schedule_id, starts_at) evita duplicar aulas já geradas
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&toCreate, 100).Error
}

// lockSession trava a linha da aula (Postgres) para a contagem de vagas não sofrer corrida
func lockSession(tx *gorm.DB) *gorm.DB {
	if tx.Dialector.Name() == "postgres" {
		return tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return tx
}

func countTaken(tx *gorm.DB, sessionID uuid.UUID) int64 {
	var n int64
	tx.Model(&domain.Booking{}).
		Where("session_id = ? AND status IN ?", sessionID, []domain.BookingStatus{domain.BookingConfirmed, domain.BookingAttended}).
		Count(&n)
	return n
}

// weekBounds segunda 00:00 até a próxima segunda (fuso da academia)
func weekBounds(t time.Time) (time.Time, time.Time) {
	d := config.StartOfDay(t)
	offset := (int(d.Weekday()) + 6) % 7
	start := d.AddDate(0, 0, -offset)
	return start, start.AddDate(0, 0, 7)
}

// Book reserva vaga para a aluna; se lotada, entra na lista de espera
func (s *AgendaService) Book(gymID, studentID, sessionID uuid.UUID) (*domain.Booking, error) {
	var gym domain.Gym
	if err := s.db.First(&gym, "id = ?", gymID).Error; err != nil {
		return nil, err
	}
	var booking domain.Booking

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session domain.ClassSession
		if err := lockSession(tx).Preload("Modality").First(&session, "id = ? AND gym_id = ?", sessionID, gymID).Error; err != nil {
			return bizErr("Aula não encontrada")
		}
		now := time.Now()
		if session.Status != domain.SessionScheduled {
			return bizErr("Esta aula foi cancelada")
		}
		if !session.StartsAt.After(now) {
			return bizErr("Esta aula já começou")
		}
		window := gym.BookingWindowDays
		if window <= 0 {
			window = 7
		}
		if session.StartsAt.After(now.AddDate(0, 0, window)) {
			return bizErr("As reservas abrem %d dias antes da aula", window)
		}

		var student domain.Student
		if err := tx.Preload("Plan").First(&student, "id = ? AND gym_id = ?", studentID, gymID).Error; err != nil {
			return bizErr("Perfil de aluna não encontrado")
		}
		if student.Status != domain.StatusAtivo {
			return bizErr("Sua matrícula não está ativa. Fale com a recepção.")
		}

		var existing domain.Booking
		found := tx.Where("session_id = ? AND student_id = ?", sessionID, studentID).First(&existing).Error == nil
		if found && existing.Status != domain.BookingCanceled {
			return bizErr("Você já tem reserva nesta aula")
		}

		// Limite de aulas por semana do plano
		if student.Plan != nil && student.Plan.ClassesPerWeek > 0 {
			ws, we := weekBounds(session.StartsAt)
			var used int64
			tx.Model(&domain.Booking{}).
				Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
				Where("bookings.student_id = ? AND bookings.status IN ? AND class_sessions.starts_at >= ? AND class_sessions.starts_at < ?",
					studentID, []domain.BookingStatus{domain.BookingConfirmed, domain.BookingAttended, domain.BookingWaitlist}, ws.UTC(), we.UTC()).
				Count(&used)
			if int(used) >= student.Plan.ClassesPerWeek {
				return bizErr("Seu plano permite %d aula(s) por semana e você já atingiu o limite", student.Plan.ClassesPerWeek)
			}
		}

		status := domain.BookingConfirmed
		if countTaken(tx, sessionID) >= int64(session.Capacity) {
			status = domain.BookingWaitlist
		}

		if found {
			existing.Status = status
			existing.ReminderSentAt = nil
			existing.CreatedAt = time.Now() // volta ao fim da fila de espera
			booking = existing
			return tx.Save(&booking).Error
		}
		sid := studentID
		booking = domain.Booking{GymID: gymID, SessionID: sessionID, StudentID: &sid, Status: status}
		return tx.Create(&booking).Error
	})
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// Cancel cancela a reserva da aluna, respeitando a antecedência mínima, e promove a lista de espera
func (s *AgendaService) Cancel(gymID, studentID, sessionID uuid.UUID) error {
	var gym domain.Gym
	if err := s.db.First(&gym, "id = ?", gymID).Error; err != nil {
		return err
	}
	var promoted *domain.Booking
	var session domain.ClassSession

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockSession(tx).Preload("Modality").First(&session, "id = ? AND gym_id = ?", sessionID, gymID).Error; err != nil {
			return bizErr("Aula não encontrada")
		}
		var b domain.Booking
		if err := tx.Where("session_id = ? AND student_id = ? AND status IN ?", sessionID, studentID,
			[]domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist}).First(&b).Error; err != nil {
			return bizErr("Você não tem reserva ativa nesta aula")
		}
		if b.Status == domain.BookingConfirmed {
			deadline := session.StartsAt.Add(-time.Duration(gym.CancelDeadlineHours) * time.Hour)
			if time.Now().After(deadline) {
				return bizErr("O cancelamento só é possível até %dh antes da aula", gym.CancelDeadlineHours)
			}
		}
		wasConfirmed := b.Status == domain.BookingConfirmed
		b.Status = domain.BookingCanceled
		if err := tx.Save(&b).Error; err != nil {
			return err
		}
		if wasConfirmed {
			var next domain.Booking
			if tx.Where("session_id = ? AND status = ?", sessionID, domain.BookingWaitlist).Order("created_at asc").First(&next).Error == nil {
				next.Status = domain.BookingConfirmed
				if err := tx.Save(&next).Error; err != nil {
					return err
				}
				promoted = &next
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if promoted != nil && promoted.StudentID != nil {
		s.notifier.NotifyStudents(gymID, []uuid.UUID{*promoted.StudentID}, domain.NotifyClass,
			"Vaga confirmada! 🎉",
			fmt.Sprintf("Abriu vaga em %s, %s. Sua reserva saiu da lista de espera.", modalityName(&session), FormatWhen(session.StartsAt)),
			"/aluno/?tab=agenda")
	}
	return nil
}

// CancelSession cancela a aula inteira e avisa as alunas com reserva
func (s *AgendaService) CancelSession(gymID, sessionID uuid.UUID, reason string) (int, error) {
	var session domain.ClassSession
	var studentIDs []uuid.UUID
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Modality").First(&session, "id = ? AND gym_id = ?", sessionID, gymID).Error; err != nil {
			return bizErr("Aula não encontrada")
		}
		if session.Status == domain.SessionCanceled {
			return bizErr("A aula já está cancelada")
		}
		session.Status = domain.SessionCanceled
		if reason != "" {
			session.Notes = reason
		}
		if err := tx.Save(&session).Error; err != nil {
			return err
		}
		active := []domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist}
		tx.Model(&domain.Booking{}).Where("session_id = ? AND status IN ? AND student_id IS NOT NULL", sessionID, active).Pluck("student_id", &studentIDs)
		return tx.Model(&domain.Booking{}).Where("session_id = ? AND status IN ?", sessionID, active).Update("status", domain.BookingCanceled).Error
	})
	if err != nil {
		return 0, err
	}
	body := fmt.Sprintf("A aula de %s em %s foi cancelada.", modalityName(&session), FormatWhen(session.StartsAt))
	if reason != "" {
		body += " Motivo: " + reason
	}
	s.notifier.NotifyStudents(gymID, studentIDs, domain.NotifyClass, "Aula cancelada", body, "/aluno/?tab=agenda")
	return len(studentIDs), nil
}

// BookTrial reserva vaga de aula experimental para um lead
func (s *AgendaService) BookTrial(tx *gorm.DB, gymID, leadID, sessionID uuid.UUID) (*domain.ClassSession, error) {
	var session domain.ClassSession
	if err := lockSession(tx).Preload("Modality").First(&session, "id = ? AND gym_id = ?", sessionID, gymID).Error; err != nil {
		return nil, bizErr("Aula não encontrada")
	}
	if session.Status != domain.SessionScheduled || !session.StartsAt.After(time.Now()) {
		return nil, bizErr("Esta aula não está mais disponível")
	}
	if session.Modality == nil || !session.Modality.AllowTrial {
		return nil, bizErr("Esta aula não aceita aula experimental")
	}
	if countTaken(tx, sessionID) >= int64(session.Capacity) {
		return nil, bizErr("Esta aula está lotada, escolha outro horário")
	}
	lid := leadID
	if err := tx.Create(&domain.Booking{GymID: gymID, SessionID: sessionID, LeadID: &lid, Status: domain.BookingConfirmed}).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func modalityName(s *domain.ClassSession) string {
	if s.Title != "" {
		return s.Title
	}
	if s.Modality != nil {
		return s.Modality.Name
	}
	return "aula"
}

var weekdaysPT = []string{"domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"}

// FormatWhen "quarta 08/10 às 18:30" no fuso da academia
func FormatWhen(t time.Time) string {
	t = t.In(config.Loc())
	return fmt.Sprintf("%s %02d/%02d às %02d:%02d", weekdaysPT[t.Weekday()], t.Day(), int(t.Month()), t.Hour(), t.Minute())
}
