package service

import (
	"crypto/rand"
	"math/big"
)

// Sem caracteres ambíguos (0/O, 1/l/I) para facilitar digitar a senha provisória
const tempAlphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// TempPassword gera senha provisória aleatória (a usuária troca no primeiro acesso)
func TempPassword() string {
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(tempAlphabet))))
		b[i] = tempAlphabet[n.Int64()]
	}
	return string(b)
}
