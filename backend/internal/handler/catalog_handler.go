package handler

import (
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"gorm.io/gorm"
)

// CatalogHandler cadastros da gestão: planos, modalidades, professoras e dados da academia
type CatalogHandler struct {
	db *gorm.DB
}

func NewCatalogHandler(db *gorm.DB) *CatalogHandler {
	return &CatalogHandler{db: db}
}

/* ----------------------------- Planos ----------------------------- */

type PlanRequest struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Price          float64 `json:"price"`
	DurationMonths int     `json:"duration_months"`
	ClassesPerWeek int     `json:"classes_per_week"`
	Active         *bool   `json:"active"`
}

func (req PlanRequest) apply(p *domain.Plan) error {
	if strings.TrimSpace(req.Name) == "" {
		return &service.BusinessError{Msg: "Nome do plano é obrigatório"}
	}
	if req.Price < 0 || req.ClassesPerWeek < 0 {
		return &service.BusinessError{Msg: "Valores do plano inválidos"}
	}
	p.Name = strings.TrimSpace(req.Name)
	p.Description = req.Description
	p.Price = req.Price
	p.DurationMonths = req.DurationMonths
	if p.DurationMonths < 1 {
		p.DurationMonths = 1
	}
	p.ClassesPerWeek = req.ClassesPerWeek
	if req.Active != nil {
		p.Active = *req.Active
	}
	return nil
}

func (h *CatalogHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var plans []domain.Plan
	q := h.db.Where("gym_id = ?", gymID)
	if r.URL.Query().Get("all") != "1" {
		q = q.Where("active = ?", true)
	}
	q.Order("price asc").Find(&plans)

	// Quantas alunas em cada plano
	type row struct {
		PlanID string
		N      int
	}
	var rows []row
	h.db.Model(&domain.Student{}).Select("plan_id, count(*) as n").Where("gym_id = ? AND plan_id IS NOT NULL", gymID).Group("plan_id").Scan(&rows)
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.PlanID] = r.N
	}
	out := make([]map[string]interface{}, 0, len(plans))
	for _, p := range plans {
		out = append(out, map[string]interface{}{"plan": p, "students": counts[p.ID.String()]})
	}
	respondJSON(w, http.StatusOK, out)
}

func (h *CatalogHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req PlanRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	plan := domain.Plan{GymID: gymID, Active: true}
	if err := req.apply(&plan); err != nil {
		respondErr(w, err)
		return
	}
	if err := h.db.Create(&plan).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "plan.create", "plan", plan.ID, "Criou o plano %s (R$ %s)", plan.Name, service.FormatBRL(plan.Price))
	respondJSON(w, http.StatusCreated, plan)
}

func (h *CatalogHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var plan domain.Plan
	if err := h.db.First(&plan, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Plano não encontrado")
		return
	}
	var req PlanRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	oldPrice := plan.Price
	if err := req.apply(&plan); err != nil {
		respondErr(w, err)
		return
	}
	if err := h.db.Save(&plan).Error; err != nil {
		respondErr(w, err)
		return
	}
	if oldPrice != plan.Price {
		audit(h.db, r, "plan.price", "plan", plan.ID, "Mudou o preço do plano %s: R$ %s → R$ %s (vale para novas matrículas)", plan.Name, service.FormatBRL(oldPrice), service.FormatBRL(plan.Price))
	} else {
		audit(h.db, r, "plan.update", "plan", plan.ID, "Atualizou o plano %s", plan.Name)
	}
	respondJSON(w, http.StatusOK, plan)
}

/* --------------------------- Modalidades --------------------------- */

type ModalityRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	AllowTrial  *bool  `json:"allow_trial"`
	Active      *bool  `json:"active"`
}

func (req ModalityRequest) apply(m *domain.Modality) error {
	if strings.TrimSpace(req.Name) == "" {
		return &service.BusinessError{Msg: "Nome da modalidade é obrigatório"}
	}
	m.Name = strings.TrimSpace(req.Name)
	m.Description = req.Description
	if req.Color != "" {
		m.Color = req.Color
	}
	if req.AllowTrial != nil {
		m.AllowTrial = *req.AllowTrial
	}
	if req.Active != nil {
		m.Active = *req.Active
	}
	return nil
}

func (h *CatalogHandler) ListModalities(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var list []domain.Modality
	q := h.db.Where("gym_id = ?", gymID)
	if r.URL.Query().Get("all") != "1" {
		q = q.Where("active = ?", true)
	}
	q.Order("name asc").Find(&list)
	respondJSON(w, http.StatusOK, list)
}

func (h *CatalogHandler) CreateModality(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req ModalityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	m := domain.Modality{GymID: gymID, Active: true, AllowTrial: true}
	if err := req.apply(&m); err != nil {
		respondErr(w, err)
		return
	}
	if err := h.db.Create(&m).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "modality.create", "modality", m.ID, "Criou a modalidade %s", m.Name)
	respondJSON(w, http.StatusCreated, m)
}

func (h *CatalogHandler) UpdateModality(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var m domain.Modality
	if err := h.db.First(&m, "id = ? AND gym_id = ?", id, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Modalidade não encontrada")
		return
	}
	var req ModalityRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.apply(&m); err != nil {
		respondErr(w, err)
		return
	}
	if err := h.db.Save(&m).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "modality.update", "modality", m.ID, "Atualizou a modalidade %s", m.Name)
	respondJSON(w, http.StatusOK, m)
}

