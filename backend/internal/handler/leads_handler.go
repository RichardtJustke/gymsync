package handler

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeadsHandler struct {
	db       *gorm.DB
	agenda   *service.AgendaService
	notifier *service.Notifier
	perms    *service.Permissions
	billing  *service.BillingService
	team     *service.TeamService
}

func NewLeadsHandler(db *gorm.DB, agenda *service.AgendaService, notifier *service.Notifier, perms *service.Permissions,
	billing *service.BillingService, team *service.TeamService) *LeadsHandler {
	return &LeadsHandler{db: db, agenda: agenda, notifier: notifier, perms: perms, billing: billing, team: team}
}

// publicGym academia atendida pelas rotas públicas do site (PUBLIC_GYM_ID ou a primeira cadastrada)
func (h *LeadsHandler) publicGym() (*domain.Gym, error) {
	var gym domain.Gym
	q := h.db
	if config.C.PublicGymID != "" {
		q = q.Where("id = ?", config.C.PublicGymID)
	}
	if err := q.Order("created_at asc").First(&gym).Error; err != nil {
		return nil, err
	}
	return &gym, nil
}

/* ------------------------------ Público (site) ------------------------------ */

// PublicGymInfo dados básicos para o site
func (h *LeadsHandler) PublicGymInfo(w http.ResponseWriter, r *http.Request) {
	gym, err := h.publicGym()
	if err != nil {
		respondError(w, http.StatusNotFound, "Academia não configurada")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"name": gym.Name, "phone": gym.Phone, "address": gym.Address, "email": gym.Email})
}

