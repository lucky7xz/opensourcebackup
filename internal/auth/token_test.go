package auth_test

import (
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/auth"
)

func TestGenerateToken_ProducesUniqueTokens(t *testing.T) {
	t1, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	t2, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if t1 == t2 {
		t.Error("two consecutive tokens must not be equal")
	}
}

func TestGenerateToken_HasSufficientLength(t *testing.T) {
	tok, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if len(tok) < 40 {
		t.Errorf("token too short: %d chars", len(tok))
	}
}

func TestHashToken_IsDeterministic(t *testing.T) {
	tok := "test-token-value"
	h1 := auth.HashToken(tok)
	h2 := auth.HashToken(tok)
	if h1 != h2 {
		t.Error("hash must be deterministic")
	}
}

func TestHashToken_DiffersForDifferentTokens(t *testing.T) {
	if auth.HashToken("token-a") == auth.HashToken("token-b") {
		t.Error("different tokens must produce different hashes")
	}
}

func TestHashToken_DoesNotContainRawToken(t *testing.T) {
	tok := "my-secret-token"
	h := auth.HashToken(tok)
	if h == tok {
		t.Error("hash must not equal the raw token")
	}
}
