package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type StudentsHandler struct {
	db              *gorm.DB
	importerService *service.ImporterService
	billing         *service.BillingService
	team            *service.TeamService
}

func NewStudentsHandler(db *gorm.DB, importer *service.ImporterService, billing *service.BillingService, team *service.TeamService) *StudentsHandler {
	return &StudentsHandler{db: db, importerService: importer, billing: billing, team: team}
}

// enroll pós-matrícula: primeira mensalidade + venda atribuída (padrão: quem está matriculando)
func enroll(tx *gorm.DB, billing *service.BillingService, team *service.TeamService, r *http.Request, student *domain.Student, sellerID *uuid.UUID) (*domain.Sale, error) {
	claims := middleware.GetUserFromContext(r.Context())
	if _, err := billing.FirstInvoice(tx, student); err != nil {
		return nil, err
	}
	if student.PlanID == nil {
		return nil, nil
	}
	seller := claims.UserID
	if sellerID != nil && *sellerID != uuid.Nil {
		seller = *sellerID
	}
	sid := student.ID
	return team.RegisterSale(tx, student.GymID, service.SaleInput{
		SellerID: seller, StudentID: &sid, PlanID: student.PlanID, Kind: domain.SaleNew, CreatedBy: claims.UserID,
	})
}

// ListStudents lista alunas da academia com filtros e paginação
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

	query := h.db.Model(&domain.Student{}).Where("students.gym_id = ?", gymID)
	if status != "" {
		query = query.Where("students.status = ?", strings.ToUpper(status))
	}
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Joins("JOIN users ON users.id = students.user_id").
			Where("LOWER(users.name) LIKE ? OR users.document LIKE ? OR users.phone LIKE ? OR LOWER(users.email) LIKE ?", pattern, pattern, pattern, pattern)
	}

	var total int64
	query.Count(&total)

	var students []domain.Student
	if err := query.Preload("User").Preload("Plan").Order("students.created_at desc").Limit(limit).Offset(offset).Find(&students).Error; err != nil {
		respondErr(w, err)
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
	Name     string     `json:"name"`
	Document string     `json:"document"`
	Email    string     `json:"email"`
	Phone    string     `json:"phone"`
	PlanID   *uuid.UUID `json:"plan_id"`
	DueDay   int        `json:"due_day"`
	Notes    string     `json:"notes"`
	Weight   float64    `json:"weight"`
	Height   float64    `json:"height"`
	SellerID *uuid.UUID `json:"seller_id"` // quem vendeu (comissão); padrão: quem está cadastrando
}

// CreateStudent cadastra uma aluna e devolve a senha provisória (exibida uma única vez)
func (h *StudentsHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req CreateStudentRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	var student domain.Student
	var temp string
	err := h.db.Transaction(func(tx *gorm.DB) error {
		user, pass, err := service.CreateUser(tx, gymID, service.NewAccountInput{
			Name: req.Name, Email: req.Email, Document: req.Document, Phone: req.Phone, Role: domain.RoleAluno,
		})
		if err != nil {
			return err
		}
		temp = pass
		dueDay := req.DueDay
		if dueDay < 1 || dueDay > 31 {
			dueDay = 10
		}
		student = domain.Student{
			GymID:  gymID,
			UserID: user.ID,
			DueDay: dueDay,
			Status: domain.StatusAtivo,
			Weight: req.Weight,
			Height: req.Height,
			Notes:  req.Notes,
		}
		if err := service.ApplyPlan(tx, gymID, &student, req.PlanID); err != nil {
			return err
		}
		if err := tx.Create(&student).Error; err != nil {
			return err
		}
		student.User = *user
		_, err = enroll(tx, h.billing, h.team, r, &student, req.SellerID)
		return err
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "student.create", "student", student.ID, "Matriculou %s (%s)", student.User.Name, planLabel(&student))

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"student":       student,
		"temp_password": temp,
	})
}

