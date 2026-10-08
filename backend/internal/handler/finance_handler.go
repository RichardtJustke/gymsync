package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"
	"gymsync/backend/internal/middleware"
	"gymsync/backend/internal/service"
	"gymsync/backend/pkg/mockbilling"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FinanceHandler mensalidades, recebimentos e inadimplência
type FinanceHandler struct {
	db      *gorm.DB
	billing *service.BillingService
	perms   *service.Permissions
}

func NewFinanceHandler(db *gorm.DB, billing *service.BillingService, perms *service.Permissions) *FinanceHandler {
	return &FinanceHandler{db: db, billing: billing, perms: perms}
}

// ListInvoices ?status=&month=AAAA-MM&q=&student_id=
func (h *FinanceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	qs := r.URL.Query()
	q := h.db.Model(&domain.Invoice{}).Where("invoices.gym_id = ?", gymID)
	if st := strings.ToUpper(qs.Get("status")); st != "" {
		q = q.Where("invoices.status IN ?", strings.Split(st, ","))
	}
	if m := qs.Get("month"); m != "" {
		month, err := service.ParseMonth(m)
		if err != nil {
			respondErr(w, err)
			return
		}
		q = q.Where("invoices.due_date >= ? AND invoices.due_date < ?", month, month.AddDate(0, 1, 0))
	}
	if sid, err := uuid.Parse(qs.Get("student_id")); err == nil {
		q = q.Where("invoices.student_id = ?", sid)
	}
	if s := strings.TrimSpace(qs.Get("q")); s != "" {
		p := "%" + strings.ToLower(s) + "%"
		q = q.Joins("JOIN students ON students.id = invoices.student_id").Joins("JOIN users ON users.id = students.user_id").
			Where("LOWER(users.name) LIKE ? OR users.document LIKE ? OR LOWER(invoices.description) LIKE ?", p, p, p)
	}
	var list []domain.Invoice
	q.Preload("Student.User").Order("invoices.due_date desc").Limit(500).Find(&list)
	respondJSON(w, http.StatusOK, list)
}

func (h *FinanceHandler) Summary(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	month, err := service.ParseMonth(r.URL.Query().Get("month"))
	if err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, h.billing.Summary(gymID, month))
}

type CreateInvoiceRequest struct {
	StudentID   uuid.UUID `json:"student_id"`
	Amount      float64   `json:"amount"`
	DueDate     string    `json:"due_date"` // AAAA-MM-DD
	Description string    `json:"description"`
}

// CreateInvoice cobrança avulsa (taxa, produto, avaliação física...)
func (h *FinanceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req CreateInvoiceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var st domain.Student
	if err := h.db.Preload("User").First(&st, "id = ? AND gym_id = ?", req.StudentID, gymID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Aluno não encontrado")
		return
	}
	due, err := time.ParseInLocation("2006-01-02", req.DueDate, config.Loc())
	if err != nil {
		respondError(w, http.StatusBadRequest, "Data de vencimento inválida")
		return
	}
	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		desc = "Cobrança avulsa"
	}
	inv := domain.Invoice{GymID: gymID, StudentID: st.ID, Kind: domain.InvoiceExtra, Amount: req.Amount, DueDate: due, Description: desc}
	if err := h.billing.Create(h.db, &inv); err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "invoice.create", "invoice", inv.ID, "Lançou cobrança de R$ %s para %s (%s)", service.FormatBRL(inv.Amount), st.User.Name, desc)
	respondJSON(w, http.StatusCreated, inv)
}

type PayInvoiceRequest struct {
	Method   domain.PaymentMethod `json:"method"`
	Amount   *float64             `json:"amount"`
	Discount float64              `json:"discount"`
}

// PayInvoice baixa manual (recepção recebeu em dinheiro, cartão, PIX direto...)
func (h *FinanceHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req PayInvoiceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	uid := claims.UserID
	inv, err := h.billing.RegisterPayment(claims.GymID, id, service.PaymentInput{Method: req.Method, Amount: req.Amount, Discount: req.Discount, ReceivedBy: &uid})
	if err != nil {
		respondErr(w, err)
		return
	}
	extra := ""
	if inv.Discount > 0 {
		extra = fmt.Sprintf(" com desconto de R$ %s", service.FormatBRL(inv.Discount))
	}
	audit(h.db, r, "invoice.pay", "invoice", inv.ID, "Recebeu R$ %s (%s)%s — %s", service.FormatBRL(inv.PaidAmount), inv.PaymentMethod, extra, h.studentName(inv.StudentID))
	respondJSON(w, http.StatusOK, inv)
}

