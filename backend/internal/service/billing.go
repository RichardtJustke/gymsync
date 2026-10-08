package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gymsync/backend/internal/config"
	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentGateway emite a cobrança (boleto/PIX). Hoje o app usa um gateway simulado;
// para cobrar de verdade basta implementar esta interface com o provedor escolhido.
type PaymentGateway interface {
	Name() string
	Issue(inv *domain.Invoice, gym *domain.Gym) error
}

type BillingService struct {
	db       *gorm.DB
	gateway  PaymentGateway
	notifier *Notifier
}

func NewBillingService(db *gorm.DB, gateway PaymentGateway, notifier *Notifier) *BillingService {
	return &BillingService{db: db, gateway: gateway, notifier: notifier}
}

func (b *BillingService) GatewayName() string { return b.gateway.Name() }

// MonthRef competência AAAA-MM no fuso da academia
func MonthRef(t time.Time) string { return t.In(config.Loc()).Format("2006-01") }

// ParseMonth lê AAAA-MM (vazio = mês atual) e devolve o 1º dia às 00:00 no fuso da academia
func ParseMonth(s string) (time.Time, error) {
	loc := config.Loc()
	if s == "" {
		now := config.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc), nil
	}
	t, err := time.ParseInLocation("2006-01", s, loc)
	if err != nil {
		return time.Time{}, bizErr("Mês inválido, use AAAA-MM")
	}
	return t, nil
}