// GetStudent obtém detalhes completos de uma aluna
func (h *StudentsHandler) GetStudent(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}

	var student domain.Student
	if err := h.db.Preload("User").Preload("Plan").Where("id = ? AND gym_id = ?", studentID, gymID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Aluna não encontrada")
		return
	}

	var plans []domain.WorkoutPlan
	h.db.Preload("Workouts.Items.Exercise").
		Where("gym_id = ? AND student_id = ?", gymID, studentID).
		Order("created_at desc").
		Find(&plans)

	var checkins []domain.CheckIn
	h.db.Where("gym_id = ? AND student_id = ?", gymID, studentID).
		Order("created_at desc").
		Limit(10).
		Find(&checkins)

	var bookings []domain.Booking
	h.db.Preload("Session.Modality").
		Where("gym_id = ? AND student_id = ?", gymID, studentID).
		Order("created_at desc").
		Limit(10).
		Find(&bookings)

	var invoices []domain.Invoice
	h.db.Where("gym_id = ? AND student_id = ?", gymID, studentID).Order("due_date desc").Limit(12).Find(&invoices)

	var sales []domain.Sale
	h.db.Preload("Seller", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Select("id", "name") }).
		Where("gym_id = ? AND student_id = ?", gymID, studentID).Order("created_at desc").Limit(10).Find(&sales)

	var last30 int64
	h.db.Model(&domain.CheckIn{}).
		Where("gym_id = ? AND student_id = ? AND created_at >= ?", gymID, studentID, time.Now().AddDate(0, 0, -30)).
		Count(&last30)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"student":         student,
		"workout_plans":   plans,
		"checkins":        checkins,
		"bookings":        bookings,
		"invoices":        invoices,
		"sales":           sales,
		"checkins_last30": last30,
	})
}

type UpdateStudentRequest struct {
	Name     *string    `json:"name"`
	Email    *string    `json:"email"`
	Document *string    `json:"document"`
	Phone    *string    `json:"phone"`
	Status   *string    `json:"status"`
	PlanID   *uuid.UUID `json:"plan_id"`
	DueDay   *int       `json:"due_day"`
	Weight   *float64   `json:"weight"`
	Height   *float64   `json:"height"`
	Notes    *string    `json:"notes"`
	// Motivo ao cancelar a matrícula (relatório de cancelamentos)
	CancelReason *string `json:"cancel_reason"`
}

var validStatuses = map[domain.StudentStatus]bool{
	domain.StatusAtivo: true, domain.StatusInadimplente: true, domain.StatusCancelado: true, domain.StatusPendente: true,
}

// UpdateStudent atualiza dados cadastrais (gestão/professora)
func (h *StudentsHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req UpdateStudentRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	var student domain.Student
	if err := h.db.Preload("User").Where("id = ? AND gym_id = ?", studentID, gymID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Aluna não encontrada")
		return
	}

	oldStatus, oldPlan := student.Status, student.PlanName
	err := h.db.Transaction(func(tx *gorm.DB) error {
		u := &student.User
		if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
			u.Name = strings.TrimSpace(*req.Name)
		}
		if req.Email != nil {
			u.Email = strings.ToLower(strings.TrimSpace(*req.Email))
		}
		if req.Document != nil {
			u.Document = cleanDocument(*req.Document)
		}
		if req.Phone != nil {
			u.Phone = *req.Phone
		}
		if u.Email == "" && u.Document == "" {
			return &service.BusinessError{Msg: "A aluna precisa de e-mail ou CPF para entrar no app"}
		}
		if err := service.EnsureLoginAvailable(tx, u.Email, u.Document, &u.ID); err != nil {
			return err
		}
		if err := tx.Model(u).Select("name", "email", "document", "phone").Updates(u).Error; err != nil {
			return err
		}

		if req.Status != nil {
			st := domain.StudentStatus(strings.ToUpper(*req.Status))
			if !validStatuses[st] {
				return &service.BusinessError{Msg: "Status inválido"}
			}
			if st == domain.StatusCancelado && student.Status != domain.StatusCancelado {
				now := time.Now()
				student.CanceledAt = &now
				student.CancelReason = ""
				if req.CancelReason != nil {
					student.CancelReason = strings.TrimSpace(*req.CancelReason)
				}
				// Libera as vagas reservadas e para de cobrar
				tx.Model(&domain.Booking{}).Where("student_id = ? AND status IN ?", student.ID,
					[]domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist}).Update("status", domain.BookingCanceled)
				h.billing.CancelOpenForStudent(tx, student.ID, "Matrícula cancelada")
			}
			if st != domain.StatusCancelado {
				student.CanceledAt, student.CancelReason = nil, ""
			}
			student.Status = st
		}
		if req.PlanID != nil {
			if err := service.ApplyPlan(tx, gymID, &student, req.PlanID); err != nil {
				return err
			}
		}
		if req.DueDay != nil && *req.DueDay >= 1 && *req.DueDay <= 31 {
			student.DueDay = *req.DueDay
		}
		if req.Weight != nil {
			student.Weight = *req.Weight
		}
		if req.Height != nil {
			student.Height = *req.Height
		}
		if req.Notes != nil {
			student.Notes = *req.Notes
		}
		return tx.Omit("User", "Plan").Save(&student).Error
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	name := student.User.Name
	switch {
	case oldStatus != student.Status && student.Status == domain.StatusCancelado:
		reason := student.CancelReason
		if reason == "" {
			reason = "sem motivo informado"
		}
		audit(h.db, r, "student.cancel", "student", student.ID, "Cancelou a matrícula de %s (%s)", name, reason)
	case oldStatus != student.Status:
		audit(h.db, r, "student.status", "student", student.ID, "Mudou a situação de %s: %s → %s", name, oldStatus, student.Status)
	}
	if oldPlan != student.PlanName {
		audit(h.db, r, "student.plan", "student", student.ID, "Trocou o plano de %s: %s → %s", name, oldPlan, student.PlanName)
	}
	if oldStatus == student.Status && oldPlan == student.PlanName {
		audit(h.db, r, "student.update", "student", student.ID, "Atualizou o cadastro de %s", name)
	}
	respondJSON(w, http.StatusOK, student)
}

