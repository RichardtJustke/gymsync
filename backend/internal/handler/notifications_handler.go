package handler

import (
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationsHandler struct {
	db       *gorm.DB
	notifier *service.Notifier
}

func NewNotificationsHandler(db *gorm.DB, notifier *service.Notifier) *NotificationsHandler {
	return &NotificationsHandler{db: db, notifier: notifier}
}

// MyNotifications caixa de entrada (últimas 50) + quantidade não lida
func (h *NotificationsHandler) MyNotifications(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var list []domain.Notification
	h.db.Where("user_id = ?", claims.UserID).Order("created_at desc").Limit(50).Find(&list)
	var unread int64
	h.db.Model(&domain.Notification{}).Where("user_id = ? AND read_at IS NULL", claims.UserID).Count(&unread)
	respondJSON(w, http.StatusOK, map[string]interface{}{"data": list, "unread": unread})
}

type MarkReadRequest struct {
	IDs []uuid.UUID `json:"ids"` // vazio = marcar todas
}

func (h *NotificationsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req MarkReadRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	q := h.db.Model(&domain.Notification{}).Where("user_id = ? AND read_at IS NULL", claims.UserID)
	if len(req.IDs) > 0 {
		q = q.Where("id IN ?", req.IDs)
	}
	q.Update("read_at", time.Now())
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

type DeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"` // android | ios | web
}

// RegisterDevice salva o token de push do aparelho (um token pertence a uma usuária por vez)
func (h *NotificationsHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req DeviceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" || len(req.Token) > 500 {
		respondError(w, http.StatusBadRequest, "Token inválido")
		return
	}
	d := domain.DeviceToken{UserID: claims.UserID, Token: req.Token, Platform: req.Platform}
	err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "token"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "platform", "updated_at"}),
	}).Create(&d).Error
	if err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// UnregisterDevice remove o token no logout (para não notificar quem saiu da conta)
func (h *NotificationsHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var req DeviceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	h.db.Where("user_id = ? AND token = ?", claims.UserID, req.Token).Delete(&domain.DeviceToken{})
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

type AnnouncementRequest struct {
	Kind       domain.NotificationKind `json:"kind"`        // AVISO ou EVENTO
	Title      string                  `json:"title"`
	Body       string                  `json:"body"`
	ModalityID *uuid.UUID              `json:"modality_id"` // opcional: só alunas com reserva nessa modalidade (últimos 60 dias)
}

// CreateAnnouncement gestão envia aviso/evento para as alunas (caixa de entrada + push)
func (h *NotificationsHandler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := claims.GymID
	var req AnnouncementRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title, req.Body = strings.TrimSpace(req.Title), strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" {
		respondError(w, http.StatusBadRequest, "Título e mensagem são obrigatórios")
		return
	}
	if len(req.Title) > 120 || len(req.Body) > 1000 {
		respondError(w, http.StatusBadRequest, "Mensagem muito longa")
		return
	}
	if req.Kind != domain.NotifyEvent {
		req.Kind = domain.NotifyNotice
	}

	var userIDs []uuid.UUID
	audience := "ALL"
	q := h.db.Model(&domain.Student{}).Where("students.gym_id = ? AND students.status = ?", gymID, domain.StatusAtivo)
	if req.ModalityID != nil {
		audience = "MODALITY:" + req.ModalityID.String()
		q = q.Where("students.id IN (?)", h.db.Model(&domain.Booking{}).Select("bookings.student_id").
			Joins("JOIN class_sessions ON class_sessions.id = bookings.session_id").
			Where("class_sessions.modality_id = ? AND bookings.created_at >= ?", *req.ModalityID, time.Now().AddDate(0, 0, -60)))
	}
	q.Pluck("students.user_id", &userIDs)

	sent := h.notifier.NotifyUsers(gymID, userIDs, req.Kind, req.Title, req.Body, "/aluno/?tab=inicio")
	a := domain.Announcement{GymID: gymID, Kind: req.Kind, Title: req.Title, Body: req.Body, Audience: audience, SentCount: sent, CreatedBy: claims.UserID}
	h.db.Create(&a)
	audit(h.db, r, "announcement.send", "announcement", a.ID, "Enviou aviso \"%s\" para %d aluno(s)", a.Title, a.SentCount)
	respondJSON(w, http.StatusCreated, a)
}

func (h *NotificationsHandler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var list []domain.Announcement
	h.db.Where("gym_id = ?", gymID).Order("created_at desc").Limit(100).Find(&list)
	respondJSON(w, http.StatusOK, list)
}
