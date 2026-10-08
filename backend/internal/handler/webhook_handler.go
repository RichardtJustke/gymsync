package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookHandler recebe confirmações de pagamento do gateway (PIX pago, boleto compensado).
// Independe do provedor: o adaptador real só precisa repassar o evento neste formato.
type WebhookHandler struct {
	db      *gorm.DB
	billing *service.BillingService
	secret  string
}

func NewWebhookHandler(db *gorm.DB, billing *service.BillingService, secret string) *WebhookHandler {
	return &WebhookHandler{db: db, billing: billing, secret: secret}
}

type paymentEvent struct {
	EventID   string   `json:"event_id"`
	Type      string   `json:"type"` // PAYMENT_CONFIRMED
	InvoiceID string   `json:"invoice_id"`
	Method    string   `json:"method"`
	Amount    *float64 `json:"amount"`
}

// ValidSignature confere o X-Signature (HMAC-SHA256 hex do corpo) em tempo constante
func ValidSignature(secret string, body []byte, sig string) bool {
	if secret == "" || sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := mac.Sum(nil)
	got, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(sig), "sha256="))
	return err == nil && hmac.Equal(want, got)
}

// Payments POST /api/webhooks/payments
func (h *WebhookHandler) Payments(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Corpo inválido")
		return
	}
	if !ValidSignature(h.secret, body, r.Header.Get("X-Signature")) {
		respondError(w, http.StatusUnauthorized, "Assinatura inválida")
		return
	}
	var ev paymentEvent
	if json.Unmarshal(body, &ev) != nil || ev.EventID == "" {
		respondError(w, http.StatusBadRequest, "Evento inválido")
		return
	}

	// Idempotência: o 1º insert vence; reentregas do mesmo evento só recebem 200
	rec := domain.WebhookEvent{EventID: ev.EventID, Type: ev.Type}
	res := h.db.Create(&rec)
	if res.Error != nil {
		var n int64
		h.db.Model(&domain.WebhookEvent{}).Where("event_id = ?", ev.EventID).Count(&n)
		if n > 0 {
			respondJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
			return
		}
		respondErr(w, res.Error)
		return
	}

	if ev.Type != "PAYMENT_CONFIRMED" {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	invID, err := uuid.Parse(ev.InvoiceID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invoice_id inválido")
		return
	}
	var inv domain.Invoice
	if err := h.db.Select("id", "gym_id").First(&inv, "id = ?", invID).Error; err != nil {
		respondJSON(w, http.StatusOK, map[string]string{"status": "unknown_invoice"})
		return
	}
	method := domain.PaymentMethod(strings.ToUpper(ev.Method))
	if method == "" {
		method = domain.PayPix
	}
	if _, err := h.billing.RegisterPayment(inv.GymID, inv.ID, service.PaymentInput{Method: method, Amount: ev.Amount}); err != nil {
		if service.IsBusiness(err) { // ex.: já paga/cancelada — não adianta o gateway reenviar
			log.Printf("[WEBHOOK] evento %s ignorado: %v", ev.EventID, err)
			respondJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
			return
		}
		h.db.Delete(&domain.WebhookEvent{}, "event_id = ?", ev.EventID) // falha técnica: permite retentativa
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "processed"})
}