func planLabel(s *domain.Student) string {
	if s.PlanName == "" {
		return "sem plano"
	}
	return fmt.Sprintf("%s, R$ %s", s.PlanName, service.FormatBRL(s.PlanPrice))
}

// ResetPassword gera nova senha provisória para a aluna (esqueceu a senha → recepção)
func (h *StudentsHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	studentID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var student domain.Student
	if err := h.db.Preload("User").Where("id = ? AND gym_id = ?", studentID, gymID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Aluna não encontrada")
		return
	}
	temp := service.TempPassword()
	hash, _ := bcrypt.GenerateFromPassword([]byte(temp), bcrypt.DefaultCost)
	if err := h.db.Model(&domain.User{}).Where("id = ?", student.UserID).Updates(map[string]interface{}{
		"password_hash": string(hash),
		"first_access":  true,
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		respondErr(w, err)
		return
	}
	login := student.User.Email
	if login == "" {
		login = student.User.Document
	}
	audit(h.db, r, "student.password", "student", student.ID, "Gerou nova senha provisória para %s", student.User.Name)
	respondJSON(w, http.StatusOK, map[string]string{"login": login, "temp_password": temp})
}

type UpdateMyProfileRequest struct {
	Phone  *string  `json:"phone"`
	Weight *float64 `json:"weight"`
	Height *float64 `json:"height"`
	Notes  *string  `json:"notes"`
}

// UpdateMyProfile a própria aluna atualiza telefone e medidas
func (h *StudentsHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req UpdateMyProfileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var student domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", claims.GymID, claims.UserID).First(&student).Error; err != nil {
		respondError(w, http.StatusNotFound, "Perfil de aluna não encontrado")
		return
	}
	updates := map[string]interface{}{}
	if req.Weight != nil && *req.Weight >= 0 && *req.Weight < 400 {
		updates["weight"] = *req.Weight
	}
	if req.Height != nil && *req.Height >= 0 && *req.Height < 260 {
		updates["height"] = *req.Height
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if len(updates) > 0 {
		h.db.Model(&student).Updates(updates)
	}
	if req.Phone != nil {
		h.db.Model(&domain.User{}).Where("id = ?", claims.UserID).Update("phone", *req.Phone)
	}
	h.db.Preload("User").Preload("Plan").First(&student, "id = ?", student.ID)
	respondJSON(w, http.StatusOK, student)
}

// ImportPreview analisa o arquivo enviado (Excel ou CSV) e retorna o preview de colunas e linhas
func (h *StudentsHandler) ImportPreview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "Erro ao receber arquivo (máx. 16MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Nenhum arquivo enviado com o campo 'file'")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Falha ao ler conteúdo do arquivo")
		return
	}

	preview, err := h.importerService.ParseFile(header.Filename, fileBytes)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, preview)
}

type ExecuteImportRequest struct {
	Rows []service.ParsedStudentRow `json:"rows"`
}

// ExecuteImport efetiva a importação das alunas revisadas pela gestão
func (h *StudentsHandler) ExecuteImport(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req ExecuteImportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Rows) == 0 {
		respondError(w, http.StatusBadRequest, "Nenhuma linha enviada para importação")
		return
	}

	summary, err := h.importerService.ExecuteImport(gymID, req.Rows)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Erro na importação: %v", err))
		return
	}
	audit(h.db, r, "student.import", "student", uuid.Nil, "Importou planilha: %d linha(s) processada(s)", len(req.Rows))

	respondJSON(w, http.StatusOK, summary)
}