type CancelInvoiceRequest struct {
	Reason string `json:"reason"`
}

func (h *FinanceHandler) CancelInvoice(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req CancelInvoiceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		respondError(w, http.StatusBadRequest, "Informe o motivo do cancelamento")
		return
	}
	inv, err := h.billing.Cancel(gymID, id, req.Reason)
	if err != nil {
		respondErr(w, err)
		return
	}
	audit(h.db, r, "invoice.cancel", "invoice", inv.ID, "Cancelou cobrança de R$ %s — %s. Motivo: %s", service.FormatBRL(inv.Amount), h.studentName(inv.StudentID), req.Reason)
	respondJSON(w, http.StatusOK, inv)
}

type GenerateRequest struct {
	Month string `json:"month"`
}

// Generate gera as mensalidades do mês na hora (o agendador também faz isso todo dia)
func (h *FinanceHandler) Generate(w http.ResponseWriter, r *http.Request) {
	gymID := middleware.GetGymIDFromContext(r.Context())
	var req GenerateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	month, err := service.ParseMonth(req.Month)
	if err != nil {
		respondErr(w, err)
		return
	}
	n, err := h.billing.GenerateMonth(gymID, month)
	if err != nil {
		respondErr(w, err)
		return
	}
	h.billing.RefreshOverdue(gymID)
	if n > 0 {
		audit(h.db, r, "invoice.generate", "invoice", uuid.Nil, "Gerou %d mensalidade(s) de %s", n, month.Format("01/2006"))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"created": n})
}

// MyInvoices cobranças da aluna logada
func (h *FinanceHandler) MyInvoices(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	var st domain.Student
	if err := h.db.Where("gym_id = ? AND user_id = ?", claims.GymID, claims.UserID).First(&st).Error; err != nil {
		respondError(w, http.StatusNotFound, "Perfil de aluna não encontrado")
		return
	}
	var list []domain.Invoice
	h.db.Where("student_id = ? AND status <> ?", st.ID, domain.InvoiceCancel).Order("due_date desc").Limit(36).Find(&list)
	respondJSON(w, http.StatusOK, list)
}

// canSee aluna dona da cobrança ou equipe com acesso ao financeiro
func (h *FinanceHandler) canSee(r *http.Request, inv *domain.Invoice) bool {
	claims := middleware.GetUserFromContext(r.Context())
	if inv.GymID != claims.GymID {
		return false
	}
	if claims.Role == domain.RoleAluno {
		var n int64
		h.db.Model(&domain.Student{}).Where("id = ? AND user_id = ?", inv.StudentID, claims.UserID).Count(&n)
		return n > 0
	}
	return h.perms.Has(claims.GymID, claims.Role, domain.PermFinanceView)
}

// MockPay simula a confirmação do gateway (PIX pago) — só existe enquanto o gateway é o simulado
func (h *FinanceHandler) MockPay(w http.ResponseWriter, r *http.Request) {
	// Pagamento simulado nunca existe em produção: qualquer aluno poderia "pagar" sem pagar
	if h.billing.GatewayName() != "mock" || config.C.IsProduction() {
		respondError(w, http.StatusNotFound, "Indisponível")
		return
	}
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var inv domain.Invoice
	if err := h.db.First(&inv, "id = ?", id).Error; err != nil || !h.canSee(r, &inv) {
		respondError(w, http.StatusNotFound, "Cobrança não encontrada")
		return
	}
	paid, err := h.billing.RegisterPayment(inv.GymID, inv.ID, service.PaymentInput{Method: domain.PayPix})
	if err != nil {
		respondErr(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Pagamento simulado confirmado!", "invoice": paid})
}

// Boleto HTML imprimível do boleto simulado
func (h *FinanceHandler) Boleto(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var inv domain.Invoice
	if err := h.db.Preload("Student.User").First(&inv, "id = ?", id).Error; err != nil || !h.canSee(r, &inv) {
		respondError(w, http.StatusNotFound, "Cobrança não encontrada")
		return
	}
	var gym domain.Gym
	h.db.First(&gym, "id = ?", inv.GymID)
	name, doc := "Aluno", ""
	if inv.Student != nil {
		name, doc = inv.Student.User.Name, inv.Student.User.Document
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'")
	fmt.Fprint(w, mockbilling.GenerateBoletoHTML(&inv, &gym, name, doc))
}

func (h *FinanceHandler) studentName(studentID uuid.UUID) string {
	var st domain.Student
	if h.db.Preload("User").First(&st, "id = ?", studentID).Error != nil {
		return "aluno"
	}
	return st.User.Name
}
