package gateway

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPairingStore_NewPairingStore(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	require.NotNil(t, ps)
}

func TestPairingStore_DefaultTTL(t *testing.T) {
	ps := NewPairingStore(0)
	assert.Equal(t, 10*time.Minute, ps.codeTTL)
}

func TestPairingStore_GenerateCode(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, err := ps.GenerateCode("telegram")
	require.NoError(t, err)
	assert.Len(t, code, 6) // 3 bytes = 6 hex chars
}

func TestPairingStore_GenerateCode_Unique(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := ps.GenerateCode("telegram")
		require.NoError(t, err)
		codes[code] = true
	}
	assert.Greater(t, len(codes), 90) // high probability of uniqueness
}

func TestPairingStore_ClaimCode_Success(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, _ := ps.GenerateCode("telegram")

	ok := ps.ClaimCode(code, "telegram", "user1")
	assert.True(t, ok)
	assert.True(t, ps.IsAuthorized("telegram", "user1"))
}

func TestPairingStore_ClaimCode_InvalidCode(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	ok := ps.ClaimCode("invalid", "telegram", "user1")
	assert.False(t, ok)
}

func TestPairingStore_ClaimCode_AlreadyClaimed(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, _ := ps.GenerateCode("telegram")

	ps.ClaimCode(code, "telegram", "user1")
	ok := ps.ClaimCode(code, "telegram", "user2")
	assert.False(t, ok)
}

func TestPairingStore_IsAuthorized(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	assert.False(t, ps.IsAuthorized("telegram", "user1"))

	code, _ := ps.GenerateCode("telegram")
	ps.ClaimCode(code, "telegram", "user1")

	assert.True(t, ps.IsAuthorized("telegram", "user1"))
	assert.False(t, ps.IsAuthorized("telegram", "user2"))
}

func TestPairingStore_RevokeUser(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, _ := ps.GenerateCode("telegram")
	ps.ClaimCode(code, "telegram", "user1")

	ok := ps.RevokeUser("telegram", "user1")
	assert.True(t, ok)
	assert.False(t, ps.IsAuthorized("telegram", "user1"))
}

func TestPairingStore_RevokeUser_NotFound(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	ok := ps.RevokeUser("telegram", "nobody")
	assert.False(t, ok)
}

func TestPairingStore_AuthorizedUsers(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	c1, _ := ps.GenerateCode("telegram")
	c2, _ := ps.GenerateCode("discord")
	ps.ClaimCode(c1, "telegram", "user1")
	ps.ClaimCode(c2, "discord", "user2")

	users := ps.AuthorizedUsers()
	assert.Len(t, users, 2)
}

func TestPairingStore_PendingCodes(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	ps.GenerateCode("telegram")
	ps.GenerateCode("discord")

	pending := ps.PendingCodes()
	assert.Len(t, pending, 2)
}

func TestPairingStore_PendingCodes_ExcludesClaimed(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, _ := ps.GenerateCode("telegram")
	ps.GenerateCode("discord")
	ps.ClaimCode(code, "telegram", "user1")

	pending := ps.PendingCodes()
	assert.Len(t, pending, 1)
}

func TestPairingStore_CleanExpired(t *testing.T) {
	ps := NewPairingStore(1 * time.Millisecond)
	ps.GenerateCode("telegram")
	ps.GenerateCode("discord")

	time.Sleep(10 * time.Millisecond)
	removed := ps.CleanExpired()
	assert.Equal(t, 2, removed)
}

func TestPairingStore_VerifyOnMessage_AlreadyAuthorized(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, _ := ps.GenerateCode("telegram")
	ps.ClaimCode(code, "telegram", "user1")

	ok := ps.VerifyOnMessage("telegram", "user1", "random text")
	assert.True(t, ok)
}

func TestPairingStore_VerifyOnMessage_ClaimsCode(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	code, _ := ps.GenerateCode("telegram")

	ok := ps.VerifyOnMessage("telegram", "user1", code)
	assert.True(t, ok)
	assert.True(t, ps.IsAuthorized("telegram", "user1"))
}

func TestPairingStore_VerifyOnMessage_InvalidText(t *testing.T) {
	ps := NewPairingStore(5 * time.Minute)
	ok := ps.VerifyOnMessage("telegram", "user1", "not a code")
	assert.False(t, ok)
}
