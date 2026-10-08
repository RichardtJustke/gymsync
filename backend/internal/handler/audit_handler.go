package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// audit registra uma ação da equipe no histórico (falha ao gravar não interrompe a operação)
func audit(db *gorm.DB, r *http.Request, action, entity string, entityID uuid.UUID, format string, a ...any) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		return
	}
	uid := claims.UserID
	entry := domain.AuditLog{
		GymID:    claims.GymID,
		UserID:   &uid,
		UserName: claims.Name,
		UserRole: claims.Role,
		Action:   action,
		Entity:   entity,
		Summary:  fmt.Sprintf(format, a...),
		IP:       r.RemoteAddr,
	}
	if entityID != uuid.Nil {
		id := entityID
		entry.EntityID = &id
	}
	if err := db.Create(&entry).Error; err != nil {
		log.Printf("[AUDIT] falha ao registrar %s: %v", action, err)
	}
}

type AuditHandler struct {
	db *gorm.DB
}

func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// ListAudit histórico filtrável: ?entity=&user_id=&entity_id=&q=&from=AAAA-MM-DD&to=AAAA-MM-DD&page=
func (h *AuditHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	qs := r.URL.Query()
	q := h.db.Model(&domain.AuditLog{}).Where("gym_id = ?", gymID)
	if e := qs.Get("entity"); e != "" {
		q = q.Where("entity = ?", e)
	}
	if u, err := uuid.Parse(qs.Get("user_id")); err == nil {
		q = q.Where("user_id = ?", u)
	}
	if e, err := uuid.Parse(qs.Get("entity_id")); err == nil {
		q = q.Where("entity_id = ?", e)
	}
	if s := strings.TrimSpace(qs.Get("q")); s != "" {
		q = q.Where("LOWER(summary) LIKE ?", "%"+strings.ToLower(s)+"%")
	}
	loc := config.Loc()
	if f, err := time.ParseInLocation("2006-01-02", qs.Get("from"), loc); err == nil {
		q = q.Where("created_at >= ?", f)
	}
	if t, err := time.ParseInLocation("2006-01-02", qs.Get("to"), loc); err == nil {
		q = q.Where("created_at < ?", t.AddDate(0, 0, 1))
	}
	page, _ := strconv.Atoi(qs.Get("page"))
	if page < 1 {
		page = 1
	}
	const limit = 50
	var total int64
	q.Count(&total)
	var list []domain.AuditLog
	q.Order("created_at desc").Limit(limit).Offset((page - 1) * limit).Find(&list)
	respondJSON(w, http.StatusOK, map[string]interface{}{"data": list, "total": total, "page": page, "limit": limit})
}
