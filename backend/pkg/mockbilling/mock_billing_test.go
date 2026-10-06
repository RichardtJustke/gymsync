package mockbilling

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateBoletoDigits(t *testing.T) {
	digits := GenerateBoletoDigits(119.90, 10)
	if len(digits) < 40 {
		t.Errorf("Linha digitável muito curta: %s", digits)
	}

	parts := strings.Split(digits, " ")
	if len(parts) != 5 {
		t.Errorf("Linha digitável deve possuir 5 blocos separados por espaço, obteve %d", len(parts))
	}
}

func TestGeneratePixPayload(t *testing.T) {
	id := uuid.New()
	payload := GeneratePixPayload(id, 89.90, "GymSync Fitness")

	if !strings.HasPrefix(payload, "00020126580014br.gov.bcb.pix") {
		t.Errorf("Payload PIX não possui cabeçalho padrão EMV: %s", payload)
	}

	if !strings.Contains(payload, "GYMSYNCFITNESS") {
		t.Errorf("Payload PIX deve conter o nome da academia: %s", payload)
	}
}
