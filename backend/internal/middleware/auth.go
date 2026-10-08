package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContextKey string

const (
	UserContextKey ContextKey = "current_user"
	GymContextKey  ContextKey = "current_gym"
)

// Sessão longa (app mobile), mas revogável via TokenVersion
const tokenTTL = 30 * 24 * time.Hour

type CustomClaims struct {
	UserID       uuid.UUID   `json:"user_id"`
	GymID        uuid.UUID   `json:"gym_id"`
	Role         domain.Role `json:"role"`
	Email        string      `json:"email"`
	Name         string      `json:"name"`
	TokenVersion int         `json:"tv"`
	jwt.RegisteredClaims
}

// GenerateToken gera token JWT para um usuário
func GenerateToken(user *domain.User) (string, error) {
	claims := CustomClaims{
		UserID:       user.ID,
		GymID:        user.GymID,
		Role:         user.Role,
		Email:        user.Email,
		Name:         user.Name,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.C.JWTSecret)
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}

// Auth valida o Bearer Token e confere no banco se a sessão ainda é válida
// (usuária não excluída e versão do token igual à atual).
func Auth(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				unauthorized(w, "Token de autenticação não fornecido")
				return
			}

			claims := &CustomClaims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				return config.C.JWTSecret, nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
			if err != nil || !token.Valid {
				unauthorized(w, "Token inválido ou expirado")
				return
			}

			var user domain.User
			if err := db.Select("id", "gym_id", "name", "role", "token_version", "active").First(&user, "id = ?", claims.UserID).Error; err != nil {
				unauthorized(w, "Sessão encerrada")
				return
			}
			if user.TokenVersion != claims.TokenVersion || user.GymID != claims.GymID {
				unauthorized(w, "Sessão encerrada")
				return
			}
			if !user.Active {
				unauthorized(w, "Acesso desativado. Fale com a gestão da academia.")
				return
			}
			// Papel e nome vêm do banco, não do token (mudança de cargo vale na hora)
			claims.Role = user.Role
			claims.Name = user.Name

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			ctx = context.WithValue(ctx, GymContextKey, claims.GymID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles restringe o endpoint a determinados papéis
func RequireRoles(allowedRoles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				unauthorized(w, "Usuário não autenticado")
				return
			}
			for _, role := range allowedRoles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Acesso negado para este perfil de usuário"}`))
		})
	}
}

// RequirePerm libera o endpoint para quem tiver ao menos uma das permissões
func RequirePerm(perms *service.Permissions, required ...domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				unauthorized(w, "Usuário não autenticado")
				return
			}
			if perms.Has(claims.GymID, claims.Role, required...) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Você não tem permissão para esta área. Fale com a gestão."}`))
		})
	}
}

// RequireAlunoOrPerm libera para a aluna (o handler confere se o recurso é dela) ou para a equipe com a permissão
func RequireAlunoOrPerm(perms *service.Permissions, required ...domain.Permission) func(http.Handler) http.Handler {
	staff := RequirePerm(perms, required...)
	return func(next http.Handler) http.Handler {
		guarded := staff(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c := GetUserFromContext(r.Context()); c != nil && c.Role == domain.RoleAluno {
				next.ServeHTTP(w, r)
				return
			}
			guarded.ServeHTTP(w, r)
		})
	}
}

// RequireStaff libera para qualquer pessoa da equipe ativa (não aluno)
func RequireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil || !domain.IsStaffRole(claims.Role) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Acesso restrito à equipe"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extrai claims autenticadas
func GetUserFromContext(ctx context.Context) *CustomClaims {
	claims, ok := ctx.Value(UserContextKey).(*CustomClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetGymIDFromContext extrai o ID da academia (Tenant)
func GetGymIDFromContext(ctx context.Context) uuid.UUID {
	gymID, ok := ctx.Value(GymContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return gymID
}

// SecurityHeaders cabeçalhos básicos de proteção para a API
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		if config.C.IsProduction() {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// LimitBody limita o tamanho do corpo das requisições
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
