package usecase

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInventoryUnauthorized = errors.New("Senha inválida. A alteração não foi autorizada.")

// Salted PBKDF2 verifier for the inventory administrator password. The password
// itself is never shipped to the browser, stored in audit records, or returned.
const inventoryPasswordVerifier = "d91604030dac6d5ce3a323641fd69763:e10ef5ebc5259517051bb5b904ff2cf78799dc9773de99c8e4df6e4e41103d92"

func (s *InventoryService) authorize(password string) error {
	if len(password) == 0 || len(password) > 256 {
		return ErrInventoryUnauthorized
	}
	saltHex, expectedHex, ok := strings.Cut(s.passwordVerifier, ":")
	salt, err := hex.DecodeString(saltHex)
	if !ok || err != nil || len(salt) != 16 {
		return ErrInventoryUnauthorized
	}
	expected, err := hex.DecodeString(expectedHex)
	if err != nil || len(expected) != 32 {
		return ErrInventoryUnauthorized
	}
	actual, err := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	if err != nil || subtle.ConstantTimeCompare(actual, expected) != 1 {
		return ErrInventoryUnauthorized
	}
	return nil
}

type InventoryDeleteInput struct {
	RequestID string `json:"requestId"`
	Password  string `json:"password"`
}
