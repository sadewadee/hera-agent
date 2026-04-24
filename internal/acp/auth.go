package acp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthManager handles JWT token generation and validation for ACP.
type AuthManager struct {
	secret  []byte
	revoked map[string]time.Time // small revocation set for RevokeToken
	mu      sync.Mutex
}

// NewAuthManager creates a new auth manager. If secret is empty, a random
// 32-byte secret is generated.
func NewAuthManager(secret string) *AuthManager {
	if secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		secret = hex.EncodeToString(b)
	}
	return &AuthManager{
		secret:  []byte(secret),
		revoked: make(map[string]time.Time),
	}
}

// HeraClaims are the JWT claims embedded in every ACP token.
type HeraClaims struct {
	ClientID string `json:"client_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT for the given client ID.
func (am *AuthManager) GenerateToken(clientID string) (string, error) {
	claims := HeraClaims{
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "hera-acp",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.secret)
}

// ValidateToken verifies and parses a JWT. It returns the client ID and true
// on success, or ("", false) when the token is invalid, expired, or revoked.
func (am *AuthManager) ValidateToken(tokenString string) (string, bool) {
	// Check revocation list first.
	am.mu.Lock()
	if _, revoked := am.revoked[tokenString]; revoked {
		am.mu.Unlock()
		return "", false
	}
	am.mu.Unlock()

	token, err := jwt.ParseWithClaims(tokenString, &HeraClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return am.secret, nil
	})
	if err != nil {
		return "", false
	}

	claims, ok := token.Claims.(*HeraClaims)
	if !ok || !token.Valid {
		return "", false
	}
	return claims.ClientID, true
}

// RevokeToken adds a token to the revocation set. The entry is kept for 1 hour
// (matching the token's max lifetime) so that a revoked token cannot be reused
// before its natural expiry.
func (am *AuthManager) RevokeToken(tokenString string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.revoked[tokenString] = time.Now().Add(time.Hour)
}

// CleanExpired removes revocation entries whose retention period has elapsed.
func (am *AuthManager) CleanExpired() {
	am.mu.Lock()
	defer am.mu.Unlock()
	now := time.Now()
	for token, expiry := range am.revoked {
		if now.After(expiry) {
			delete(am.revoked, token)
		}
	}
}
