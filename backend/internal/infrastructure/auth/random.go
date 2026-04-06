package auth

import (
	"crypto/rand"
	"encoding/hex"
)

type TokenGenerator struct{}

func NewTokenGenerator() TokenGenerator { return TokenGenerator{} }

func (g TokenGenerator) NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (g TokenGenerator) NewToken() (string, error) {
	var b [32]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
