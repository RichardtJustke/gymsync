package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgendaHandler struct {
	db       *gorm.DB
	agenda   *service.AgendaService
	notifier *service.Notifier
}

func NewAgendaHandler(db *gorm.DB, agenda *service.AgendaService, notifier *service.Notifier) *AgendaHandler {
	return &AgendaHandler{db: db, agenda: agenda, notifier: notifier}
}

// teacherOK professor(a) ativo(a) da academia
func (h *AgendaHandler) teacherOK(gymID, id uuid.UUID) bool {
	var n int64
	h.db.Model(&domain.User{}).Where("id = ? AND gym_id = ? AND role = ? AND active = ?", id, gymID, domain.RoleProfessor, true).Count(&n)
	return n > 0
}

// teacherConflict aula da mesma pessoa no mesmo horário (ignora a própria aula)
func (h *AgendaHandler) teacherConflict(gymID, teacherID uuid.UUID, start, end time.Time, except uuid.UUID) *domain.ClassSession {
	var s domain.ClassSession
	err := h.db.Preload("Modality").
		Where("gym_id = ? AND teacher_id = ? AND status = ? AND id <> ? AND starts_at < ? AND ends_at > ?",
			gymID, teacherID, domain.SessionScheduled, except, end.UTC(), start.UTC()).
		First(&s).Error
	if err != nil {
		return nil
	}
	return &s
}

func sessionLabel(s *domain.ClassSession) string {
	name := s.Title
	if name == "" && s.Modality != nil {
		name = s.Modality.Name
	}
	if name == "" {
		name = "Aula"
	}
	return name + " de " + service.FormatWhen(s.StartsAt)
}

/* ---------------------- Horários recorrentes ---------------------- */

type ScheduleRequest struct {
	ModalityID  uuid.UUID  `json:"modality_id"`
	TeacherID   *uuid.UUID `json:"teacher_id"`
	Weekday     int        `json:"weekday"`
	StartTime   string     `json:"start_time"`
	DurationMin int        `json:"duration_min"`
	Capacity    int        `json:"capacity"`
	Room        string     `json:"room"`
}

func (h *AgendaHandler) validateSchedule(gymID uuid.UUID, req *ScheduleRequest) error {
	if req.Weekday < 0 || req.Weekday > 6 {
		return &service.BusinessError{Msg: "Dia da semana inválido"}
	}
	if _, _, err := service.ParseHHMM(req.StartTime); err != nil {
		return err
	}
	if req.DurationMin < 15 || req.DurationMin > 240 {
		return &service.BusinessError{Msg: "Duração deve ficar entre 15 e 240 minutos"}
	}
	if req.Capacity < 1 || req.Capacity > 500 {
		return &service.BusinessError{Msg: "Capacidade inválida"}
	}
	var n int64
	h.db.Model(&domain.Modality{}).Where("id = ? AND gym_id = ?", req.ModalityID, gymID).Count(&n)
	if n == 0 {
		return &service.BusinessError{Msg: "Modalidade não encontrada"}
	}
	if req.TeacherID != nil && *req.TeacherID != uuid.Nil {
		if !h.teacherOK(gymID, *req.TeacherID) {
			return &service.BusinessError{Msg: "Professor(a) não encontrado(a) ou desativado(a)"}
		}
	} else {
		req.TeacherID = nil
	}
	return nil
}

func (h *AgendaHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var list []domain.ClassSchedule
	h.db.Preload("Modality").Preload("Teacher").
		Where("gym_id = ? AND active = ?", gymID, true).
		Order("weekday asc, start_time asc").Find(&list)
	respondJSON(w, http.StatusOK, list)
}

func (h *AgendaHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req ScheduleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.validateSchedule(gymID, &req); err != nil {
		respondErr(w, err)
		return
	}
	sc := domain.ClassSchedule{
		GymID: gymID, ModalityID: req.ModalityID, TeacherID: req.TeacherID, Weekday: req.Weekday,
		StartTime: req.StartTime, DurationMin: req.DurationMin, Capacity: req.Capacity, Room: req.Room, Active: true,
	}
	if err := h.db.Create(&sc).Error; err != nil {
		respondErr(w, err)
		return
	}
	h.ensureUpcoming(gymID)
	audit(h.db, r, "schedule.create", "schedule", sc.ID, "Criou turma fixa: %s às %s", weekdayName(sc.Weekday), sc.StartTime)
	respondJSON(w, http.StatusCreated, sc)
}

