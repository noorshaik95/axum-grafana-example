package handlers

import (
	"crypto/rand"
	"encoding/hex"
)

// randomRequestID generates a fresh correlation id when the caller didn't
// send one. 16 hex chars is enough entropy for request-scoped correlation.
func randomRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-fallback"
	}
	return "req-" + hex.EncodeToString(b[:])
}
