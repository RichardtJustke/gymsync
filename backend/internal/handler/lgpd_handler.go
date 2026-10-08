package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LGPDHandler struct {
	db *gorm.DB
}

func NewLGPDHandler(db *gorm.DB) *LGPDHandler {
	return &LGPDHandler{db: db}
}

// ExportMyData exporta todos os dados pessoais do titular em formato JSON portátil (LGPD Art. 18, inciso V)
func (h *LGPDHandler) ExportMyData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var u domain.User
	if err := h.db.First(&u, "id = ?", claims.UserID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	exportBundle := map[string]interface{}{
		"export_info": map[string]interface{}{
			"protocol":    fmt.Sprintf("LGPD-EXP-%d", time.Now().Unix()),
			"exported_at": time.Now().Format(time.RFC3339),
			"legal_basis": "Lei Geral de Proteção de Dados (Lei 13.709/2018) - Art. 18",
			"gym_id":      u.GymID,
		},
		"usuario": map[string]interface{}{
			"id":         u.ID,
			"nome":       u.Name,
			"email":      u.Email,
			"documento":  u.Document,
			"telefone":   u.Phone,
			"funcao":     u.Role,
			"criado_em":  u.CreatedAt,
			"atualizado": u.UpdatedAt,
		},
	}

	// Histórico de consentimentos registrados
	var consents []domain.LGPDConsent
	h.db.Where("user_id = ?", u.ID).Order("accepted_at desc").Find(&consents)
	exportBundle["consentimentos"] = consents

	// Se for aluno(a), compila ficha de matrícula, treinos, avaliações, check-ins e financeiro
	if u.Role == domain.RoleAluno {
		var student domain.Student
		if err := h.db.Where("user_id = ?", u.ID).First(&student).Error; err == nil {
			exportBundle["matricula"] = map[string]interface{}{
				"id":         student.ID,
				"status":     student.Status,
				"plano":      student.PlanName,
				"preco":      student.PlanPrice,
				"peso_kg":    student.Weight,
				"altura_cm":  student.Height,
				"dia_venc":   student.DueDay,
				"observacao": student.Notes,
				"criado_em":  student.CreatedAt,
			}

			// Avaliações físicas
			var assessments []domain.PhysicalAssessment
			h.db.Where("student_id = ?", student.ID).Order("date desc").Find(&assessments)
			exportBundle["avaliacoes_fisicas"] = assessments

			// Planos de treino
			var workoutPlans []domain.WorkoutPlan
			h.db.Where("student_id = ?", student.ID).
				Preload("Workouts.Items.Exercise").
				Order("created_at desc").
				Find(&workoutPlans)
			exportBundle["planos_treino"] = workoutPlans

			// Check-ins
			var checkins []domain.CheckIn
			h.db.Where("student_id = ?", student.ID).Order("created_at desc").Find(&checkins)
			exportBundle["checkins"] = checkins

			// Histórico financeiro / faturas
			var invoices []domain.Invoice
			h.db.Where("student_id = ?", student.ID).Order("due_date desc").Find(&invoices)
			exportBundle["faturas"] = invoices
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"portabilidade_lgpd_%s.json\"", u.ID.String()[:8]))
	json.NewEncoder(w).Encode(exportBundle)
}

// GetMyConsent retorna o status do último consentimento registrado
func (h *LGPDHandler) GetMyConsent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var consent domain.LGPDConsent
	err := h.db.Where("user_id = ?", claims.UserID).Order("accepted_at desc").First(&consent).Error
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"accepted":       false,
			"policy_version": "v1.0-2026",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"accepted":         true,
		"consent_id":       consent.ID,
		"policy_version":   consent.PolicyVersion,
		"accepted_at":      consent.AcceptedAt,
		"analytics_opt_in": consent.AnalyticsOptIn,
		"marketing_opt_in": consent.MarketingOptIn,
	})
}

// RegisterConsent registra a aceitação dos termos de privacidade e política de dados
func (h *LGPDHandler) RegisterConsent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var req struct {
		PolicyVersion  string `json:"policy_version"`
		AnalyticsOptIn *bool  `json:"analytics_opt_in"`
		MarketingOptIn *bool  `json:"marketing_opt_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	version := req.PolicyVersion
	if version == "" {
		version = "v1.0-2026"
	}

	analytics := true
	if req.AnalyticsOptIn != nil {
		analytics = *req.AnalyticsOptIn
	}

	marketing := false
	if req.MarketingOptIn != nil {
		marketing = *req.MarketingOptIn
	}

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	if parts := strings.Split(ip, ","); len(parts) > 0 {
		ip = strings.TrimSpace(parts[0])
	}

	consent := domain.LGPDConsent{
		GymID:          claims.GymID,
		UserID:         claims.UserID,
		PolicyVersion:  version,
		AcceptedAt:     time.Now(),
		IPAddress:      ip,
		UserAgent:      r.UserAgent(),
		AnalyticsOptIn: analytics,
		MarketingOptIn: marketing,
	}

	if err := h.db.Create(&consent).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Falha ao registrar consentimento")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"consent_id":  consent.ID,
		"accepted_at": consent.AcceptedAt,
	})
}

// AnonymizeMyData executa a anonimização de dados pessoais (LGPD Art. 18, inciso VI)
func (h *LGPDHandler) AnonymizeMyData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Não autenticado")
		return
	}

	var u domain.User
	if err := h.db.First(&u, "id = ?", claims.UserID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Senha de confirmação é obrigatória")
		return
	}

	// Valida senha
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Senha incorreta")
		return
	}

	tx := h.db.Begin()

	// Substitui dados de identificação por hashes anônimos
	anonSuffix := u.ID.String()[:8]
	anonDoc := fmt.Sprintf("ANON-%s", anonSuffix)
	anonEmail := fmt.Sprintf("anon-%s@privacidade.unika.app", anonSuffix)
	anonName := "Aluno(a) Anonimizado (LGPD)"

	if err := tx.Model(&domain.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"name":          anonName,
		"email":         anonEmail,
		"document":      anonDoc,
		"phone":         "",
		"active":        false,
		"token_version": u.TokenVersion + 1,
	}).Error; err != nil {
		tx.Rollback()
		respondError(w, http.StatusInternalServerError, "Erro ao anonimizar usuário")
		return
	}

	// Se for aluno, marca cancelado e limpa observações médicas/pessoais
	tx.Model(&domain.Student{}).Where("user_id = ?", u.ID).Updates(map[string]interface{}{
		"status":        domain.StatusCancelado,
		"cancel_reason": "Anonimização voluntária solicitada pelo titular (LGPD)",
		"notes":         "[DADOS PESSOAIS REMOVIDOS A PEDIDO DO TITULAR]",
	})

	tx.Commit()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"message": "Seus dados pessoais foram anonimizados conforme LGPD Art. 18.",
	})
}
