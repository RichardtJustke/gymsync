package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkoutsHandler struct {
	db       *gorm.DB
	notifier *service.Notifier
}

func NewWorkoutsHandler(db *gorm.DB, notifier *service.Notifier) *WorkoutsHandler {
	return &WorkoutsHandler{db: db, notifier: notifier}
}

// GetStudentWorkouts retorna as fichas de um determinado aluno
func (h *WorkoutsHandler) GetStudentWorkouts(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de aluno inválido"})
		return
	}

	var plans []domain.WorkoutPlan
	err = h.db.Preload("Teacher").
		Preload("Workouts", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc, letter asc")
		}).
		Preload("Workouts.Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc")
		}).
		Preload("Workouts.Items.Exercise").
		Where("gym_id = ? AND student_id = ?", gymID, studentID).
		Order("created_at desc").
		Find(&plans).Error

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao buscar fichas"})
		return
	}

	respondJSON(w, http.StatusOK, plans)
}

// GetMyWorkouts retorna a ficha ativa do aluno autenticado (Portal do Aluno)
func (h *WorkoutsHandler) GetMyWorkouts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())

	var student domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Perfil de aluno não encontrado"})
		return
	}

	var plan domain.WorkoutPlan
	err := h.db.Preload("Teacher").
		Preload("Workouts", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc, letter asc")
		}).
		Preload("Workouts.Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc")
		}).
		Preload("Workouts.Items.Exercise").
		Where("gym_id = ? AND student_id = ? AND active = true", gymID, student.ID).
		Order("created_at desc").
		First(&plan).Error

	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"has_plan": false,
			"student":  student,
			"plan":     nil,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"has_plan": true,
		"student":  student,
		"plan":     plan,
	})
}

type CreateWorkoutPlanRequest struct {
	Title    string                `json:"title"`
	Goal     string                `json:"goal"`
	Workouts []CreateWorkoutDetail `json:"workouts"`
}

type CreateWorkoutDetail struct {
	Letter      string                     `json:"letter"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	OrderIndex  int                        `json:"order_index"`
	Items       []CreateWorkoutExerciseItem `json:"items"`
}

type CreateWorkoutExerciseItem struct {
	ExerciseID  uuid.UUID `json:"exercise_id"`
	Sets        int       `json:"sets"`
	Reps        string    `json:"reps"`
	LoadKg      float64   `json:"load_kg"`
	RestSeconds int       `json:"rest_seconds"`
	Notes       string    `json:"notes"`
}

// CreateWorkoutPlan cria nova ficha completa para o aluno
func (h *WorkoutsHandler) CreateWorkoutPlan(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de aluno inválido"})
		return
	}

	var req CreateWorkoutPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if req.Title == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Título da ficha é obrigatório"})
		return
	}

	// A aluna precisa ser desta academia
	var student domain.Student
	if err := h.db.Where("id = ? AND gym_id = ?", studentID, gymID).First(&student).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Aluna não encontrada"})
		return
	}

	// Os exercícios precisam ser globais ou desta academia
	exerciseIDs := map[uuid.UUID]bool{}
	for _, wk := range req.Workouts {
		for _, it := range wk.Items {
			exerciseIDs[it.ExerciseID] = true
		}
	}
	if len(exerciseIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(exerciseIDs))
		for id := range exerciseIDs {
			ids = append(ids, id)
		}
		var valid int64
		h.db.Model(&domain.Exercise{}).Where("id IN ? AND (gym_id IS NULL OR gym_id = ?)", ids, gymID).Count(&valid)
		if int(valid) != len(ids) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Exercício inválido na ficha"})
			return
		}
	}

	tx := h.db.Begin()

	// Desativa fichas anteriores
	tx.Model(&domain.WorkoutPlan{}).
		Where("gym_id = ? AND student_id = ? AND active = true", gymID, studentID).
		Update("active", false)

	plan := domain.WorkoutPlan{
		GymID:     gymID,
		StudentID: studentID,
		TeacherID: claims.UserID,
		Title:     req.Title,
		Goal:      req.Goal,
		Active:    true,
		StartDate: time.Now(),
	}
	if err := tx.Create(&plan).Error; err != nil {
		tx.Rollback()
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao criar plano de treino"})
		return
	}

	for wIdx, wReq := range req.Workouts {
		workout := domain.Workout{
			WorkoutPlanID: plan.ID,
			Letter:        wReq.Letter,
			Name:          wReq.Name,
			Description:   wReq.Description,
			OrderIndex:    wIdx + 1,
		}
		if err := tx.Create(&workout).Error; err != nil {
			tx.Rollback()
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao criar divisão de treino"})
			return
		}

		for itemIdx, itReq := range wReq.Items {
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

			item := domain.WorkoutExercise{
				WorkoutID:   workout.ID,
				ExerciseID:  itReq.ExerciseID,
				Sets:        sets,
				Reps:        reps,
				LoadKg:      itReq.LoadKg,
				RestSeconds: rest,
				Notes:       itReq.Notes,
				OrderIndex:  itemIdx + 1,
			}
			if err := tx.Create(&item).Error; err != nil {
				tx.Rollback()
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao vincular exercício ao treino"})
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao salvar ficha"})
		return
	}

	h.notifier.NotifyStudents(gymID, []uuid.UUID{studentID}, domain.NotifyWorkout,
		"Nova ficha de treino 💪", "Sua professora montou a ficha \""+plan.Title+"\". Confira no app!", "/aluno/?tab=treino")

	respondJSON(w, http.StatusCreated, plan)
}

type UpdateLoadRequest struct {
	LoadKg float64 `json:"load_kg"`
}

// UpdateExerciseLoad permite ao aluno registrar progressão de carga
func (h *WorkoutsHandler) UpdateExerciseLoad(w http.ResponseWriter, r *http.Request) {
	itemIdStr := chi.URLParam(r, "itemId")
	itemId, err := uuid.Parse(itemIdStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID do item inválido"})
		return
	}

	var req UpdateLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if req.LoadKg < 0 || req.LoadKg > 1000 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Carga inválida"})
		return
	}

	// Só a dona da ficha (ou a equipe da mesma academia) pode alterar a carga
	claims := middleware.GetUserFromContext(r.Context())
	owner := h.db.Model(&domain.WorkoutExercise{}).
		Joins("JOIN workouts ON workouts.id = workout_exercises.workout_id").
		Joins("JOIN workout_plans ON workout_plans.id = workouts.workout_plan_id").
		Where("workout_exercises.id = ? AND workout_plans.gym_id = ?", itemId, claims.GymID)
	if claims.Role == domain.RoleAluno {
		owner = owner.Joins("JOIN students ON students.id = workout_plans.student_id").Where("students.user_id = ?", claims.UserID)
	}
	var count int64
	owner.Count(&count)
	if count == 0 {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Exercício não encontrado na sua ficha"})
		return
	}

	if err := h.db.Model(&domain.WorkoutExercise{}).Where("id = ?", itemId).Update("load_kg", req.LoadKg).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao atualizar carga"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"load_kg": req.LoadKg,
	})
}
