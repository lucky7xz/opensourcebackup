package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrInvalidToken is returned when a token does not match any valid record.
var ErrInvalidToken = errors.New("auth: invalid or expired token")

// ErrTokenAlreadyUsed is returned when an enrollment token has already been consumed.
var ErrTokenAlreadyUsed = errors.New("auth: enrollment token already used")

// ErrTokenRevoked is returned when a token has been revoked.
var ErrTokenRevoked = errors.New("auth: token revoked")

// GenerateToken creates a cryptographically random 32-byte token
// encoded as URL-safe base64 (43 characters, 256 bits of entropy).
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 hash of the token.
// Only hashes are stored in the database — the plain token is never persisted.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
