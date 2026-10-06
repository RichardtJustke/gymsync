package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CheckInHandler struct {
	db *gorm.DB
}

func NewCheckInHandler(db *gorm.DB) *CheckInHandler {
	return &CheckInHandler{db: db}
}

type CheckInRequest struct {
	StudentID string `json:"student_id,omitempty"`
}

// DoCheckIn registra a presença do aluno na academia
func (h *CheckInHandler) DoCheckIn(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())

	var targetStudentID uuid.UUID

	if claims.Role == domain.RoleAluno {
		var student domain.Student
		if err := h.db.Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error; err != nil {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Perfil de aluno não localizado"})
			return
		}

		if student.Status == domain.StatusInadimplente {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "Acesso bloqueado: Mensalidade pendente. Procure a recepção ou efetue o pagamento no app."})
			return
		}
		targetStudentID = student.ID
	} else {
		var req CheckInRequest
		json.NewDecoder(r.Body).Decode(&req)
		parsed, err := uuid.Parse(req.StudentID)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "student_id obrigatório e válido"})
			return
		}
		targetStudentID = parsed
	}

	checkIn := domain.CheckIn{
		GymID:     gymID,
		StudentID: targetStudentID,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&checkIn).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao registrar check-in"})
		return
	}

	h.db.Preload("Student.User").First(&checkIn, "id = ?", checkIn.ID)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"message":  "Check-in realizado com sucesso! Bom treino!",
		"check_in": checkIn,
	})
}

// GetTodayCheckIns lista os alunos que treinaram hoje
func (h *CheckInHandler) GetTodayCheckIns(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	startOfDay := time.Now().Truncate(24 * time.Hour)

	var list []domain.CheckIn
	err := h.db.Preload("Student.User").
		Where("gym_id = ? AND created_at >= ?", gymID, startOfDay).
		Order("created_at desc").
		Find(&list).Error

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao buscar presenças de hoje"})
		return
	}

	respondJSON(w, http.StatusOK, list)
}
