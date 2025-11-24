package achem

import (
	"crypto/rand"
	"encoding/hex"
)

func NewRandomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
