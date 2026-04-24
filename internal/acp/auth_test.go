package acp

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewAuthManager(t *testing.T) {
	t.Run("with provided secret", func(t *testing.T) {
		am := NewAuthManager("my-secret")
		if am == nil {
			t.Fatal("NewAuthManager() returned nil")
		}
		if string(am.secret) != "my-secret" {
			t.Errorf("secret = %q, want %q", string(am.secret), "my-secret")
		}
	})

	t.Run("generates random secret when empty", func(t *testing.T) {
		am := NewAuthManager("")
		if am == nil {
			t.Fatal("NewAuthManager() returned nil")
		}
		if len(am.secret) == 0 {
			t.Error("expected non-empty secret when none provided")
		}
	})
}

func TestGenerateToken(t *testing.T) {
	am := NewAuthManager("test-secret")

	token, err := am.GenerateToken("client-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	// JWT format: header.payload.signature (three dot-separated parts).
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("token has %d dot-separated parts, want 3 (JWT format)", len(parts))
	}
}

func TestValidateToken_Valid(t *testing.T) {
	am := NewAuthManager("test-secret")

	token, err := am.GenerateToken("client-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	clientID, ok := am.ValidateToken(token)
	if !ok {
		t.Error("ValidateToken() returned false for a freshly generated token")
	}
	if clientID != "client-1" {
		t.Errorf("ValidateToken() clientID = %q, want %q", clientID, "client-1")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	am := NewAuthManager("test-secret")

	clientID, ok := am.ValidateToken("random-garbage-string")
	if ok {
		t.Error("ValidateToken() returned true for a random string")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q, want empty", clientID)
	}

	clientID, ok = am.ValidateToken("")
	if ok {
		t.Error("ValidateToken() returned true for empty string")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q, want empty", clientID)
	}

	clientID, ok = am.ValidateToken("aabbccdd.11223344.55667788")
	if ok {
		t.Error("ValidateToken() returned true for a crafted but unsigned token")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q, want empty", clientID)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	am1 := NewAuthManager("secret-one")
	am2 := NewAuthManager("secret-two")

	token, err := am1.GenerateToken("client-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	clientID, ok := am2.ValidateToken(token)
	if ok {
		t.Error("ValidateToken() returned true for token signed with a different secret")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q, want empty", clientID)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Create a token with past expiry directly using jwt library.
	claims := HeraClaims{
		ClientID: "expired-client",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "hera-acp",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			ID:        "test-expired-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.secret)
	if err != nil {
		t.Fatalf("failed to create expired token: %v", err)
	}

	clientID, ok := am.ValidateToken(tokenString)
	if ok {
		t.Error("ValidateToken() returned true for an expired token")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q, want empty", clientID)
	}
}

func TestRevokeToken(t *testing.T) {
	am := NewAuthManager("test-secret")

	token, err := am.GenerateToken("client-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Token is valid before revocation.
	clientID, ok := am.ValidateToken(token)
	if !ok {
		t.Fatal("token should be valid before revocation")
	}
	if clientID != "client-1" {
		t.Errorf("clientID = %q, want %q", clientID, "client-1")
	}

	am.RevokeToken(token)

	clientID, ok = am.ValidateToken(token)
	if ok {
		t.Error("ValidateToken() returned true after revocation")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q after revocation, want empty", clientID)
	}
}

func TestCleanExpired(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Generate a token and revoke it (adds to revocation set with 1h expiry).
	token1, err := am.GenerateToken("client-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	am.RevokeToken(token1)

	// Manually set the revocation entry to be already expired.
	am.mu.Lock()
	am.revoked[token1] = time.Now().Add(-1 * time.Minute)
	am.mu.Unlock()

	// Generate a second token and revoke it (this one stays in the set).
	token2, err := am.GenerateToken("client-2")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	am.RevokeToken(token2)

	am.CleanExpired()

	// token1's revocation entry should be cleaned (expired revocation).
	am.mu.Lock()
	_, t1Exists := am.revoked[token1]
	_, t2Exists := am.revoked[token2]
	am.mu.Unlock()

	if t1Exists {
		t.Error("expired revocation entry was not cleaned")
	}
	if !t2Exists {
		t.Error("active revocation entry was incorrectly removed by CleanExpired")
	}
}

func TestValidateToken_RejectsNoneAlgorithm(t *testing.T) {
	am := NewAuthManager("test-secret")

	// Create a token with "none" algorithm (alg attack).
	claims := HeraClaims{
		ClientID: "attacker",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "hera-acp",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			ID:        "attack-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to create none-alg token: %v", err)
	}

	clientID, ok := am.ValidateToken(tokenString)
	if ok {
		t.Error("ValidateToken() accepted a token with 'none' algorithm")
	}
	if clientID != "" {
		t.Errorf("ValidateToken() clientID = %q, want empty for none-alg token", clientID)
	}
}
