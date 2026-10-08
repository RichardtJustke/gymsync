package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Código de check-in rotativo (estilo TOTP): a recepção exibe um QR com o código atual,
// a aluna escaneia no app. O código muda a cada CheckinStep e é amarrado à academia,
// então print/foto antigo não funciona e não dá para fazer check-in de casa.
const (
	CheckinStep  = 30 * time.Second
	checkinGrace = 2 // aceita o código atual e os 2 anteriores (~60s para escanear)
	CheckinQRPrefix = "unika-checkin:"
)

type CheckinCodes struct {
	secret []byte
}

func NewCheckinCodes(secret []byte) *CheckinCodes {
	// Deriva uma chave própria para não reutilizar a chave do JWT diretamente
	m := hmac.New(sha256.New, secret)
	m.Write([]byte("checkin-codes-v1"))
	return &CheckinCodes{secret: m.Sum(nil)}
}

func (c *CheckinCodes) codeFor(gymID uuid.UUID, counter int64) string {
	m := hmac.New(sha256.New, c.secret)
	m.Write(gymID[:])
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(counter))
	m.Write(buf[:])
	sum := m.Sum(nil)
	// Truncamento dinâmico (RFC 4226) → 6 dígitos
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1_000_000)
}

// Current código vigente e quantos segundos faltam para trocar
func (c *CheckinCodes) Current(gymID uuid.UUID, now time.Time) (string, int) {
	counter := now.Unix() / int64(CheckinStep.Seconds())
	next := (counter + 1) * int64(CheckinStep.Seconds())
	return c.codeFor(gymID, counter), int(next - now.Unix())
}

// Valid confere o código dentro da janela de tolerância
func (c *CheckinCodes) Valid(gymID uuid.UUID, code string, now time.Time) bool {
	if len(code) != 6 {
		return false
	}
	counter := now.Unix() / int64(CheckinStep.Seconds())
	for i := int64(0); i <= checkinGrace; i++ {
		if hmac.Equal([]byte(c.codeFor(gymID, counter-i)), []byte(code)) {
			return true
		}
	}
	return false
}
