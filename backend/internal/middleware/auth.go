package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var JWTSecret = []byte("gymsync-secret-key-change-in-production-2026")

type ContextKey string

const (
	UserContextKey ContextKey = "current_user"
	GymContextKey  ContextKey = "current_gym"
)

type CustomClaims struct {
	UserID uuid.UUID   `json:"user_id"`
	GymID  uuid.UUID   `json:"gym_id"`
	Role   domain.Role `json:"role"`
	Email  string      `json:"email"`
	Name   string      `json:"name"`
	jwt.RegisteredClaims
}

// GenerateToken gera token JWT para um usuário
func GenerateToken(user *domain.User) (string, error) {
	claims := CustomClaims{
		UserID: user.ID,
		GymID:  user.GymID,
		Role:   user.Role,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * 7 * time.Hour)), // 7 dias
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// AuthMiddleware intercepta requisições e valida o Bearer Token
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Token de autenticação não fornecido"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"Formato de autorização inválido. Use Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims := &CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"error":"Token inválido ou expirado"}`, http.StatusUnauthorized)
			return
		}

		// Injeta no contexto da requisição para isolamento Multi-Tenant
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		ctx = context.WithValue(ctx, GymContextKey, claims.GymID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRoles restringe o endpoint a determinados papéis
func RequireRoles(allowedRoles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*CustomClaims)
			if !ok {
				http.Error(w, `{"error":"Usuário não autenticado"}`, http.StatusUnauthorized)
				return
			}

			allowed := false
			for _, r := range allowedRoles {
				if claims.Role == r {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, `{"error":"Acesso negado para este perfil de usuário"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
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