// UpdateSchedule: mudar dia/horário/duração recria as aulas futuras (avisando quem tinha reserva);
// mudar capacidade/professora/sala só propaga para as aulas futuras.
func (h *AgendaHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var sc domain.ClassSchedule
	if err := h.db.First(&sc, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Horário não encontrado")
		return
	}
	var req ScheduleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.validateSchedule(gymID, &req); err != nil {
		respondErr(w, err)
		return
	}
	timeChanged := sc.Weekday != req.Weekday || sc.StartTime != req.StartTime || sc.DurationMin != req.DurationMin || sc.ModalityID != req.ModalityID

	sc.ModalityID, sc.TeacherID, sc.Weekday, sc.StartTime = req.ModalityID, req.TeacherID, req.Weekday, req.StartTime
	sc.DurationMin, sc.Capacity, sc.Room = req.DurationMin, req.Capacity, req.Room
	if err := h.db.Save(&sc).Error; err != nil {
		respondErr(w, err)
		return
	}

	if timeChanged {
		h.dropFutureSessions(gymID, sc.ID, "O horário desta turma mudou. Confira a nova agenda e reserve de novo.")
	} else {
		h.db.Model(&domain.ClassSession{}).
			Where("schedule_id = ? AND starts_at > ? AND status = ?", sc.ID, time.Now(), domain.SessionScheduled).
			Updates(map[string]interface{}{"capacity": sc.Capacity, "teacher_id": sc.TeacherID, "room": sc.Room})
	}
	h.ensureUpcoming(gymID)
	audit(h.db, r, "schedule.update", "schedule", sc.ID, "Alterou turma fixa: %s às %s", weekdayName(sc.Weekday), sc.StartTime)
	respondJSON(w, http.StatusOK, sc)
}

func (h *AgendaHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	res := h.db.Model(&domain.ClassSchedule{}).Where("id = ? AND gym_id = ?", id, gymID).Update("active", false)
	if res.RowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Horário não encontrado")
		return
	}
	h.db.Where("id = ?", id).Delete(&domain.ClassSchedule{})
	h.dropFutureSessions(gymID, id, "Esta turma foi encerrada.")
	audit(h.db, r, "schedule.delete", "schedule", id, "Encerrou uma turma fixa")
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// dropFutureSessions apaga aulas futuras sem reserva e cancela (com aviso) as que têm reserva
func (h *AgendaHandler) dropFutureSessions(gymID, scheduleID uuid.UUID, reason string) {
	var sessions []domain.ClassSession
	h.db.Where("gym_id = ? AND schedule_id = ? AND starts_at > ? AND status = ?", gymID, scheduleID, time.Now(), domain.SessionScheduled).Find(&sessions)
	for _, s := range sessions {
		var active int64
		h.db.Model(&domain.Booking{}).Where("session_id = ? AND status IN ?", s.ID,
			[]domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist}).Count(&active)
		if active == 0 {
			h.db.Where("session_id = ?", s.ID).Delete(&domain.Booking{})
			h.db.Unscoped().Delete(&s) // libera o índice único para regenerar
			continue
		}
		h.agenda.CancelSession(gymID, s.ID, reason)
		// Desvincula do horário para a nova aula poder ser gerada no mesmo slot
		h.db.Model(&domain.ClassSession{}).Where("id = ?", s.ID).Update("schedule_id", nil)
	}
}

func (h *AgendaHandler) ensureUpcoming(gymID uuid.UUID) {
	var gym domain.Gym
	h.db.First(&gym, "id = ?", gymID)
	days := gym.BookingWindowDays
	if days <= 0 {
		days = 7
	}
	now := config.Now()
	h.agenda.EnsureSessions(gymID, now, now.AddDate(0, 0, days+1))
}

/* ------------------------------ Aulas ------------------------------ */

type SessionView struct {
	domain.ClassSession
	Booked   int64  `json:"booked"`
	Waitlist int64  `json:"waitlist"`
	MyStatus string `json:"my_status,omitempty"`
}

