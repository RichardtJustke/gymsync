package handler

import (
	"fmt"
	"net/http"

	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/pkg/mockbilling"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillingHandler struct {
	db             *gorm.DB
	billingService *mockbilling.MockBillingService
}

func NewBillingHandler(db *gorm.DB, billing *mockbilling.MockBillingService) *BillingHandler {
	return &BillingHandler{
		db:             db,
		billingService: billing,
	}
}

// ListInvoices lista faturas da academia para o gestor
func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	status := r.URL.Query().Get("status")

	query := h.db.Preload("Student.User").Where("gym_id = ?", gymID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var invoices []domain.Invoice
	if err := query.Order("due_date asc").Find(&invoices).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao listar faturas"})
		return
	}

	respondJSON(w, http.StatusOK, invoices)
}

// GetMyInvoices lista faturas do aluno logado
func (h *BillingHandler) GetMyInvoices(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	gymID := middleware.GetGymIDFromContext(r.Context())

	var student domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", gymID, claims.UserID).First(&student).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Perfil de aluno não encontrado"})
		return
	}

	var invoices []domain.Invoice
	if err := h.db.Where("gym_id = ? AND student_id = ?", gymID, student.ID).Order("due_date desc").Find(&invoices).Error; err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Falha ao buscar faturas"})
		return
	}

	respondJSON(w, http.StatusOK, invoices)
}

// RenderBoleto renderiza a visualização para impressão/download do boleto mock
func (h *BillingHandler) RenderBoleto(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		http.Error(w, "ID de fatura inválido", http.StatusBadRequest)
		return
	}

	var invoice domain.Invoice
	if err := h.db.Preload("Student.User").First(&invoice, "id = ?", invoiceID).Error; err != nil {
		http.Error(w, "Fatura não encontrada", http.StatusNotFound)
		return
	}

	var gym domain.Gym
	h.db.First(&gym, "id = ?", invoice.GymID)

	studentName := "Aluno GymSync"
	studentDoc := "000.000.000-00"
	if invoice.Student != nil && invoice.Student.User.Name != "" {
		studentName = invoice.Student.User.Name
		studentDoc = invoice.Student.User.Document
	}

	html := mockbilling.GenerateBoletoHTML(&invoice, &gym, studentName, studentDoc)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

// MockPay simula pagamento em ambiente de testes e atualiza o status instantaneamente
func (h *BillingHandler) MockPay(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := chi.URLParam(r, "id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ID de fatura inválido"})
		return
	}

	invoice, err := h.billingService.SimulatePayment(invoiceID)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Pagamento simulado com sucesso! Status atualizado para PAGO.",
		"invoice": invoice,
	})
}
