package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssessmentsHandler struct {
	db *gorm.DB
}

func NewAssessmentsHandler(db *gorm.DB) *AssessmentsHandler {
	return &AssessmentsHandler{db: db}
}

type AssessmentRequest struct {
	Date        *time.Time `json:"date"`
	Weight      float64    `json:"weight"`
	Height      float64    `json:"height"`
	BodyFat     float64    `json:"body_fat"`
	MuscleMass  float64    `json:"muscle_mass"`
	Chest       float64    `json:"chest"`
	Waist       float64    `json:"waist"`
	Abdomen     float64    `json:"abdomen"`
	Hips        float64    `json:"hips"`
	RightArm    float64    `json:"right_arm"`
	LeftArm     float64    `json:"left_arm"`
	RightThigh  float64    `json:"right_thigh"`
	LeftThigh   float64    `json:"left_thigh"`
	RightCalf   float64    `json:"right_calf"`
	LeftCalf    float64    `json:"left_calf"`
	Notes       string     `json:"notes"`
}

// ListStudentAssessments lista as avaliações físicas de um determinado aluno (para professores e staff)
func (h *AssessmentsHandler) ListStudentAssessments(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de aluno inválido"})
		return
	}

	var assessments []domain.PhysicalAssessment
	err = h.db.Preload("Evaluator").
		Where("gym_id = ? AND student_id = ?", gymID, studentID).
		Order("date desc, created_at desc").
		Find(&assessments).Error

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao buscar avaliações físicas"})
		return
	}

	respondJSON(w, http.StatusOK, assessments)
}

// CreateAssessment registra nova avaliação física pelo professor/avaliador
func (h *AssessmentsHandler) CreateAssessment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de aluno inválido"})
		return
	}

	var req AssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	// Aluno precisa ser desta academia
	var student domain.Student
	if err := h.db.Where("id = ? AND gym_id = ?", studentID, gymID).First(&student).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Aluno não encontrado"})
		return
	}

	date := time.Now()
	if req.Date != nil && !req.Date.IsZero() {
		date = *req.Date
	}

	evaluatorID := claims.UserID
	assessment := domain.PhysicalAssessment{
		GymID:       gymID,
		StudentID:   studentID,
		EvaluatorID: &evaluatorID,
		Date:        date,
		Weight:      req.Weight,
		Height:      req.Height,
		BodyFat:     req.BodyFat,
		MuscleMass:  req.MuscleMass,
		Chest:       req.Chest,
		Waist:       req.Waist,
		Abdomen:     req.Abdomen,
		Hips:        req.Hips,
		RightArm:    req.RightArm,
		LeftArm:     req.LeftArm,
		RightThigh:  req.RightThigh,
		LeftThigh:   req.LeftThigh,
		RightCalf:   req.RightCalf,
		LeftCalf:    req.LeftCalf,
		Notes:       req.Notes,
	}

	if err := h.db.Create(&assessment).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao registrar avaliação física"})
		return
	}

	// Atualiza peso e altura no cadastro do aluno
	updates := map[string]interface{}{}
	if req.Weight > 0 {
		updates["weight"] = req.Weight
	}
	if req.Height > 0 {
		updates["height"] = req.Height
	}
	if len(updates) > 0 {
		h.db.Model(&domain.Student{}).Where("id = ?", studentID).Updates(updates)
	}

	h.db.Preload("Evaluator").First(&assessment, assessment.ID)
	respondJSON(w, http.StatusCreated, assessment)
}

// DeleteAssessment exclui uma avaliação física
func (h *AssessmentsHandler) DeleteAssessment(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de aluno inválido"})
		return
	}
	assessmentIDStr := chi.URLParam(r, "assessmentId")
	assessmentID, err := uuid.Parse(assessmentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de avaliação inválido"})
		return
	}

	if err := h.db.Where("id = ? AND student_id = ? AND gym_id = ?", assessmentID, studentID, gymID).Delete(&domain.PhysicalAssessment{}).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao excluir avaliação"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Avaliação excluída com sucesso"})
}

// GetMyAssessments retorna as avaliações e medidas do aluno logado (Portal do Aluno)
func (h *AssessmentsHandler) GetMyAssessments(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())

	var student domain.Student
	if err := h.db.Preload("User").Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Aluno não encontrado"})
		return
	}

	var assessments []domain.PhysicalAssessment
	err := h.db.Preload("Evaluator").
		Where("gym_id = ? AND student_id = ?", gymID, student.ID).
		Order("date desc, created_at desc").
		Find(&assessments).Error

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao buscar avaliações"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"student":     student,
		"assessments": assessments,
	})
}

// CreateMyAssessment permite que o aluno registre sua própria pesagem/medidas
func (h *AssessmentsHandler) CreateMyAssessment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())

	var student domain.Student
	if err := h.db.Preload("User").Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Aluno não encontrado"})
		return
	}

	var req AssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	date := time.Now()
	if req.Date != nil && !req.Date.IsZero() {
		date = *req.Date
	}

	assessment := domain.PhysicalAssessment{
		GymID:      gymID,
		StudentID:  student.ID,
		Date:       date,
		Weight:     req.Weight,
		Height:     req.Height,
		BodyFat:    req.BodyFat,
		MuscleMass: req.MuscleMass,
		Chest:      req.Chest,
		Waist:      req.Waist,
		Abdomen:    req.Abdomen,
		Hips:       req.Hips,
		RightArm:   req.RightArm,
		LeftArm:    req.LeftArm,
		RightThigh: req.RightThigh,
		LeftThigh:  req.LeftThigh,
		RightCalf:  req.RightCalf,
		LeftCalf:   req.LeftCalf,
		Notes:      req.Notes,
	}

	if err := h.db.Create(&assessment).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao registrar medição"})
		return
	}

	// Atualiza peso e altura no perfil
	updates := map[string]interface{}{}
	if req.Weight > 0 {
		updates["weight"] = req.Weight
	}
	if req.Height > 0 {
		updates["height"] = req.Height
	}
	if len(updates) > 0 {
		h.db.Model(&domain.Student{}).Where("id = ?", student.ID).Updates(updates)
	}

	respondJSON(w, http.StatusCreated, assessment)
}