/* --------------------------- Professoras --------------------------- */

func (h *CatalogHandler) ListTeachers(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var list []domain.User
	h.db.Where("gym_id = ? AND role = ? AND active = ?", gymID, domain.RoleProfessor, true).Order("name asc").Find(&list)
	respondJSON(w, http.StatusOK, list)
}

type TeacherRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Document string `json:"document"`
	Phone    string `json:"phone"`
}

func (h *CatalogHandler) CreateTeacher(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req TeacherRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		respondError(w, http.StatusBadRequest, "E-mail é obrigatório para professoras")
		return
	}
	var user *domain.User
	var temp string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var err error
		user, temp, err = service.CreateUser(tx, gymID, service.NewAccountInput{
			Name: req.Name, Email: req.Email, Document: req.Document, Phone: req.Phone, Role: domain.RoleProfessor,
		})
		return err
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	h.db.Create(&domain.StaffProfile{GymID: gymID, UserID: user.ID, PayType: domain.PayHourly})
	audit(h.db, r, "staff.create", "staff", user.ID, "Cadastrou %s como Professor(a)", user.Name)
	respondJSON(w, http.StatusCreated, map[string]interface{}{"teacher": user, "temp_password": temp})
}

func (h *CatalogHandler) UpdateTeacher(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var user domain.User
	if err := h.db.First(&user, "id = ? AND gym_id = ? AND role = ?", id, gymID, domain.RoleProfessor).Error; err != nil {
		respondError(w, http.StatusNotFound, "Professora não encontrada")
		return
	}
	var req TeacherRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		user.Name = strings.TrimSpace(req.Name)
	}
	user.Email = strings.ToLower(strings.TrimSpace(req.Email))
	user.Phone = req.Phone
	user.Document = cleanDocument(req.Document)
	if user.Email == "" {
		respondError(w, http.StatusBadRequest, "E-mail é obrigatório para professoras")
		return
	}
	if err := service.EnsureLoginAvailable(h.db, user.Email, user.Document, &user.ID); err != nil {
		respondErr(w, err)
		return
	}
	if err := h.db.Model(&user).Select("name", "email", "phone", "document").Updates(&user).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "staff.update", "staff", user.ID, "Atualizou o cadastro de %s", user.Name)
	respondJSON(w, http.StatusOK, user)
}

// DeleteTeacher desativa a professora: encerra o acesso e tira das turmas futuras
func (h *CatalogHandler) DeleteTeacher(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND gym_id = ? AND role = ?", id, gymID, domain.RoleProfessor).Delete(&domain.User{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return &service.BusinessError{Msg: "Professora não encontrada"}
		}
		tx.Model(&domain.ClassSchedule{}).Where("gym_id = ? AND teacher_id = ?", gymID, id).Update("teacher_id", nil)
		return tx.Model(&domain.ClassSession{}).Where("gym_id = ? AND teacher_id = ? AND starts_at > ?", gymID, id, time.Now()).Update("teacher_id", nil).Error
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "staff.delete", "staff", id, "Removeu um(a) professor(a) da equipe")
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

/* ----------------------------- Academia ----------------------------- */

func (h *CatalogHandler) GetGym(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var gym domain.Gym
	if err := h.db.First(&gym, "id = ?", gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Academia não encontrada")
		return
	}
	respondJSON(w, http.StatusOK, gym)
}

type GymRequest struct {
	Name                string `json:"name"`
	Phone               string `json:"phone"`
	Email               string `json:"email"`
	Address             string `json:"address"`
	BookingWindowDays   int    `json:"booking_window_days"`
	CancelDeadlineHours int    `json:"cancel_deadline_hours"`
	OverdueBlockDays    *int   `json:"overdue_block_days"`
}

func (h *CatalogHandler) UpdateGym(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var gym domain.Gym
	if err := h.db.First(&gym, "id = ?", gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Academia não encontrada")
		return
	}
	var req GymRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		gym.Name = strings.TrimSpace(req.Name)
	}
	gym.Phone, gym.Email, gym.Address = req.Phone, req.Email, req.Address
	if req.BookingWindowDays < 1 || req.BookingWindowDays > 60 || req.CancelDeadlineHours < 0 || req.CancelDeadlineHours > 72 {
		respondError(w, http.StatusBadRequest, "Janela de reserva (1–60 dias) ou prazo de cancelamento (0–72h) inválidos")
		return
	}
	gym.BookingWindowDays, gym.CancelDeadlineHours = req.BookingWindowDays, req.CancelDeadlineHours
	if req.OverdueBlockDays != nil {
		if *req.OverdueBlockDays < 0 || *req.OverdueBlockDays > 90 {
			respondError(w, http.StatusBadRequest, "Bloqueio por atraso deve ficar entre 0 (desligado) e 90 dias")
			return
		}
		gym.OverdueBlockDays = *req.OverdueBlockDays
	}
	if err := h.db.Save(&gym).Error; err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "gym.update", "gym", gym.ID, "Atualizou dados e regras da academia")
	respondJSON(w, http.StatusOK, gym)
}
