package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TemplatesHandler struct {
	db *gorm.DB
}

func NewTemplatesHandler(db *gorm.DB) *TemplatesHandler {
	return &TemplatesHandler{db: db}
}

type CreateTemplateRequest struct {
	Title       string                      `json:"title"`
	Goal        string                      `json:"goal"`
	Level       string                      `json:"level"`
	Description string                      `json:"description"`
	Workouts    []CreateTemplateDivRequest  `json:"workouts"`
}

type CreateTemplateDivRequest struct {
	Letter      string                       `json:"letter"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	OrderIndex  int                          `json:"order_index"`
	Items       []CreateTemplateItemRequest  `json:"items"`
}

type CreateTemplateItemRequest struct {
	ExerciseID  uuid.UUID `json:"exercise_id"`
	Sets        int       `json:"sets"`
	Reps        string    `json:"reps"`
	LoadKg      float64   `json:"load_kg"`
	RestSeconds int       `json:"rest_seconds"`
	Notes       string    `json:"notes"`
	OrderIndex  int       `json:"order_index"`
}

// ListTemplates retorna todos os modelos disponíveis para a academia (globais + personalizados)
func (h *TemplatesHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	goal := strings.TrimSpace(r.URL.Query().Get("goal"))
	level := strings.TrimSpace(r.URL.Query().Get("level"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	query := h.db.Preload("Workouts", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index asc, letter asc")
	}).Preload("Workouts.Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index asc")
	}).Preload("Workouts.Items.Exercise").
		Where("gym_id IS NULL OR gym_id = ?", gymID)

	if goal != "" && goal != "TODOS" {
		query = query.Where("LOWER(goal) LIKE ?", "%"+strings.ToLower(goal)+"%")
	}
	if level != "" && level != "TODOS" {
		query = query.Where("LOWER(level) = ?", strings.ToLower(level))
	}
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(goal) LIKE ?", like, like, like)
	}

	var templates []domain.WorkoutTemplate
	if err := query.Order("gym_id desc, created_at desc").Find(&templates).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao buscar modelos de treino"})
		return
	}

	respondJSON(w, http.StatusOK, templates)
}

// GetTemplate retorna detalhes de um modelo específico
func (h *TemplatesHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	templateIDStr := chi.URLParam(r, "id")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de modelo inválido"})
		return
	}

	var tmpl domain.WorkoutTemplate
	err = h.db.Preload("Workouts", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index asc, letter asc")
	}).Preload("Workouts.Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_index asc")
	}).Preload("Workouts.Items.Exercise").
		Where("id = ? AND (gym_id IS NULL OR gym_id = ?)", templateID, gymID).
		First(&tmpl).Error

	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Modelo não encontrado"})
		return
	}

	respondJSON(w, http.StatusOK, tmpl)
}

// CreateTemplate cria um novo modelo salvo para a academia
func (h *TemplatesHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Título do modelo é obrigatório"})
		return
	}

	level := req.Level
	if level == "" {
		level = "Intermediário"
	}
	goal := req.Goal
	if goal == "" {
		goal = "Hipertrofia"
	}

	tx := h.db.Begin()

	tmpl := domain.WorkoutTemplate{
		GymID:       &gymID,
		CreatedByID: &claims.UserID,
		Title:       req.Title,
		Goal:        goal,
		Level:       level,
		Description: req.Description,
	}

	if err := tx.Create(&tmpl).Error; err != nil {
		tx.Rollback()
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao salvar modelo"})
		return
	}

	for dIdx, divReq := range req.Workouts {
		div := domain.WorkoutTemplateDiv{
			WorkoutTemplateID: tmpl.ID,
			Letter:            divReq.Letter,
			Name:              divReq.Name,
			Description:       divReq.Description,
			OrderIndex:        dIdx + 1,
		}
		if err := tx.Create(&div).Error; err != nil {
			tx.Rollback()
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao salvar divisões do modelo"})
			return
		}

		for itIdx, itReq := range divReq.Items {
			sets := itReq.Sets
			if sets <= 0 {
				sets = 4
			}
			rest := itReq.RestSeconds
			if rest <= 0 {
				rest = 60
			}
			reps := itReq.Reps
			if reps == "" {
				reps = "10-12"
			}

			item := domain.WorkoutTemplateItem{
				WorkoutTemplateDivID: div.ID,
				ExerciseID:           itReq.ExerciseID,
				Sets:                 sets,
				Reps:                 reps,
				LoadKg:               itReq.LoadKg,
				RestSeconds:          rest,
				Notes:                itReq.Notes,
				OrderIndex:           itIdx + 1,
			}
			if err := tx.Create(&item).Error; err != nil {
				tx.Rollback()
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao salvar itens do modelo"})
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao confirmar criação de modelo"})
		return
	}

	// Carrega modelo completo para retorno
	h.db.Preload("Workouts.Items.Exercise").First(&tmpl, tmpl.ID)
	respondJSON(w, http.StatusCreated, tmpl)
}

// DeleteTemplate remove um modelo da academia (modelos globais não podem ser excluídos)
func (h *TemplatesHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	templateIDStr := chi.URLParam(r, "id")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de modelo inválido"})
		return
	}

	var tmpl domain.WorkoutTemplate
	if err := h.db.Where("id = ? AND gym_id = ?", templateID, gymID).First(&tmpl).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Modelo não encontrado ou não pertence a esta academia"})
		return
	}

	if err := h.db.Delete(&tmpl).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao excluir modelo"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Modelo excluído com sucesso"})
}
