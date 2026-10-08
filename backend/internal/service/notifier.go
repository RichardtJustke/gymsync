package service

import (
	"context"
	"log"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notifier grava a notificação na caixa de entrada de cada usuária e dispara o push
type Notifier struct {
	db   *gorm.DB
	push PushSender
}

func NewNotifier(db *gorm.DB, push PushSender) *Notifier {
	return &Notifier{db: db, push: push}
}

// NotifyUsers cria notificações para os usuários e envia push em segundo plano. Retorna quantos receberam.
func (n *Notifier) NotifyUsers(gymID uuid.UUID, userIDs []uuid.UUID, kind domain.NotificationKind, title, body, link string) int {
	if len(userIDs) == 0 {
		return 0
	}
	rows := make([]domain.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		rows = append(rows, domain.Notification{GymID: gymID, UserID: uid, Kind: kind, Title: title, Body: body, Link: link})
	}
	if err := n.db.CreateInBatches(&rows, 200).Error; err != nil {
		log.Printf("[NOTIFY] Falha ao gravar notificações: %v", err)
		return 0
	}

	go func() {
		var tokens []string
		n.db.Model(&domain.DeviceToken{}).Where("user_id IN ?", userIDs).Pluck("token", &tokens)
		if len(tokens) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		invalid := n.push.Send(ctx, tokens, PushMessage{Title: title, Body: body, Link: link})
		if len(invalid) > 0 {
			n.db.Where("token IN ?", invalid).Delete(&domain.DeviceToken{})
		}
	}()
	return len(userIDs)
}

// NotifyStudents resolve os usuários das alunas e notifica
func (n *Notifier) NotifyStudents(gymID uuid.UUID, studentIDs []uuid.UUID, kind domain.NotificationKind, title, body, link string) int {
	if len(studentIDs) == 0 {
		return 0
	}
	var userIDs []uuid.UUID
	n.db.Model(&domain.Student{}).Where("gym_id = ? AND id IN ?", gymID, studentIDs).Pluck("user_id", &userIDs)
	return n.NotifyUsers(gymID, userIDs, kind, title, body, link)
}

// NotifyRole notifica todos os usuários de um papel (ex.: gestão recebe aviso de novo lead)
func (n *Notifier) NotifyRole(gymID uuid.UUID, role domain.Role, kind domain.NotificationKind, title, body, link string) int {
	var userIDs []uuid.UUID
	n.db.Model(&domain.User{}).Where("gym_id = ? AND role = ?", gymID, role).Pluck("id", &userIDs)
	return n.NotifyUsers(gymID, userIDs, kind, title, body, link)
}
