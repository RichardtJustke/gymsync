package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gymsync/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RawImportRow linha crua lida da planilha
type RawImportRow struct {
	RowIndex int               `json:"row_index"`
	RawData  map[string]string `json:"raw_data"`
}

// ParsedStudentRow registro processado e normalizado
type ParsedStudentRow struct {
	RowIndex  int     `json:"row_index"`
	Name      string  `json:"name"`
	Document  string  `json:"document"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	PlanName  string  `json:"plan_name"`
	PlanPrice float64 `json:"plan_price"`
	DueDay    int     `json:"due_day"`
	Status    string  `json:"status"`
	IsValid   bool    `json:"is_valid"`
	Error     string  `json:"error,omitempty"`
}

// ImportPreviewResult resultado do Dry Run para o usuário revisar na tela
type ImportPreviewResult struct {
	TotalRows      int                `json:"total_rows"`
	ValidRows      int                `json:"valid_rows"`
	InvalidRows    int                `json:"invalid_rows"`
	DetectedFields map[string]string  `json:"detected_fields"` // Ex: "Nome do Aluno" -> "name"
	Rows           []ParsedStudentRow `json:"rows"`
}

// ImportSummary resumo após a gravação definitiva no banco
type ImportSummary struct {
	TotalProcessed int      `json:"total_processed"`
	CreatedCount   int      `json:"created_count"`
	UpdatedCount   int      `json:"updated_count"`
	SkippedCount   int      `json:"skipped_count"`
	Errors         []string `json:"errors"`
}

type ImporterService struct {
	db *gorm.DB
}

func NewImporterService(db *gorm.DB) *ImporterService {
	return &ImporterService{db: db}
}

// ParseFile lê bytes de CSV ou Excel e extrai linhas estruturadas
func (s *ImporterService) ParseFile(filename string, fileData []byte) (*ImportPreviewResult, error) {
	lowerName := strings.ToLower(filename)
	var records [][]string
	var err error

	if strings.HasSuffix(lowerName, ".xlsx") || strings.HasSuffix(lowerName, ".xls") {
		records, err = readExcel(fileData)
	} else {
		records, err = readCSV(fileData)
	}

	if err != nil {
		return nil, fmt.Errorf("falha ao interpretar arquivo: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("arquivo deve conter pelo menos 1 linha de cabeçalho e 1 linha de dados")
	}

	headers := records[0]
	mapping := detectColumnMapping(headers)

	result := &ImportPreviewResult{
		TotalRows:      len(records) - 1,
		DetectedFields: make(map[string]string),
		Rows:           make([]ParsedStudentRow, 0, len(records)-1),
	}

	for idx, col := range mapping {
		if col != "" {
			result.DetectedFields[headers[idx]] = col
		}
	}

	for rIdx := 1; rIdx < len(records); rIdx++ {
		row := records[rIdx]
		if isEmptyRow(row) {
			continue
		}

		parsed := parseRow(rIdx+1, row, mapping)
		if parsed.IsValid {
			result.ValidRows++
		} else {
			result.InvalidRows++
		}
		result.Rows = append(result.Rows, parsed)
	}

	return result, nil
}

// ExecuteImport grava os alunos no banco para a academia especificada
func (s *ImporterService) ExecuteImport(gymID uuid.UUID, rows []ParsedStudentRow) (*ImportSummary, error) {
	summary := &ImportSummary{
		TotalProcessed: len(rows),
		Errors:         make([]string, 0),
	}

	for _, item := range rows {
		if !item.IsValid {
			summary.SkippedCount++
			continue
		}

		// Cria ou busca usuário
		cleanDoc := cleanDocument(item.Document)
		var user domain.User
		err := s.db.Where("gym_id = ? AND (document = ? OR email = ?)", gymID, cleanDoc, item.Email).First(&user).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			summary.Errors = append(summary.Errors, fmt.Sprintf("Linha %d (%s): erro de consulta: %v", item.RowIndex, item.Name, err))
			continue
		}

		// Senha inicial: 6 primeiros dígitos do CPF ou "123456"
		initialPass := "123456"
		if len(cleanDoc) >= 6 {
			initialPass = cleanDoc[:6]
		}
		passHash, _ := bcrypt.GenerateFromPassword([]byte(initialPass), bcrypt.DefaultCost)

		if err == gorm.ErrRecordNotFound {
			// Novo usuário
			user = domain.User{
				GymID:        gymID,
				Name:         item.Name,
				Email:        item.Email,
				Document:     cleanDoc,
				Phone:        item.Phone,
				PasswordHash: string(passHash),
				Role:         domain.RoleAluno,
				FirstAccess:  true,
			}
			if err := s.db.Create(&user).Error; err != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("Linha %d (%s): falha ao criar usuário: %v", item.RowIndex, item.Name, err))
				continue
			}

			// Cria Perfil de Aluno
			status := domain.StatusAtivo
			if strings.EqualFold(item.Status, "INADIMPLENTE") {
				status = domain.StatusInadimplente
			}

			dueDay := item.DueDay
			if dueDay <= 0 || dueDay > 31 {
				dueDay = 10
			}

			student := domain.Student{
				GymID:        gymID,
				UserID:       user.ID,
				PlanName:     item.PlanName,
				PlanPrice:    item.PlanPrice,
				DueDay:       dueDay,
				Status:       status,
				ImportSource: "importador-planilha",
			}

			if err := s.db.Create(&student).Error; err != nil {
				summary.Errors = append(summary.Errors, fmt.Sprintf("Linha %d (%s): falha ao criar perfil de aluno: %v", item.RowIndex, item.Name, err))
				continue
			}

			// Cria automaticamente fatura do mês atual mockada para o novo aluno
			s.createInitialInvoice(gymID, student.ID, item.PlanPrice, dueDay)

			summary.CreatedCount++
		} else {
			// Usuário já existe, atualiza dados do aluno
			var student domain.Student
			if err := s.db.Where("gym_id = ? AND user_id = ?", gymID, user.ID).First(&student).Error; err == nil {
				student.PlanName = item.PlanName
				if item.PlanPrice > 0 {
					student.PlanPrice = item.PlanPrice
				}
				s.db.Save(&student)
				summary.UpdatedCount++
			}
		}
	}

	return summary, nil
}

func (s *ImporterService) createInitialInvoice(gymID, studentID uuid.UUID, amount float64, dueDay int) {
	if amount <= 0 {
		amount = 99.90
	}
	now := time.Now()
	dueDate := time.Date(now.Year(), now.Month(), dueDay, 23, 59, 59, 0, now.Location())
	if dueDate.Before(now) {
		dueDate = dueDate.AddDate(0, 1, 0)
	}

	invoice := domain.Invoice{
		GymID:        gymID,
		StudentID:    studentID,
		Amount:       amount,
		DueDate:      dueDate,
		Status:       domain.InvoicePendente,
		BoletoDigits: fmt.Sprintf("34191.%05d %05d.510047 91020.150008 5 %014d", studentID.ID()&0xFFFF, dueDay*1000, int(amount*100)),
		PixCopyPaste: fmt.Sprintf("00020126580014br.gov.bcb.pix0136gymsync-%s520400005303986540%.2f5802BR5916GYMSYNC FITNESS6009SAO PAULO62070503***6304ABCD", studentID.String(), amount),
		Description:  "Mensalidade Inicial de Boas-Vindas",
	}
	s.db.Create(&invoice)
}

// Helpers de leitura e normalização

func readCSV(data []byte) ([][]string, error) {
	// Detecta delimitador (vírgula ou ponto e vírgula)
	sample := string(data)
	if len(sample) > 500 {
		sample = sample[:500]
	}
	comma := ','
	if strings.Count(sample, ";") > strings.Count(sample, ",") {
		comma = ';'
	} else if strings.Count(sample, "\t") > strings.Count(sample, ",") {
		comma = '\t'
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = comma
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	return reader.ReadAll()
}

func readExcel(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("planilha vazia sem abas")
	}

	return f.GetRows(sheets[0])
}

func detectColumnMapping(headers []string) []string {
	mapping := make([]string, len(headers))
	for i, h := range headers {
		clean := normalize(h)
		switch {
		case containsAny(clean, "email", "e-mail", "correio"):
			mapping[i] = "email"
		case containsAny(clean, "cpf", "doc", "documento", "identidade"):
			mapping[i] = "document"
		case containsAny(clean, "tel", "celular", "fone", "whatsapp", "zap"):
			mapping[i] = "phone"
		case containsAny(clean, "valor", "preco", "preço", "mensalidade", "tarifa"):
			mapping[i] = "plan_price"
		case containsAny(clean, "venc", "dia", "vencimento"):
			mapping[i] = "due_day"
		case containsAny(clean, "plano", "modalidade", "tipo"):
			mapping[i] = "plan_name"
		case containsAny(clean, "status", "situacao", "situação", "estado"):
			mapping[i] = "status"
		case containsAny(clean, "nome", "aluno", "cliente", "estudante"):
			mapping[i] = "name"
		default:
			mapping[i] = ""
		}
	}
	return mapping
}

func parseRow(rowIdx int, row []string, mapping []string) ParsedStudentRow {
	item := ParsedStudentRow{
		RowIndex: rowIdx,
		IsValid:  true,
		PlanName: "Plano Padrão",
		DueDay:   10,
		Status:   "ATIVO",
	}

	for i, col := range mapping {
		if i >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[i])
		switch col {
		case "name":
			item.Name = val
		case "document":
			item.Document = val
		case "email":
			item.Email = strings.ToLower(val)
		case "phone":
			item.Phone = val
		case "plan_name":
			if val != "" {
				item.PlanName = val
			}
		case "plan_price":
			item.PlanPrice = parsePrice(val)
		case "due_day":
			if d, err := strconv.Atoi(val); err == nil && d > 0 && d <= 31 {
				item.DueDay = d
			}
		case "status":
			if strings.EqualFold(val, "INADIMPLENTE") || strings.EqualFold(val, "ATRASADO") || strings.EqualFold(val, "DEVENDO") {
				item.Status = "INADIMPLENTE"
			}
		}
	}

	if item.Name == "" {
		item.IsValid = false
		item.Error = "Nome é obrigatório"
	}

	if item.Email == "" && item.Document == "" && item.Phone == "" {
		item.IsValid = false
		item.Error = "Necessário ao menos CPF, Telefone ou E-mail para identificar o aluno"
	}

	// Gera e-mail fictício caso o aluno só tenha nome e telefone/CPF
	if item.Email == "" {
		cleanName := strings.ToLower(strings.ReplaceAll(item.Name, " ", "."))
		item.Email = fmt.Sprintf("%s_%d@aluno.gymsync.local", cleanName, rowIdx)
	}

	return item
}

func parsePrice(val string) float64 {
	val = strings.ReplaceAll(val, "R$", "")
	val = strings.ReplaceAll(val, " ", "")
	val = strings.ReplaceAll(val, ".", "")
	val = strings.ReplaceAll(val, ",", ".")
	p, _ := strconv.ParseFloat(val, 64)
	return p
}

func cleanDocument(doc string) string {
	var sb strings.Builder
	for _, r := range doc {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("ã", "a", "á", "a", "à", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "õ", "o", "ú", "u", "ç", "c")
	return replacer.Replace(s)
}

func containsAny(s string, terms ...string) bool {
	for _, t := range terms {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
