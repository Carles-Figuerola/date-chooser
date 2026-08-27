// Package token generates cryptographically random, URL-safe tokens used as
// unguessable access credentials for polls (participant and admin links).
package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// tokenBytes is the number of random bytes read per token: 16 bytes = 128
// bits of entropy, encoded as base64url (RawURLEncoding, no padding).
const tokenBytes = 16

// New returns a new cryptographically random, URL-safe token. It reads
// randomness exclusively from crypto/rand — never a non-cryptographic PRNG —
// because tokens are the sole access-control mechanism for polls (see
// threat T-01-01 in the phase threat model).
func New() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: reading random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