// DueDateFor vencimento do mês respeitando meses curtos (dia 31 em fevereiro vira o último dia)
func DueDateFor(month time.Time, dueDay int) time.Time {
	if dueDay < 1 {
		dueDay = 10
	}
	last := month.AddDate(0, 1, -1).Day()
	if dueDay > last {
		dueDay = last
	}
	return time.Date(month.Year(), month.Month(), dueDay, 0, 0, 0, 0, month.Location())
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func (b *BillingService) gym(tx *gorm.DB, gymID uuid.UUID) *domain.Gym {
	var g domain.Gym
	if tx.First(&g, "id = ?", gymID).Error != nil {
		return nil
	}
	return &g
}

// Create grava uma cobrança e pede ao gateway os dados de pagamento
func (b *BillingService) Create(tx *gorm.DB, inv *domain.Invoice) error {
	if inv.Amount <= 0 {
		return bizErr("Valor da cobrança deve ser maior que zero")
	}
	if inv.Status == "" {
		inv.Status = domain.InvoicePendente
	}
	if inv.Kind == "" {
		inv.Kind = domain.InvoiceExtra
	}
	inv.Amount = round2(inv.Amount)
	if err := b.gateway.Issue(inv, b.gym(tx, inv.GymID)); err != nil {
		return fmt.Errorf("gateway %s: %w", b.gateway.Name(), err)
	}
	return tx.Create(inv).Error
}

// FirstInvoice cobra a primeira mensalidade na matrícula (vence no dia). Assim a geração mensal pula este mês.
func (b *BillingService) FirstInvoice(tx *gorm.DB, student *domain.Student) (*domain.Invoice, error) {
	if student.PlanPrice <= 0 {
		return nil, nil
	}
	now := config.Now()
	inv := domain.Invoice{
		GymID: student.GymID, StudentID: student.ID, Kind: domain.InvoiceMonthly, Reference: MonthRef(now),
		Amount: student.PlanPrice, DueDate: config.StartOfDay(now),
		Description: fmt.Sprintf("Matrícula — %s (%s)", student.PlanName, now.Format("01/2006")),
	}
	if err := b.Create(tx, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// GenerateMonth cria as mensalidades do mês para quem tem plano pago e ainda não tem cobrança daquela competência
func (b *BillingService) GenerateMonth(gymID uuid.UUID, month time.Time) (int, error) {
	ref := month.Format("2006-01")
	monthEnd := month.AddDate(0, 1, 0)

	var students []domain.Student
	if err := b.db.Where("gym_id = ? AND status IN ? AND plan_price > 0 AND created_at < ?", gymID,
		[]domain.StudentStatus{domain.StatusAtivo, domain.StatusInadimplente}, monthEnd).Find(&students).Error; err != nil {
		return 0, err
	}
	var already []uuid.UUID
	b.db.Model(&domain.Invoice{}).Where("gym_id = ? AND kind = ? AND reference = ?", gymID, domain.InvoiceMonthly, ref).Pluck("student_id", &already)
	has := map[uuid.UUID]bool{}
	for _, id := range already {
		has[id] = true
	}

	created := 0
	for _, st := range students {
		if has[st.ID] {
			continue
		}
		due := DueDateFor(month, st.DueDay)
		// Quem entrou depois do vencimento deste mês já pagou na matrícula (ou começa no mês seguinte)
		if st.CreatedAt.After(due.Add(24 * time.Hour)) {
			continue
		}
		inv := domain.Invoice{
			GymID: gymID, StudentID: st.ID, Kind: domain.InvoiceMonthly, Reference: ref,
			Amount: st.PlanPrice, DueDate: due,
			Description: fmt.Sprintf("Mensalidade %s — %s", month.Format("01/2006"), st.PlanName),
		}
		if err := b.Create(b.db, &inv); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

// RefreshOverdue marca cobranças vencidas e, se a academia configurou, bloqueia quem passou do prazo
func (b *BillingService) RefreshOverdue(gymID uuid.UUID) (overdue int64, blocked int) {
	today := config.StartOfDay(config.Now())
	res := b.db.Model(&domain.Invoice{}).
		Where("gym_id = ? AND status = ? AND due_date < ?", gymID, domain.InvoicePendente, today).
		Update("status", domain.InvoiceVencido)
	overdue = res.RowsAffected

	gym := b.gym(b.db, gymID)
	if gym == nil || gym.OverdueBlockDays <= 0 {
		return overdue, 0
	}
	limit := today.AddDate(0, 0, -gym.OverdueBlockDays)
	var ids []uuid.UUID
	b.db.Model(&domain.Student{}).
		Where("gym_id = ? AND status = ? AND id IN (?)", gymID, domain.StatusAtivo,
			b.db.Model(&domain.Invoice{}).Select("student_id").Where("gym_id = ? AND status = ? AND due_date < ?", gymID, domain.InvoiceVencido, limit)).
		Pluck("id", &ids)
	if len(ids) == 0 {
		return overdue, 0
	}
	b.db.Model(&domain.Student{}).Where("id IN ?", ids).Update("status", domain.StatusInadimplente)
	b.notifier.NotifyStudents(gymID, ids, domain.NotifyDue, "Mensalidade em atraso",
		"Sua entrada foi bloqueada por mensalidade em atraso. Regularize no app ou na recepção.", "/aluno/?tab=perfil")
	return overdue, len(ids)
}

// reconcile libera a aluna inadimplente quando não sobra nenhuma cobrança vencida
func (b *BillingService) reconcile(tx *gorm.DB, studentID uuid.UUID) {
	var n int64
	tx.Model(&domain.Invoice{}).Where("student_id = ? AND status = ?", studentID, domain.InvoiceVencido).Count(&n)
	if n == 0 {
		tx.Model(&domain.Student{}).Where("id = ? AND status = ?", studentID, domain.StatusInadimplente).Update("status", domain.StatusAtivo)
	}
}

// PaymentInput baixa de uma cobrança
type PaymentInput struct {
	Method     domain.PaymentMethod
	Amount     *float64 // valor recebido (padrão: valor com desconto)
	Discount   float64
	ReceivedBy *uuid.UUID
}

var validMethods = map[domain.PaymentMethod]bool{
	domain.PayPix: true, domain.PayBoleto: true, domain.PayCard: true, domain.PayCash: true, domain.PayTransfer: true,
}

// RegisterPayment dá baixa na cobrança (recepção recebeu, ou o gateway confirmou)
func (b *BillingService) RegisterPayment(gymID, invoiceID uuid.UUID, in PaymentInput) (*domain.Invoice, error) {
	if !validMethods[in.Method] {
		return nil, bizErr("Forma de pagamento inválida")
	}
	var inv domain.Invoice
	err := b.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&inv, "id = ? AND gym_id = ?", invoiceID, gymID).Error; err != nil {
			return bizErr("Cobrança não encontrada")
		}
		if inv.Status == domain.InvoicePago {
			return bizErr("Esta cobrança já está paga")
		}
		if inv.Status == domain.InvoiceCancel {
			return bizErr("Esta cobrança foi cancelada")
		}
		if in.Discount < 0 || in.Discount > inv.Amount {
			return bizErr("Desconto inválido")
		}
		inv.Discount = round2(in.Discount)
		paid := inv.Amount - inv.Discount
		if in.Amount != nil {
			if *in.Amount <= 0 {
				return bizErr("Valor recebido inválido")
			}
			paid = *in.Amount
		}
		now := time.Now()
		inv.PaidAmount = round2(paid)
		inv.PaidAt = &now
		inv.Status = domain.InvoicePago
		inv.PaymentMethod = in.Method
		inv.ReceivedByID = in.ReceivedBy
		if err := tx.Omit("Student").Save(&inv).Error; err != nil {
			return err
		}
		b.reconcile(tx, inv.StudentID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	b.notifier.NotifyStudents(gymID, []uuid.UUID{inv.StudentID}, domain.NotifyDue, "Pagamento confirmado ✅",
		fmt.Sprintf("Recebemos R$ %s referente a %s. Obrigado!", FormatBRL(inv.PaidAmount), inv.Description), "/aluno/?tab=perfil")
	return &inv, nil
}

// Cancel cancela uma cobrança em aberto (ex.: lançada errado, bolsa, isenção)
func (b *BillingService) Cancel(gymID, invoiceID uuid.UUID, reason string) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := b.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&inv, "id = ? AND gym_id = ?", invoiceID, gymID).Error; err != nil {
			return bizErr("Cobrança não encontrada")
		}
		if inv.Status == domain.InvoicePago {
			return bizErr("Cobrança paga não pode ser cancelada")
		}
		if inv.Status == domain.InvoiceCancel {
			return bizErr("Cobrança já cancelada")
		}
		inv.Status = domain.InvoiceCancel
		inv.CancelReason = strings.TrimSpace(reason)
		if err := tx.Omit("Student").Save(&inv).Error; err != nil {
			return err
		}
		b.reconcile(tx, inv.StudentID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// CancelOpenForStudent cancela cobranças futuras em aberto (matrícula cancelada)
func (b *BillingService) CancelOpenForStudent(tx *gorm.DB, studentID uuid.UUID, reason string) {
	tx.Model(&domain.Invoice{}).
		Where("student_id = ? AND status = ? AND due_date >= ?", studentID, domain.InvoicePendente, config.StartOfDay(config.Now())).
		Updates(map[string]interface{}{"status": domain.InvoiceCancel, "cancel_reason": reason})
}

// FinanceSummary números do financeiro no mês
type FinanceSummary struct {
	Month              string             `json:"month"`
	Gateway            string             `json:"gateway"`
	Expected           float64            `json:"expected"` // Mensalidades + avulsas com vencimento no mês (sem canceladas)
	Received           float64            `json:"received"` // Recebido no mês (pela data do pagamento)
	ReceivedCount      int64              `json:"received_count"`
	Pending            float64            `json:"pending"` // Em aberto com vencimento no mês
	PendingCount       int64              `json:"pending_count"`
	OverdueTotal       float64            `json:"overdue_total"` // Tudo que está vencido (qualquer mês)
	OverdueCount       int64              `json:"overdue_count"`
	DelinquentStudents int64              `json:"delinquent_students"`
	Discounts          float64            `json:"discounts"`
	ByMethod           map[string]float64 `json:"by_method"`
	DailyReceived      []DayValue         `json:"daily_received"`
}

type DayValue struct {
	Day   int     `json:"day"`
	Value float64 `json:"value"`
}

func (b *BillingService) Summary(gymID uuid.UUID, month time.Time) FinanceSummary {
	end := month.AddDate(0, 1, 0)
	s := FinanceSummary{Month: month.Format("2006-01"), Gateway: b.gateway.Name(), ByMethod: map[string]float64{}}

	var due []domain.Invoice
	b.db.Select("amount", "discount", "status").
		Where("gym_id = ? AND due_date >= ? AND due_date < ? AND status <> ?", gymID, month, end, domain.InvoiceCancel).Find(&due)
	for _, inv := range due {
		s.Expected += inv.Amount - inv.Discount
		if inv.Status == domain.InvoicePendente || inv.Status == domain.InvoiceVencido {
			s.Pending += inv.Amount - inv.Discount
			s.PendingCount++
		}
	}

	var paid []domain.Invoice
	b.db.Select("paid_amount", "discount", "payment_method", "paid_at").
		Where("gym_id = ? AND status = ? AND paid_at >= ? AND paid_at < ?", gymID, domain.InvoicePago, month, end).Find(&paid)
	daily := map[int]float64{}
	for _, inv := range paid {
		s.Received += inv.PaidAmount
		s.Discounts += inv.Discount
		s.ReceivedCount++
		s.ByMethod[string(inv.PaymentMethod)] += inv.PaidAmount
		if inv.PaidAt != nil {
			daily[inv.PaidAt.In(config.Loc()).Day()] += inv.PaidAmount
		}
	}
	for d := 1; d <= end.AddDate(0, 0, -1).Day(); d++ {
		s.DailyReceived = append(s.DailyReceived, DayValue{Day: d, Value: round2(daily[d])})
	}

	var overdue []domain.Invoice
	b.db.Select("amount", "discount", "student_id").Where("gym_id = ? AND status = ?", gymID, domain.InvoiceVencido).Find(&overdue)
	students := map[uuid.UUID]bool{}
	for _, inv := range overdue {
		s.OverdueTotal += inv.Amount - inv.Discount
		s.OverdueCount++
		students[inv.StudentID] = true
	}
	s.DelinquentStudents = int64(len(students))

	s.Expected, s.Received, s.Pending = round2(s.Expected), round2(s.Received), round2(s.Pending)
	s.OverdueTotal, s.Discounts = round2(s.OverdueTotal), round2(s.Discounts)
	for k, v := range s.ByMethod {
		s.ByMethod[k] = round2(v)
	}
	return s
}

// RunDaily rotina do agendador: gera as mensalidades do mês (e do próximo a partir do dia 25) e atualiza atrasos
func (b *BillingService) RunDaily(gymID uuid.UUID) error {
	month, _ := ParseMonth("")
	if _, err := b.GenerateMonth(gymID, month); err != nil {
		return err
	}
	if config.Now().Day() >= 25 {
		if _, err := b.GenerateMonth(gymID, month.AddDate(0, 1, 0)); err != nil {
			return err
		}
	}
	b.RefreshOverdue(gymID)
	return nil
}

// FormatBRL 1234.5 → "1.234,50"
func FormatBRL(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	cents := int64(math.Round(v * 100))
	intPart := cents / 100
	s := fmt.Sprintf("%d", intPart)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	res := fmt.Sprintf("%s,%02d", out, cents%100)
	if neg {
		return "-" + res
	}
	return res
}