// parseRange lê ?from=AAAA-MM-DD&days=N (padrão: hoje + 7 dias)
func parseRange(r *http.Request, maxDays int) (time.Time, time.Time) {
	loc := config.Loc()
	from := config.StartOfDay(config.Now())
	if f := r.URL.Query().Get("from"); f != "" {
		if t, err := time.ParseInLocation("2006-01-02", f, loc); err == nil {
			from = t
		}
	}
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil {
			days = n
		}
	}
	if days < 1 {
		days = 1
	}
	if days > maxDays {
		days = maxDays
	}
	return from, from.AddDate(0, 0, days)
}

// ListSessions agenda de aulas no período, com vagas ocupadas e a situação da aluna logada
func (h *AgendaHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := claims.GymID
	from, to := parseRange(r, 31)

	if err := h.agenda.EnsureSessions(gymID, from, to); err != nil {
		respondErr(w, err)
		return
	}

	var sessions []domain.ClassSession
	q := h.db.Preload("Modality").Preload("Teacher", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Where("gym_id = ? AND starts_at >= ? AND starts_at < ?", gymID, from.UTC(), to.UTC())
	if mid := r.URL.Query().Get("modality_id"); mid != "" {
		q = q.Where("modality_id = ?", mid)
	}
	switch tid := r.URL.Query().Get("teacher_id"); tid {
	case "":
	case "me":
		q = q.Where("teacher_id = ?", claims.UserID)
	case "none":
		q = q.Where("teacher_id IS NULL")
	default:
		q = q.Where("teacher_id = ?", tid)
	}
	if claims.Role == domain.RoleAluno {
		// Aluna vê só aulas que ainda vão acontecer ou as de hoje
		q = q.Where("starts_at >= ?", config.StartOfDay(config.Now()).UTC())
	}
	q.Order("starts_at asc").Find(&sessions)

	ids := make([]uuid.UUID, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	type countRow struct {
		SessionID uuid.UUID
		Status    domain.BookingStatus
		N         int64
	}
	var counts []countRow
	if len(ids) > 0 {
		h.db.Model(&domain.Booking{}).Select("session_id, status, count(*) as n").
			Where("session_id IN ?", ids).Group("session_id, status").Scan(&counts)
	}
	booked := map[uuid.UUID]int64{}
	waiting := map[uuid.UUID]int64{}
	for _, c := range counts {
		switch c.Status {
		case domain.BookingConfirmed, domain.BookingAttended:
			booked[c.SessionID] += c.N
		case domain.BookingWaitlist:
			waiting[c.SessionID] += c.N
		}
	}

	mine := map[uuid.UUID]string{}
	if claims.Role == domain.RoleAluno && len(ids) > 0 {
		var student domain.Student
		if h.db.Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error == nil {
			var my []domain.Booking
			h.db.Where("student_id = ? AND session_id IN ? AND status <> ?", student.ID, ids, domain.BookingCanceled).Find(&my)
			for _, b := range my {
				mine[b.SessionID] = string(b.Status)
			}
		}
	}

	out := make([]SessionView, len(sessions))
	for i, s := range sessions {
		out[i] = SessionView{ClassSession: s, Booked: booked[s.ID], Waitlist: waiting[s.ID], MyStatus: mine[s.ID]}
	}
	respondJSON(w, http.StatusOK, out)
}

type SessionRequest struct {
	ModalityID  uuid.UUID  `json:"modality_id"`
	TeacherID   *uuid.UUID `json:"teacher_id"`
	Title       string     `json:"title"`
	Date        string     `json:"date"`       // AAAA-MM-DD
	StartTime   string     `json:"start_time"` // HH:MM
	DurationMin int        `json:"duration_min"`
	Capacity    int        `json:"capacity"`
	Room        string     `json:"room"`
	Notes       string     `json:"notes"`
}

// CreateSession aula avulsa / evento / aulão (fora da grade recorrente)
func (h *AgendaHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req SessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	sr := ScheduleRequest{ModalityID: req.ModalityID, TeacherID: req.TeacherID, StartTime: req.StartTime, DurationMin: req.DurationMin, Capacity: req.Capacity}
	if err := h.validateSchedule(gymID, &sr); err != nil {
		respondErr(w, err)
		return
	}
	day, err := time.ParseInLocation("2006-01-02", req.Date, config.Loc())
	if err != nil {
		respondError(w, http.StatusBadRequest, "Data inválida")
		return
	}
	hh, mm, _ := service.ParseHHMM(req.StartTime)
	start := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, config.Loc())
	if start.Before(time.Now()) {
		respondError(w, http.StatusBadRequest, "A aula precisa ser no futuro")
		return
	}
	end := start.Add(time.Duration(req.DurationMin) * time.Minute)
	if sr.TeacherID != nil {
		if c := h.teacherConflict(gymID, *sr.TeacherID, start, end, uuid.Nil); c != nil {
			respondError(w, http.StatusBadRequest, "Professor(a) já tem aula nesse horário: "+sessionLabel(c))
			return
		}
	}
	s := domain.ClassSession{
		GymID: gymID, ModalityID: req.ModalityID, TeacherID: sr.TeacherID, Title: strings.TrimSpace(req.Title),
		StartsAt: start.UTC(), EndsAt: end.UTC(),
		Capacity: req.Capacity, Room: req.Room, Notes: req.Notes, Status: domain.SessionScheduled,
	}
	if err := h.db.Create(&s).Error; err != nil {
		respondErr(w, err)
		return
	}
	h.db.Preload("Modality").First(&s, "id = ?", s.ID)
	audit(h.db, r, "session.create", "session", s.ID, "Criou aula avulsa: %s", sessionLabel(&s))
	respondJSON(w, http.StatusCreated, s)
}

type UpdateSessionRequest struct {
	TeacherID *uuid.UUID `json:"teacher_id"`
	Capacity  *int       `json:"capacity"`
	Room      *string    `json:"room"`
	Notes     *string    `json:"notes"`
}

// UpdateSession ajustes pontuais em uma aula (substituição de professora, capacidade)
func (h *AgendaHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var s domain.ClassSession
	if err := h.db.Preload("Modality").Preload("Teacher").First(&s, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Aula não encontrada")
		return
	}
	var req UpdateSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	oldTeacher := s.Teacher
	teacherChanged := false
	if req.TeacherID != nil {
		if *req.TeacherID == uuid.Nil {
			teacherChanged = s.TeacherID != nil
			s.TeacherID = nil
		} else {
			if !h.teacherOK(gymID, *req.TeacherID) {
				respondError(w, http.StatusBadRequest, "Professor(a) não encontrado(a) ou desativado(a)")
				return
			}
			if c := h.teacherConflict(gymID, *req.TeacherID, s.StartsAt, s.EndsAt, s.ID); c != nil {
				respondError(w, http.StatusBadRequest, "Professor(a) já tem aula nesse horário: "+sessionLabel(c))
				return
			}
			teacherChanged = s.TeacherID == nil || *s.TeacherID != *req.TeacherID
			s.TeacherID = req.TeacherID
		}
	}
	if req.Capacity != nil {
		if *req.Capacity < 1 {
			respondError(w, http.StatusBadRequest, "Capacidade inválida")
			return
		}
		s.Capacity = *req.Capacity
	}
	if req.Room != nil {
		s.Room = *req.Room
	}
	if req.Notes != nil {
		s.Notes = *req.Notes
	}
	if err := h.db.Omit("Modality", "Teacher").Save(&s).Error; err != nil {
		respondErr(w, err)
		return
	}
	if teacherChanged && s.StartsAt.After(time.Now()) {
		h.announceSubstitution(r, &s, oldTeacher)
	} else {
		audit(h.db, r, "session.update", "session", s.ID, "Ajustou a aula %s", sessionLabel(&s))
	}
	respondJSON(w, http.StatusOK, s)
}

// announceSubstitution avisa quem reservou e a nova pessoa responsável pela aula
func (h *AgendaHandler) announceSubstitution(r *http.Request, s *domain.ClassSession, old *domain.User) {
	from := "sem professor(a)"
	if old != nil {
		from = old.Name
	}
	to := "sem professor(a)"
	if s.TeacherID != nil {
		var u domain.User
		if h.db.Select("id", "name").First(&u, "id = ?", *s.TeacherID).Error == nil {
			to = u.Name
			h.notifier.NotifyUsers(s.GymID, []uuid.UUID{u.ID}, domain.NotifyClass, "Você foi escalado(a) para uma aula",
				sessionLabel(s)+".", "/professor/?tab=agenda")
		}
		var ids []uuid.UUID
		h.db.Model(&domain.Booking{}).Where("session_id = ? AND status IN ? AND student_id IS NOT NULL", s.ID,
			[]domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist}).Pluck("student_id", &ids)
		h.notifier.NotifyStudents(s.GymID, ids, domain.NotifyClass, "Troca de professor(a)",
			sessionLabel(s)+" será com "+to+".", "/aluno/?tab=agenda")
	}
	audit(h.db, r, "session.substitute", "session", s.ID, "Substituição na %s: %s → %s", sessionLabel(s), from, to)
}

