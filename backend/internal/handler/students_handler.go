package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type StudentsHandler struct {
	db              *gorm.DB
	importerService *service.ImporterService
}

func NewStudentsHandler(db *gorm.DB, importer *service.ImporterService) *StudentsHandler {
	return &StudentsHandler{
		db:              db,
		importerService: importer,
	}
}

// ListStudents lista alunos da academia com filtros e paginação
func (h *StudentsHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	search := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := h.db.Preload("User").Where("gym_id = ?", gymID)

	if status != "" {
		query = query.Where("status = ?", strings.ToUpper(status))
	}

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Joins("JOIN users ON users.id = students.user_id").
			Where("LOWER(users.name) LIKE ? OR users.document LIKE ? OR users.phone LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	var total int64
	query.Model(&domain.Student{}).Count(&total)

	var students []domain.Student
	if err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&students).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao listar alunos"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  students,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

type CreateStudentRequest struct {
	Name      string  `json:"name"`
	Document  string  `json:"document"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	PlanName  string  `json:"plan_name"`
	PlanPrice float64 `json:"plan_price"`
	DueDay    int     `json:"due_day"`
	Notes     string  `json:"notes"`
	Weight    float64 `json:"weight"`
	Height    float64 `json:"height"`
}

// CreateStudent cria um aluno individual manualmente
func (h *StudentsHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if req.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Nome do aluno é obrigatório"})
		return
	}

	cleanDoc := cleanDocument(req.Document)
	pass := "123456"
	if len(cleanDoc) >= 6 {
		pass = cleanDoc[:6]
	}
	hashPass, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)

	tx := h.db.Begin()

	user := domain.User{
		GymID:        gymID,
		Name:         req.Name,
		Document:     cleanDoc,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hashPass),
		Role:         domain.RoleAluno,
		FirstAccess:  true,
	}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Falha ao cadastrar usuário do aluno"})
		return
	}

	dueDay := req.DueDay
	if dueDay < 1 || dueDay > 31 {
		dueDay = 10
	}
	planName := req.PlanName
	if planName == "" {
		planName = "Mensal"
	}

	student := domain.Student{
		GymID:     gymID,
		UserID:    user.ID,
		PlanName:  planName,
		PlanPrice: req.PlanPrice,
		DueDay:    dueDay,
		Status:    domain.StatusAtivo,
		Weight:    req.Weight,
		Height:    req.Height,
		Notes:     req.Notes,
	}
	if err := tx.Create(&student).Error; err != nil {
		tx.Rollback()
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao criar perfil de aluno"})
		return
	}

	tx.Commit()

	student.User = user
	respondJSON(w, http.StatusCreated, student)
}

// GetStudent obtém detalhes completos de um aluno
func (h *StudentsHandler) GetStudent(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentIDStr := chi.URLParam(r, "id")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de aluno inválido"})
		return
	}

	var student domain.Student
	err = h.db.Preload("User").
		Where("id = ? AND gym_id = ?", studentID, gymID).
		First(&student).Error

	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Aluno não encontrado"})
		return
	}

	// Busca fichas de treino
	var plans []domain.WorkoutPlan
	h.db.Preload("Workouts.Items.Exercise").
		Where("student_id = ?", studentID).
		Order("created_at desc").
		Find(&plans)

	// Busca faturas
	var invoices []domain.Invoice
	h.db.Where("student_id = ?", studentID).
		Order("due_date desc").
		Find(&invoices)

	// Busca check-ins recentes
	var checkins []domain.CheckIn
	h.db.Where("student_id = ?", studentID).
		Order("created_at desc").
		Limit(10).
		Find(&checkins)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"student":      student,
		"workout_plans": plans,
		"invoices":     invoices,
		"checkins":     checkins,
	})
}

// ImportPreview analisa o arquivo enviado (Excel ou CSV) e retorna o preview de colunas e linhas
func (h *StudentsHandler) ImportPreview(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Erro ao receber arquivo multipart"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Nenhum arquivo enviado com o campo 'file'"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Falha ao ler conteúdo do arquivo"})
		return
	}

	preview, err := h.importerService.ParseFile(header.Filename, fileBytes)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, preview)
}

type ExecuteImportRequest struct {
	Rows []service.ParsedStudentRow `json:"rows"`
}

// ExecuteImport efetiva a importação dos alunos revisados pelo gestor
func (h *StudentsHandler) ExecuteImport(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req ExecuteImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if len(req.Rows) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Nenhuma linha enviada para importação"})
		return
	}

	summary, err := h.importerService.ExecuteImport(gymID, req.Rows)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Erro na importação: %v", err)})
		return
	}

	respondJSON(w, http.StatusOK, summary)
}
