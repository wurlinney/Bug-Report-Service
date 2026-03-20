package security

import (
	"crypto/rand"
	"encoding/hex"
)

// TokenGenerator implements ports.Random with crypto-grade randomness.
type TokenGenerator struct{}

func NewTokenGenerator() TokenGenerator { return TokenGenerator{} }

func (g TokenGenerator) NewID() string {
	// 128-bit id, hex encoded.
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (g TokenGenerator) NewToken() (string, error) {
	// 256-bit token, hex encoded.
	var b [32]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
