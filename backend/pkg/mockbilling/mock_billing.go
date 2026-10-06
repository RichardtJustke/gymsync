package mockbilling

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MockBillingService struct {
	db *gorm.DB
}

func NewMockBillingService(db *gorm.DB) *MockBillingService {
	return &MockBillingService{db: db}
}

// GenerateBoletoDigits cria uma linha digitável verossímil de teste
func GenerateBoletoDigits(amount float64, dueDay int) string {
	banco := "341" // Itaú mock
	moeda := "9"   // Real
	fator := 9500 + dueDay
	valorCents := int(amount * 100)
	campo1 := fmt.Sprintf("%s%s1.79%03d", banco, moeda, rand.Intn(999))
	campo2 := fmt.Sprintf("%05d.%06d", rand.Intn(99999), rand.Intn(999999))
	campo3 := fmt.Sprintf("%05d.%06d", rand.Intn(99999), rand.Intn(999999))
	dvGeral := rand.Intn(9) + 1
	campo4 := fmt.Sprintf("%d%010d", fator, valorCents)
	return fmt.Sprintf("%s %s %s %d %s", campo1, campo2, campo3, dvGeral, campo4)
}

// GeneratePixPayload gera um código PIX Copia e Cola simulado válido em formato EMV
func GeneratePixPayload(invoiceID uuid.UUID, amount float64, gymName string) string {
	cleanName := strings.ToUpper(strings.ReplaceAll(gymName, " ", ""))
	if len(cleanName) > 20 {
		cleanName = cleanName[:20]
	}
	return fmt.Sprintf("00020126580014br.gov.bcb.pix0136gymsync-%s520400005303986540%.2f5802BR59%02d%s6009SAOPAULO62070503***6304MOCK",
		invoiceID.String(), amount, len(cleanName), cleanName)
}

// SimulatePayment simula o pagamento instantâneo da fatura pelo aluno ou webhook
func (s *MockBillingService) SimulatePayment(invoiceID uuid.UUID) (*domain.Invoice, error) {
	var invoice domain.Invoice
	if err := s.db.Preload("Student").First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return nil, fmt.Errorf("fatura não encontrada: %w", err)
	}

	if invoice.Status == domain.InvoicePago {
		return &invoice, nil // Já está pago
	}

	now := time.Now()
	invoice.Status = domain.InvoicePago
	invoice.PaidAt = &now

	if err := s.db.Save(&invoice).Error; err != nil {
		return nil, fmt.Errorf("falha ao atualizar status da fatura: %w", err)
	}

	// Se o aluno estava inadimplente, verifica se ainda possui outras faturas vencidas
	if invoice.StudentID != uuid.Nil {
		var pendingCount int64
		s.db.Model(&domain.Invoice{}).
			Where("student_id = ? AND status IN (?, ?) AND due_date < ?", invoice.StudentID, domain.InvoicePendente, domain.InvoiceVencido, now).
			Count(&pendingCount)

		if pendingCount == 0 {
			// Regulariza o aluno para ATIVO
			s.db.Model(&domain.Student{}).Where("id = ?", invoice.StudentID).Update("status", domain.StatusAtivo)
		}
	}

	return &invoice, nil
}

// GenerateBoletoHTML gera uma visualização imprimível estilizada do boleto bancário mock
func GenerateBoletoHTML(invoice *domain.Invoice, gym *domain.Gym, studentName, studentDoc string) string {
	gymName := "GymSync Academia"
	gymDoc := "00.000.000/0001-00"
	if gym != nil {
		gymName = gym.Name
		gymDoc = gym.Document
	}

	dueDateStr := invoice.DueDate.Format("02/01/2006")
	amountStr := fmt.Sprintf("R$ %.2f", invoice.Amount)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>Boleto Bancário - %s</title>
  <style>
    body { font-family: monospace, Arial, sans-serif; background: #f3f4f6; padding: 20px; }
    .boleto { max-width: 780px; margin: 0 auto; background: #fff; padding: 24px; border: 2px dashed #333; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
    .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #000; padding-bottom: 8px; margin-bottom: 12px; }
    .bank { font-size: 24px; font-weight: bold; }
    .linha-digitavel { font-size: 15px; font-weight: bold; letter-spacing: 1px; }
    .table { width: 100%%; border-collapse: collapse; margin-bottom: 12px; }
    .table td { border: 1px solid #444; padding: 6px 10px; font-size: 12px; }
    .label { font-size: 10px; text-transform: uppercase; color: #555; display: block; }
    .val { font-size: 13px; font-weight: bold; }
    .barcode { height: 60px; background: repeating-linear-gradient(90deg, #000, #000 3px, #fff 3px, #fff 6px, #000 6px, #000 8px, #fff 8px, #fff 10px); margin: 20px 0; }
    .badge { display: inline-block; padding: 4px 8px; background: #fef08a; border: 1px solid #ca8a04; border-radius: 4px; font-weight: bold; margin-bottom: 10px; }
    .btn-print { background: #2563eb; color: #fff; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; font-family: sans-serif; }
    @media print { .btn-print { display: none; } body { background: #fff; padding: 0; } }
  </style>
</head>
<body>
  <div class="boleto">
    <div style="display:flex; justify-content: space-between; align-items: center;">
      <span class="badge">AMBIENTE DE TESTE / MOCK FUNCIONAL</span>
      <button class="btn-print" onclick="window.print()">Imprimir / Salvar PDF</button>
    </div>
    <div class="header">
      <div class="bank">BANCO GYMSYNC | 341-7</div>
      <div class="linha-digitavel">%s</div>
    </div>
    <table class="table">
      <tr>
        <td style="width: 70%%;">
          <span class="label">Local de Pagamento</span>
          <span class="val">Pagável em qualquer banco, app ou correspondente bancário até o vencimento.</span>
        </td>
        <td style="width: 30%%;">
          <span class="label">Vencimento</span>
          <span class="val" style="color:#b91c1c;">%s</span>
        </td>
      </tr>
      <tr>
        <td>
          <span class="label">Beneficiário / Academia</span>
          <span class="val">%s (CNPJ: %s)</span>
        </td>
        <td>
          <span class="label">Valor do Documento</span>
          <span class="val" style="font-size: 16px;">%s</span>
        </td>
      </tr>
      <tr>
        <td>
          <span class="label">Pagador / Aluno</span>
          <span class="val">%s (CPF: %s)</span>
        </td>
        <td>
          <span class="label">Status</span>
          <span class="val">%s</span>
        </td>
      </tr>
      <tr>
        <td colspan="2">
          <span class="label">Instruções</span>
          <span class="val">%s - Não receber após 30 dias do vencimento. Cobrança de juros simulada.</span>
        </td>
      </tr>
    </table>
    <div class="barcode"></div>
    <div style="text-align: center; font-size: 11px; color: #666;">Corte na linha pontilhada | Autenticação Mecânica</div>
  </div>
</body>
</html>`, invoice.Description, invoice.BoletoDigits, dueDateStr, gymName, gymDoc, amountStr, studentName, studentDoc, invoice.Status, invoice.Description)
}
