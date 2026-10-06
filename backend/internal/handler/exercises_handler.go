package handler

import (
	"encoding/json"
	"net/http"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"gorm.io/gorm"
)

type ExercisesHandler struct {
	db *gorm.DB
}

func NewExercisesHandler(db *gorm.DB) *ExercisesHandler {
	return &ExercisesHandler{db: db}
}

// ListExercises retorna a ampla base de exercícios com filtro opcional por grupo muscular
func (h *ExercisesHandler) ListExercises(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	group := r.URL.Query().Get("group")

	query := h.db.Where("gym_id IS NULL OR gym_id = ?", gymID)
	if group != "" {
		query = query.Where("muscle_group = ?", group)
	}

	var exercises []domain.Exercise
	if err := query.Order("muscle_group asc, name asc").Find(&exercises).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao listar exercícios"})
		return
	}

	// Agrupamento por grupo muscular para facilitar a navegação no frontend
	type MuscleGroupSummary struct {
		Group string `json:"group"`
		Count int    `json:"count"`
	}
	var groups []MuscleGroupSummary
	h.db.Model(&domain.Exercise{}).
		Select("muscle_group as \"group\", count(*) as count").
		Where("gym_id IS NULL OR gym_id = ?", gymID).
		Group("muscle_group").
		Order("muscle_group asc").
		Scan(&groups)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   exercises,
		"groups": groups,
		"total":  len(exercises),
	})
}

type CreateExerciseRequest struct {
	Name        string `json:"name"`
	MuscleGroup string `json:"muscle_group"`
	Equipment   string `json:"equipment"`
	Description string `json:"description"`
	VideoURL    string `json:"video_url"`
}

// CreateExercise adiciona um exercício personalizado para a academia
func (h *ExercisesHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req CreateExerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if req.Name == "" || req.MuscleGroup == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Nome e grupo muscular são obrigatórios"})
		return
	}

	ex := domain.Exercise{
		GymID:       &gymID,
		Name:        req.Name,
		MuscleGroup: req.MuscleGroup,
		Equipment:   req.Equipment,
		Description: req.Description,
		VideoURL:    req.VideoURL,
	}

	if err := h.db.Create(&ex).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao criar exercício"})
		return
	}

	respondJSON(w, http.StatusCreated, ex)
}
