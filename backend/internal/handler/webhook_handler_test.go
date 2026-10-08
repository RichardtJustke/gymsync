package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidSignature(t *testing.T) {
	body := []byte(`{"event_id":"e1"}`)
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !ValidSignature("s3cret", body, sig) || !ValidSignature("s3cret", body, "sha256="+sig) {
		t.Fatal("assinatura válida rejeitada")
	}
	if ValidSignature("outra", body, sig) || ValidSignature("s3cret", []byte("x"), sig) || ValidSignature("", body, sig) || ValidSignature("s3cret", body, "") {
		t.Fatal("assinatura inválida aceita")
	}
}
