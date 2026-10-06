package handler

import (
	"encoding/json"
	"net/http"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type LoginRequest struct {
	Identifier string `json:"identifier"` // E-mail ou CPF
	Password   string `json:"password"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
	Gym   *domain.Gym  `json:"gym"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if req.Identifier == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Identificador e senha são obrigatórios"})
		return
	}

	var user domain.User
	cleanDoc := cleanDocument(req.Identifier)
	err := h.db.Preload("Gym").Where("email = ? OR document = ?", req.Identifier, cleanDoc).First(&user).Error
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Credenciais inválidas"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Credenciais inválidas"})
		return
	}

	token, err := middleware.GenerateToken(&user)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao gerar token de sessão"})
		return
	}

	respondJSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User:  &user,
		Gym:   user.Gym,
	})
}

type RegisterGymRequest struct {
	GymName       string `json:"gym_name"`
	GymDocument   string `json:"gym_document"`
	GymPhone      string `json:"gym_phone"`
	GymAddress    string `json:"gym_address"`
	AdminName     string `json:"admin_name"`
	AdminEmail    string `json:"admin_email"`
	AdminDocument string `json:"admin_document"`
	Password      string `json:"password"`
}

func (h *AuthHandler) RegisterGym(w http.ResponseWriter, r *http.Request) {
	var req RegisterGymRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	if req.GymName == "" || req.AdminEmail == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Campos obrigatórios não preenchidos"})
		return
	}

	// Inicia transação
	tx := h.db.Begin()

	gym := domain.Gym{
		Name:     req.GymName,
		Document: req.GymDocument,
		Phone:    req.GymPhone,
		Address:  req.GymAddress,
		ThemeHex: "#2563eb",
	}
	if err := tx.Create(&gym).Error; err != nil {
		tx.Rollback()
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Falha ao criar academia"})
		return
	}

	hashPass, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	admin := domain.User{
		GymID:        gym.ID,
		Name:         req.AdminName,
		Email:        req.AdminEmail,
		Document:     cleanDocument(req.AdminDocument),
		PasswordHash: string(hashPass),
		Role:         domain.RoleAdmin,
		FirstAccess:  false,
	}
	if err := tx.Create(&admin).Error; err != nil {
		tx.Rollback()
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Falha ao criar usuário gestor"})
		return
	}

	tx.Commit()

	token, _ := middleware.GenerateToken(&admin)
	respondJSON(w, http.StatusCreated, LoginResponse{
		Token: token,
		User:  &admin,
		Gym:   &gym,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Não autorizado"})
		return
	}

	var user domain.User
	if err := h.db.Preload("Gym").First(&user, "id = ?", claims.UserID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Usuário não encontrado"})
		return
	}

	var student *domain.Student
	if user.Role == domain.RoleAluno {
		var s domain.Student
		if err := h.db.Where("user_id = ?", user.ID).First(&s).Error; err == nil {
			student = &s
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user":    user,
		"gym":     user.Gym,
		"student": student,
	})
}

func cleanDocument(doc string) string {
	var clean []rune
	for _, r := range doc {
		if r >= '0' && r <= '9' {
			clean = append(clean, r)
		}
	}
	return string(clean)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
