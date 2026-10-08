package handler

import (
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db    *gorm.DB
	perms *service.Permissions
}

func NewAuthHandler(db *gorm.DB, perms *service.Permissions) *AuthHandler {
	return &AuthHandler{db: db, perms: perms}
}

type LoginRequest struct {
	Identifier string `json:"identifier"` // E-mail ou CPF
	Password   string `json:"password"`
}

type LoginResponse struct {
	Token       string              `json:"token"`
	User        *domain.User        `json:"user"`
	Gym         *domain.Gym         `json:"gym"`
	Permissions []domain.Permission `json:"permissions"`
}

// Hash fixo para comparar mesmo quando o usuário não existe (tempo de resposta uniforme)
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-unika"), bcrypt.DefaultCost)

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ident := strings.TrimSpace(req.Identifier)
	if ident == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Informe e-mail/CPF e senha")
		return
	}

	var user domain.User
	q := h.db.Preload("Gym")
	if strings.Contains(ident, "@") {
		q = q.Where("LOWER(email) = ?", strings.ToLower(ident))
	} else {
		doc := cleanDocument(ident)
		if doc == "" {
			respondError(w, http.StatusUnauthorized, "Credenciais inválidas")
			return
		}
		q = q.Where("document = ?", doc)
	}
	if err := q.First(&user).Error; err != nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		respondError(w, http.StatusUnauthorized, "Credenciais inválidas")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Credenciais inválidas")
		return
	}
	if !user.Active {
		respondError(w, http.StatusForbidden, "Acesso desativado. Fale com a gestão da academia.")
		return
	}

	token, err := middleware.GenerateToken(&user)
	if err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, LoginResponse{Token: token, User: &user, Gym: user.Gym, Permissions: h.perms.List(user.GymID, user.Role)})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var user domain.User
	if err := h.db.Preload("Gym").First(&user, "id = ?", claims.UserID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}

	var student *domain.Student
	if user.Role == domain.RoleAluno {
		var s domain.Student
		if err := h.db.Preload("Plan").Where("user_id = ?", user.ID).First(&s).Error; err == nil {
			student = &s
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user":        user,
		"gym":         user.Gym,
		"student":     student,
		"permissions": h.perms.List(user.GymID, user.Role),
	})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword troca a senha, encerra as outras sessões e devolve um token novo
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req ChangePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.NewPassword) < 8 {
		respondError(w, http.StatusBadRequest, "A nova senha precisa ter pelo menos 8 caracteres")
		return
	}
	var user domain.User
	if err := h.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		respondError(w, http.StatusBadRequest, "Senha atual incorreta")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondErr(w, err)
		return
	}
	user.PasswordHash = string(hash)
	user.FirstAccess = false
	user.TokenVersion++
	if err := h.db.Model(&user).Select("password_hash", "first_access", "token_version").Updates(&user).Error; err != nil {
		respondErr(w, err)
		return
	}
	token, _ := middleware.GenerateToken(&user)
	respondJSON(w, http.StatusOK, map[string]interface{}{"token": token, "user": user})
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount exclusão de conta pela própria aluna (exigência da App Store / Google Play e LGPD).
// Remove dados pessoais (anonimiza) e mantém só registros agregados sem identificação.
func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req DeleteAccountRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var user domain.User
	if err := h.db.First(&user, "id = ?", claims.UserID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Usuário não encontrado")
		return
	}
	if user.Role == domain.RoleAdmin {
		respondError(w, http.StatusBadRequest, "Contas de gestão não podem ser excluídas pelo app")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		respondError(w, http.StatusBadRequest, "Senha incorreta")
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var student domain.Student
		if tx.Where("user_id = ?", user.ID).First(&student).Error == nil {
			tx.Model(&domain.Booking{}).
				Where("student_id = ? AND status IN ?", student.ID, []domain.BookingStatus{domain.BookingConfirmed, domain.BookingWaitlist}).
				Update("status", domain.BookingCanceled)
			if err := tx.Model(&student).Updates(map[string]interface{}{
				"notes": "", "weight": 0, "height": 0, "status": domain.StatusCancelado,
			}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&student).Error; err != nil {
				return err
			}
		}
		tx.Where("user_id = ?", user.ID).Delete(&domain.DeviceToken{})
		tx.Where("user_id = ?", user.ID).Delete(&domain.Notification{})
		if err := tx.Model(&user).Updates(map[string]interface{}{
			"name":          "Conta excluída",
			"email":         "",
			"document":      "",
			"phone":         "",
			"password_hash": "!",
			"token_version": user.TokenVersion + 1,
			"updated_at":    time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}
