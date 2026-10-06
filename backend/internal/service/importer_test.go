package service

import (
	"testing"
)

func TestDetectColumnMapping(t *testing.T) {
	headers := []string{"Nome Completo", "CPF/Doc", "E-mail do Aluno", "Telefone/Zap", "Plano Atual", "Mensalidade (R$)", "Dia Vencimento"}
	mapping := detectColumnMapping(headers)

	expected := []string{"name", "document", "email", "phone", "plan_name", "plan_price", "due_day"}
	for i, exp := range expected {
		if mapping[i] != exp {
			t.Errorf("Esperado mapping[%d] = %s, mas obteve %s", i, exp, mapping[i])
		}
	}
}

func TestParseCSVContent(t *testing.T) {
	csvData := []byte("Nome;CPF;E-mail;Telefone;Plano;Valor;Vencimento\nJoão Silva;123.456.789-00;joao@teste.com;11999998888;Trimestral;129,90;15\nMaria Souza;98765432100;maria@teste.com;11888887777;Anual;89.90;20")

	svc := &ImporterService{}
	preview, err := svc.ParseFile("alunos.csv", csvData)
	if err != nil {
		t.Fatalf("Erro inesperado ao processar CSV: %v", err)
	}

	if preview.TotalRows != 2 {
		t.Errorf("Esperado 2 linhas, obteve %d", preview.TotalRows)
	}

	if preview.ValidRows != 2 {
		t.Errorf("Esperado 2 linhas válidas, obteve %d", preview.ValidRows)
	}

	if preview.Rows[0].Name != "João Silva" {
		t.Errorf("Esperado nome João Silva, obteve %s", preview.Rows[0].Name)
	}

	if preview.Rows[0].PlanPrice != 129.90 {
		t.Errorf("Esperado valor 129.90, obteve %f", preview.Rows[0].PlanPrice)
	}

	if preview.Rows[0].DueDay != 15 {
		t.Errorf("Esperado dia 15, obteve %d", preview.Rows[0].DueDay)
	}
}
