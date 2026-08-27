package token

import (
	"encoding/base64"
	"testing"
)

func TestTokenNew_NonEmpty(t *testing.T) {
	tok, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if tok == "" {
		t.Fatal("New() returned an empty token")
	}
}

func TestTokenNew_DiffersAcrossCalls(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	b, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if a == b {
		t.Fatalf("expected two successive calls to New() to differ, got %q twice", a)
	}
}

func TestTokenNew_DecodedLengthIs16Bytes(t *testing.T) {
	tok, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("token %q is not valid base64url (RawURLEncoding): %v", tok, err)
	}
	if len(decoded) != 16 {
		t.Fatalf("expected decoded token to be 16 bytes (128 bits), got %d bytes", len(decoded))
	}
}