type TrialSlot struct {
	SessionID uuid.UUID `json:"session_id"`
	Modality  string    `json:"modality"`
	Color     string    `json:"color"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Teacher   string    `json:"teacher,omitempty"`
	SpotsLeft int64     `json:"spots_left"`
}

// PublicTrialSessions horários com vaga para aula experimental (próximos 14 dias)
func (h *LeadsHandler) PublicTrialSessions(w http.ResponseWriter, r *http.Request) {
	gym, err := h.publicGym()
	if err != nil {
		respondError(w, http.StatusNotFound, "Academia não configurada")
		return
	}
	now := config.Now()
	to := now.AddDate(0, 0, 14)
	h.agenda.EnsureSessions(gym.ID, now, to)

	var sessions []domain.ClassSession
	h.db.Preload("Modality").Preload("Teacher", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Joins("JOIN modalities ON modalities.id = class_sessions.modality_id").
		Where("class_sessions.gym_id = ? AND class_sessions.status = ? AND class_sessions.starts_at > ? AND class_sessions.starts_at < ?",
			gym.ID, domain.SessionScheduled, now.Add(2*time.Hour).UTC(), to.UTC()).
		Where("modalities.allow_trial = ? AND modalities.active = ?", true, true).
		Order("class_sessions.starts_at asc").Limit(200).Find(&sessions)

	out := []TrialSlot{}
	for _, s := range sessions {
		var taken int64
		h.db.Model(&domain.Booking{}).Where("session_id = ? AND status IN ?", s.ID,
			[]domain.BookingStatus{domain.BookingConfirmed, domain.BookingAttended}).Count(&taken)
		left := int64(s.Capacity) - taken
		if left <= 0 {
			continue
		}
		slot := TrialSlot{SessionID: s.ID, StartsAt: s.StartsAt, EndsAt: s.EndsAt, SpotsLeft: left}
		if s.Modality != nil {
			slot.Modality, slot.Color = s.Modality.Name, s.Modality.Color
		}
		if s.Title != "" {
			slot.Modality = s.Title
		}
		if s.Teacher != nil {
			slot.Teacher = s.Teacher.Name
		}
		out = append(out, slot)
	}
	respondJSON(w, http.StatusOK, out)
}

type PublicLeadRequest struct {
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	Message   string     `json:"message"`
	SessionID *uuid.UUID `json:"session_id"` // horário escolhido para a aula experimental (opcional)
	Consent   bool       `json:"consent"`    // aceite da política de privacidade (LGPD)
	Website   string     `json:"website"`    // honeypot anti-spam: humanos deixam vazio
}

// PublicCreateLead formulário do site: cria lead e, se escolhido, reserva a aula experimental
func (h *LeadsHandler) PublicCreateLead(w http.ResponseWriter, r *http.Request) {
	var req PublicLeadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Website != "" {
		// Bot preencheu o honeypot: responde sucesso sem gravar
		respondJSON(w, http.StatusCreated, map[string]interface{}{"success": true})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	if len(req.Name) < 2 || len(req.Name) > 120 {
		respondError(w, http.StatusBadRequest, "Informe seu nome")
		return
	}
	if req.Email == "" && len(cleanDocument(req.Phone)) < 10 {
		respondError(w, http.StatusBadRequest, "Informe um e-mail ou WhatsApp com DDD")
		return
	}
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			respondError(w, http.StatusBadRequest, "E-mail inválido")
			return
		}
	}
	if !req.Consent {
		respondError(w, http.StatusBadRequest, "É preciso aceitar a política de privacidade")
		return
	}
	if len(req.Message) > 1000 {
		req.Message = req.Message[:1000]
	}

	gym, err := h.publicGym()
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "Cadastro indisponível no momento")
		return
	}

	now := time.Now()
	lead := domain.Lead{
		GymID: gym.ID, Name: req.Name, Email: req.Email, Phone: req.Phone, Message: req.Message,
		Source: "site", Status: domain.LeadNew, ConsentAt: &now,
	}
	var trial *domain.ClassSession
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&lead).Error; err != nil {
			return err
		}
		if req.SessionID != nil {
			s, err := h.agenda.BookTrial(tx, gym.ID, lead.ID, *req.SessionID)
			if err != nil {
				return err
			}
			trial = s
			lead.Status = domain.LeadScheduled
			lead.TrialSessionID = &s.ID
			return tx.Model(&lead).Updates(map[string]interface{}{"status": lead.Status, "trial_session_id": s.ID}).Error
		}
		return nil
	})
	if err != nil {
		respondErr(w, err)
		return
	}

	body := lead.Name + " se cadastrou pelo site."
	resp := map[string]interface{}{"success": true, "message": "Recebemos seu cadastro! Em breve entraremos em contato."}
	if trial != nil {
		when := service.FormatWhen(trial.StartsAt)
		body = lead.Name + " agendou aula experimental: " + trial.Modality.Name + ", " + when + "."
		resp["message"] = "Aula experimental agendada para " + when + ". Te esperamos!"
	}
	h.notifier.NotifyUsers(gym.ID, h.perms.UsersWith(gym.ID, domain.PermLeads), domain.NotifyLead, "Novo lead do site", body, "/admin/?tab=leads")
	respondJSON(w, http.StatusCreated, resp)
}

/* ------------------------------ Gestão ------------------------------ */

func (h *LeadsHandler) ListLeads(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var list []domain.Lead
	q := h.db.Preload("TrialSession.Modality").Preload("AssignedTo", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Select("id", "name") }).
		Where("gym_id = ?", gymID)
	if st := r.URL.Query().Get("status"); st != "" {
		q = q.Where("status = ?", strings.ToUpper(st))
	}
	switch a := r.URL.Query().Get("assigned_to"); a {
	case "":
	case "me":
		q = q.Where("assigned_to_id = ?", middleware.GetUserFromContext(r.Context()).UserID)
	case "none":
		q = q.Where("assigned_to_id IS NULL")
	default:
		if id, err := uuid.Parse(a); err == nil {
			q = q.Where("assigned_to_id = ?", id)
		}
	}
	q.Order("created_at desc").Limit(500).Find(&list)

	type row struct {
		Status string
		N      int64
	}
	var rows []row
	h.db.Model(&domain.Lead{}).Select("status, count(*) as n").Where("gym_id = ?", gymID).Group("status").Scan(&rows)
	counts := map[string]int64{}
	for _, r := range rows {
		counts[r.Status] = r.N
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"data": list, "counts": counts})
}

type LeadRequest struct {
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Phone        string  `json:"phone"`
	Message      string  `json:"message"`
	Status       string  `json:"status"`
	Notes        string  `json:"notes"`
	Source       string  `json:"source"`
	AssignedToID *string `json:"assigned_to_id"` // id da equipe; "" remove; ausente mantém
}

// resolveAssignee valida o responsável (precisa ser da equipe ativa)
func (h *LeadsHandler) resolveAssignee(gymID uuid.UUID, raw *string) (*uuid.UUID, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	if *raw == "" {
		return nil, true, nil
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		return nil, false, &service.BusinessError{Msg: "Responsável inválido"}
	}
	var u domain.User
	if h.db.First(&u, "id = ? AND gym_id = ? AND active = ?", id, gymID, true).Error != nil || !domain.IsStaffRole(u.Role) {
		return nil, false, &service.BusinessError{Msg: "Responsável não encontrado na equipe"}
	}
	return &id, true, nil
}

var leadStatuses = map[domain.LeadStatus]bool{
	domain.LeadNew: true, domain.LeadScheduled: true, domain.LeadAttended: true, domain.LeadEnrolled: true, domain.LeadLost: true,
}

// CreateLead cadastro manual (ex.: interessada que veio na recepção ou pelo Instagram)
func (h *LeadsHandler) CreateLead(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req LeadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusBadRequest, "Nome é obrigatório")
		return
	}
	source := strings.TrimSpace(strings.ToLower(req.Source))
	if source == "" {
		source = "recepcao"
	}
	lead := domain.Lead{
		GymID: gymID, Name: strings.TrimSpace(req.Name), Email: strings.ToLower(strings.TrimSpace(req.Email)),
		Phone: req.Phone, Message: req.Message, Notes: req.Notes, Source: source, Status: domain.LeadNew,
	}
	assignee, set, err := h.resolveAssignee(gymID, req.AssignedToID)
	if err != nil {
		respondErr(w, err)
		return
	}
	claims := middleware.GetUserFromContext(r.Context())
	if set {
		lead.AssignedToID = assignee
	} else if claims.Role == domain.RoleConsultor {
		uid := claims.UserID
		lead.AssignedToID = &uid // consultor(a) que cadastra fica responsável
	}
	if err := h.db.Create(&lead).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "lead.create", "lead", lead.ID, "Cadastrou o lead %s (origem: %s)", lead.Name, lead.Source)
	respondJSON(w, http.StatusCreated, lead)
}

func (h *LeadsHandler) UpdateLead(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var lead domain.Lead
	if err := h.db.First(&lead, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Lead não encontrado")
		return
	}
	var req LeadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	oldStatus, oldAssignee := lead.Status, lead.AssignedToID
	assignee, set, err := h.resolveAssignee(gymID, req.AssignedToID)
	if err != nil {
		respondErr(w, err)
		return
	}
	if set {
		lead.AssignedToID = assignee
	}
	if req.Status != "" {
		st := domain.LeadStatus(strings.ToUpper(req.Status))
		if !leadStatuses[st] {
			respondError(w, http.StatusBadRequest, "Status inválido")
			return
		}
		lead.Status = st
	}
	if strings.TrimSpace(req.Name) != "" {
		lead.Name = strings.TrimSpace(req.Name)
	}
	if req.Email != "" {
		lead.Email = strings.ToLower(strings.TrimSpace(req.Email))
	}
	if req.Phone != "" {
		lead.Phone = req.Phone
	}
	lead.Notes = req.Notes
	if err := h.db.Omit("TrialSession", "AssignedTo").Save(&lead).Error; err != nil {
		respondErr(w, err)
		return
	}
	if oldStatus != lead.Status {
		audit(h.db, r, "lead.status", "lead", lead.ID, "Lead %s: %s → %s", lead.Name, oldStatus, lead.Status)
	}
	if set && !sameID(oldAssignee, lead.AssignedToID) {
		who := "ninguém"
		if lead.AssignedToID != nil {
			var u domain.User
			h.db.Select("name").First(&u, "id = ?", *lead.AssignedToID)
			who = u.Name
			if *lead.AssignedToID != middleware.GetUserFromContext(r.Context()).UserID {
				h.notifier.NotifyUsers(gymID, []uuid.UUID{*lead.AssignedToID}, domain.NotifyLead, "Lead atribuído a você",
					lead.Name+" agora está sob sua responsabilidade.", "/admin/?tab=leads")
			}
		}
		audit(h.db, r, "lead.assign", "lead", lead.ID, "Lead %s atribuído a %s", lead.Name, who)
	}
	respondJSON(w, http.StatusOK, lead)
}

type ConvertLeadRequest struct {
	Email    string     `json:"email"`
	Document string     `json:"document"`
	Phone    string     `json:"phone"`
	PlanID   *uuid.UUID `json:"plan_id"`
	DueDay   int        `json:"due_day"`
	SellerID *uuid.UUID `json:"seller_id"` // padrão: responsável pelo lead, senão quem matricula
}

func sameID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// ConvertLead matricula o lead: cria a conta da aluna e devolve a senha provisória
func (h *LeadsHandler) ConvertLead(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req ConvertLeadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var lead domain.Lead
	if err := h.db.First(&lead, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Lead não encontrado")
		return
	}
	if lead.StudentID != nil {
		respondError(w, http.StatusBadRequest, "Este lead já foi matriculado")
		return
	}

	email := req.Email
	if email == "" {
		email = lead.Email
	}
	phone := req.Phone
	if phone == "" {
		phone = lead.Phone
	}
	var student domain.Student
	var temp string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		user, pass, err := service.CreateUser(tx, gymID, service.NewAccountInput{
			Name: lead.Name, Email: email, Document: req.Document, Phone: phone, Role: domain.RoleAluno,
		})
		if err != nil {
			return err
		}
		temp = pass
		dueDay := req.DueDay
		if dueDay < 1 || dueDay > 31 {
			dueDay = config.Now().Day()
		}
		leadID := lead.ID
		student = domain.Student{GymID: gymID, UserID: user.ID, DueDay: dueDay, Status: domain.StatusAtivo, LeadID: &leadID, ImportSource: "lead-" + lead.Source}
		if err := service.ApplyPlan(tx, gymID, &student, req.PlanID); err != nil {
			return err
		}
		if err := tx.Create(&student).Error; err != nil {
			return err
		}
		student.User = *user
		seller := req.SellerID
		if seller == nil {
			seller = lead.AssignedToID
		}
		if _, err := enroll(tx, h.billing, h.team, r, &student, seller); err != nil {
			return err
		}
		return tx.Model(&lead).Updates(map[string]interface{}{"status": domain.LeadEnrolled, "student_id": student.ID}).Error
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "lead.convert", "student", student.ID, "Matriculou o lead %s (%s)", lead.Name, planLabel(&student))
	login := student.User.Email
	if login == "" {
		login = student.User.Document
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"student": student, "login": login, "temp_password": temp})
}