type CancelSessionRequest struct {
	Reason string `json:"reason"`
}

func (h *AgendaHandler) CancelSession(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req CancelSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	n, err := h.agenda.CancelSession(gymID, id, strings.TrimSpace(req.Reason))
	if err != nil {
		respondErr(w, err)
		return
	}
	var s domain.ClassSession
	h.db.Preload("Modality").First(&s, "id = ?", id)
	audit(h.db, r, "session.cancel", "session", id, "Cancelou a %s (%d aluno(s) avisado(s)). Motivo: %s", sessionLabel(&s), n, strings.TrimSpace(req.Reason))
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "notified": n})
}

// SessionAttendees lista de reservas/presença de uma aula (equipe)
func (h *AgendaHandler) SessionAttendees(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var list []domain.Booking
	h.db.Preload("Student.User").Preload("Lead").
		Where("gym_id = ? AND session_id = ? AND status <> ?", gymID, id, domain.BookingCanceled).
		Order("created_at asc").Find(&list)
	respondJSON(w, http.StatusOK, list)
}

type MarkAttendanceRequest struct {
	Status domain.BookingStatus `json:"status"` // PRESENTE ou FALTA
}

// MarkAttendance professora/recepção marca presença ou falta
func (h *AgendaHandler) MarkAttendance(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req MarkAttendanceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status != domain.BookingAttended && req.Status != domain.BookingNoShow {
		respondError(w, http.StatusBadRequest, "Status deve ser PRESENTE ou FALTA")
		return
	}
	var b domain.Booking
	if err := h.db.First(&b, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Reserva não encontrada")
		return
	}
	if b.Status == domain.BookingWaitlist {
		respondError(w, http.StatusBadRequest, "Reserva ainda está na lista de espera")
		return
	}
	h.db.Model(&b).Update("status", req.Status)
	if b.LeadID != nil && req.Status == domain.BookingAttended {
		h.db.Model(&domain.Lead{}).Where("id = ? AND status = ?", *b.LeadID, domain.LeadScheduled).Update("status", domain.LeadAttended)
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

/* ------------------------- Reservas da aluna ------------------------- */

func (h *AgendaHandler) myStudentID(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	claims := middleware.GetUserFromContext(r.Context())
	var student domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", claims.GymID, claims.UserID).First(&student).Error; err != nil {
		return uuid.Nil, uuid.Nil, &service.BusinessError{Msg: "Perfil de aluna não encontrado"}
	}
	return claims.GymID, student.ID, nil
}

func (h *AgendaHandler) Book(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	gymID, studentID, err := h.myStudentID(r)
	if err != nil {
		respondErr(w, err)
		return
	}
	b, err := h.agenda.Book(gymID, studentID, sessionID)
	if err != nil {
		respondErr(w, err)
		return
	}
	msg := "Vaga reservada! Até lá 💪"
	if b.Status == domain.BookingWaitlist {
		msg = "Aula lotada — você entrou na lista de espera. Avisamos se abrir vaga."
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"booking": b, "message": msg})
}

func (h *AgendaHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	gymID, studentID, err := h.myStudentID(r)
	if err != nil {
		respondErr(w, err)
		return
	}
	if err := h.agenda.Cancel(gymID, studentID, sessionID); err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Reserva cancelada"})
}

// MyBookings próximas reservas da aluna
func (h *AgendaHandler) MyBookings(w http.ResponseWriter, r *http.Request) {
	gymID, studentID, err := h.myStudentID(r)
	if err != nil {
		respondErr(w, err)
		return
	}
	var list []domain.Booking
	h.db.Preload("Session.Modality").Preload("Session.Teacher", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
		Where("bookings.gym_id = ? AND bookings.student_id = ? AND bookings.status IN ? AND class_sessions.ends_at >= ?",
			gymID, studentID, []domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist, domain.BookingAttended}, time.Now()).
		Order("class_sessions.starts_at asc").Find(&list)
	respondJSON(w, http.StatusOK, list)
}

var weekdayNames = []string{"domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"}

func weekdayName(d int) string {
	if d < 0 || d > 6 {
		return "?"
	}
	return weekdayNames[d]
}
