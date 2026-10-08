package service

import (
	"strings"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// NewAccountInput dados para criar uma conta (aluna ou professora)
type NewAccountInput struct {
	Name     string
	Email    string
	Document string
	Phone    string
	Role     domain.Role
}

// CleanDocument mantém só os dígitos do CPF
func CleanDocument(doc string) string {
	var b strings.Builder
	for _, r := range doc {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// EnsureLoginAvailable garante que e-mail e CPF não estão em uso por outra conta (o login é global)
func EnsureLoginAvailable(tx *gorm.DB, email, document string, exceptUserID *uuid.UUID) error {
	check := func(field, value, label string) error {
		if value == "" {
			return nil
		}
		q := tx.Model(&domain.User{}).Where(field+" = ?", value)
		if exceptUserID != nil {
			q = q.Where("id <> ?", *exceptUserID)
		}
		var n int64
		q.Count(&n)
		if n > 0 {
			return bizErr("Já existe uma conta com este %s", label)
		}
		return nil
	}
	if err := check("email", email, "e-mail"); err != nil {
		return err
	}
	return check("document", document, "CPF")
}

// CreateUser cria a conta com senha provisória e primeiro acesso pendente
func CreateUser(tx *gorm.DB, gymID uuid.UUID, in NewAccountInput) (*domain.User, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Document = CleanDocument(in.Document)
	if in.Name == "" {
		return nil, "", bizErr("Nome é obrigatório")
	}
	if in.Email == "" && in.Document == "" {
		return nil, "", bizErr("Informe e-mail ou CPF para o login")
	}
	if err := EnsureLoginAvailable(tx, in.Email, in.Document, nil); err != nil {
		return nil, "", err
	}
	temp := TempPassword()
	hash, err := bcrypt.GenerateFromPassword([]byte(temp), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	user := domain.User{
		GymID:        gymID,
		Name:         in.Name,
		Email:        in.Email,
		Document:     in.Document,
		Phone:        in.Phone,
		PasswordHash: string(hash),
		Role:         in.Role,
		FirstAccess:  true,
	}
	if err := tx.Create(&user).Error; err != nil {
		return nil, "", err
	}
	return &user, temp, nil
}

// ApplyPlan copia nome/preço do plano para a aluna (mantém histórico mesmo se o plano mudar)
func ApplyPlan(tx *gorm.DB, gymID uuid.UUID, student *domain.Student, planID *uuid.UUID) error {
	if planID == nil || *planID == uuid.Nil {
		student.PlanID = nil
		return nil
	}
	var plan domain.Plan
	if err := tx.First(&plan, "id = ? AND gym_id = ?", *planID, gymID).Error; err != nil {
		return bizErr("Plano não encontrado")
	}
	student.PlanID = &plan.ID
	student.PlanName = plan.Name
	student.PlanPrice = plan.Price
	return nil
}
