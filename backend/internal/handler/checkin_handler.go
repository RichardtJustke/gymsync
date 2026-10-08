package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CheckInHandler struct {
	db    *gorm.DB
	codes *service.CheckinCodes
}

func NewCheckInHandler(db *gorm.DB, codes *service.CheckinCodes) *CheckInHandler {
	return &CheckInHandler{db: db, codes: codes}
}

// Evita check-in duplicado da mesma aluna em sequência
const checkinCooldown = 3 * time.Hour

// GetQRCode código vigente para a tela da recepção (renovado a cada 30s)
func (h *CheckInHandler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	code, expiresIn := h.codes.Current(gymID, time.Now())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"code":       code,
		"qr_payload": service.CheckinQRPrefix + code,
		"expires_in": expiresIn,
	})
}

type StudentCheckInRequest struct {
	Code string `json:"code"` // conteúdo do QR ("unika-checkin:123456") ou só os 6 dígitos
}

// DoCheckIn check-in da própria aluna escaneando o QR da recepção
func (h *CheckInHandler) DoCheckIn(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := claims.GymID

	var req StudentCheckInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	code := strings.TrimPrefix(strings.TrimSpace(req.Code), service.CheckinQRPrefix)
	if !h.codes.Valid(gymID, code, time.Now()) {
		respondError(w, http.StatusBadRequest, "QR code inválido ou expirado. Escaneie novamente o código da recepção.")
		return
	}

	var student domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Perfil de aluna não localizado")
		return
	}
	h.register(w, gymID, student, "QR")
}

type ManualCheckInRequest struct {
	StudentID uuid.UUID `json:"student_id"`
}

// ManualCheckIn recepção registra a entrada da aluna (ex.: celular sem bateria)
func (h *CheckInHandler) ManualCheckIn(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req ManualCheckInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var student domain.Student
	if err := h.db.Preload("User").Where("id = ? AND gym_id = ?", req.StudentID, gymID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Aluna não encontrada")
		return
	}
	if h.register(w, gymID, student, "MANUAL") {
		audit(h.db, r, "checkin.manual", "student", student.ID, "Check-in manual de %s", student.User.Name)
	}
}

// register grava o check-in; devolve true se criou um novo registro
func (h *CheckInHandler) register(w http.ResponseWriter, gymID uuid.UUID, student domain.Student, method string) bool {
	if student.Status == domain.StatusInadimplente || student.Status == domain.StatusCancelado {
		respondError(w, http.StatusForbidden, "Entrada não liberada: matrícula com pendência. Procure a recepção.")
		return false
	}

	now := time.Now()
	var recent domain.CheckIn
	if h.db.Where("gym_id = ? AND student_id = ? AND created_at >= ?", gymID, student.ID, now.Add(-checkinCooldown)).
		Order("created_at desc").First(&recent).Error == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"already":  true,
			"message":  "Seu check-in de hoje já está registrado. Bom treino!",
			"check_in": recent,
		})
		return false
	}

	checkIn := domain.CheckIn{GymID: gymID, StudentID: student.ID, Method: method, CreatedAt: now}
	message := "Check-in realizado! Bom treino!"

	err := h.db.Transaction(func(tx *gorm.DB) error {
		// Se houver reserva de aula começando agora, marca presença nela
		var booking domain.Booking
		err := tx.Preload("Session.Modality").
			Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
			Where("bookings.gym_id = ? AND bookings.student_id = ? AND bookings.status = ?", gymID, student.ID, domain.BookingConfirmed).
			Where("class_sessions.starts_at BETWEEN ? AND ?", now.Add(-45*time.Minute), now.Add(90*time.Minute)).
			Order("class_sessions.starts_at asc").
			First(&booking).Error
		if err == nil {
			if err := tx.Model(&booking).Update("status", domain.BookingAttended).Error; err != nil {
				return err
			}
			checkIn.BookingID = &booking.ID
			if booking.Session != nil && booking.Session.Modality != nil {
				message = "Check-in feito! Presença confirmada na aula de " + booking.Session.Modality.Name + "."
			}
		}
		return tx.Create(&checkIn).Error
	})
	if err != nil {
		respondErr(w, err)
		return false
	}

	h.db.Preload("Student.User").First(&checkIn, "id = ?", checkIn.ID)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"message":  message,
		"check_in": checkIn,
	})
	return true
}

// GetTodayCheckIns lista as alunas que entraram hoje (fuso da academia)
func (h *CheckInHandler) GetTodayCheckIns(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	startOfDay := config.StartOfDay(config.Now())

	var list []domain.CheckIn
	if err := h.db.Preload("Student.User").
		Where("gym_id = ? AND created_at >= ?", gymID, startOfDay).
		Order("created_at desc").
		Find(&list).Error; err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, list)
}

// GetMyCheckIns frequência da aluna no mês (?month=AAAA-MM) + sequência de semanas ativas
func (h *CheckInHandler) GetMyCheckIns(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var student domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", claims.GymID, claims.UserID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Perfil de aluna não localizado")
		return
	}

	loc := config.Loc()
	now := config.Now()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if m := r.URL.Query().Get("month"); m != "" {
		if t, err := time.ParseInLocation("2006-01", m, loc); err == nil {
			month = t
		}
	}

	var monthList []domain.CheckIn
	h.db.Where("student_id = ? AND created_at >= ? AND created_at < ?", student.ID, month, month.AddDate(0, 1, 0)).
		Order("created_at asc").Find(&monthList)
	days := []int{}
	seen := map[int]bool{}
	for _, c := range monthList {
		d := c.CreatedAt.In(loc).Day()
		if !seen[d] {
			seen[d] = true
			days = append(days, d)
		}
	}

	// Sequência: semanas consecutivas (até a atual) com pelo menos 1 check-in
	var recent []time.Time
	h.db.Model(&domain.CheckIn{}).Where("student_id = ? AND created_at >= ?", student.ID, now.AddDate(0, 0, -7*52)).Pluck("created_at", &recent)
	weeks := map[string]bool{}
	for _, t := range recent {
		y, wk := t.In(loc).ISOWeek()
		weeks[weekKey(y, wk)] = true
	}
	streak := 0
	cursor := now
	if y, wk := cursor.ISOWeek(); !weeks[weekKey(y, wk)] {
		cursor = cursor.AddDate(0, 0, -7) // semana atual ainda pode estar em andamento
	}
	for {
		y, wk := cursor.ISOWeek()
		if !weeks[weekKey(y, wk)] {
			break
		}
		streak++
		cursor = cursor.AddDate(0, 0, -7)
	}

	var total30 int64
	h.db.Model(&domain.CheckIn{}).Where("student_id = ? AND created_at >= ?", student.ID, now.AddDate(0, 0, -30)).Count(&total30)

	ws := config.StartOfDay(now).AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
	var thisWeek int64
	h.db.Model(&domain.CheckIn{}).Where("student_id = ? AND created_at >= ?", student.ID, ws).Count(&thisWeek)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"month":        month.Format("2006-01"),
		"days":         days,
		"month_count":  len(days),
		"last30_count": total30,
		"week_count":   thisWeek,
		"streak_weeks": streak,
	})
}

func weekKey(y, w int) string { return fmt.Sprintf("%d-%02d", y, w) }
