package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// PairingCode represents a pending or completed pairing.
type PairingCode struct {
	Code      string    `json:"code"`
	Platform  string    `json:"platform"`
	UserID    string    `json:"user_id,omitempty"` // empty until claimed
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Claimed   bool      `json:"claimed"`
}

// AuthorizedUser represents a user who has been authorized via pairing.
type AuthorizedUser struct {
	Platform     string    `json:"platform"`
	UserID       string    `json:"user_id"`
	AuthorizedAt time.Time `json:"authorized_at"`
	PairingCode  string    `json:"pairing_code"`
}

// PairingStore manages pairing codes and authorized users.
type PairingStore struct {
	mu         sync.Mutex
	codes      map[string]*PairingCode    // code -> PairingCode
	authorized map[string]*AuthorizedUser // "platform:userID" -> AuthorizedUser
	codeTTL    time.Duration
}

// NewPairingStore creates a new pairing store.
func NewPairingStore(codeTTL time.Duration) *PairingStore {
	if codeTTL <= 0 {
		codeTTL = 10 * time.Minute
	}
	return &PairingStore{
		codes:      make(map[string]*PairingCode),
		authorized: make(map[string]*AuthorizedUser),
		codeTTL:    codeTTL,
	}
}

// GenerateCode creates a new pairing code for a platform.
func (ps *PairingStore) GenerateCode(platform string) (string, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Generate a 6-character hex code.
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	code := hex.EncodeToString(b)

	now := time.Now()
	ps.codes[code] = &PairingCode{
		Code:      code,
		Platform:  platform,
		CreatedAt: now,
		ExpiresAt: now.Add(ps.codeTTL),
	}

	return code, nil
}

// ClaimCode attempts to claim a pairing code for a user. Returns true if
// the code was valid and not yet claimed.
func (ps *PairingStore) ClaimCode(code, platform, userID string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	pc, ok := ps.codes[code]
	if !ok {
		return false
	}

	// Check expiry.
	if time.Now().After(pc.ExpiresAt) {
		delete(ps.codes, code)
		return false
	}

	// Check if already claimed.
	if pc.Claimed {
		return false
	}

	// Claim the code.
	pc.Claimed = true
	pc.UserID = userID

	// Authorize the user.
	key := platform + ":" + userID
	ps.authorized[key] = &AuthorizedUser{
		Platform:     platform,
		UserID:       userID,
		AuthorizedAt: time.Now(),
		PairingCode:  code,
	}

	return true
}

// IsAuthorized checks if a user is authorized on a platform.
func (ps *PairingStore) IsAuthorized(platform, userID string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	key := platform + ":" + userID
	_, ok := ps.authorized[key]
	return ok
}

// Authorize pre-authorizes a user without going through the pairing-code
// flow. Used to seed the store from config.yaml's allow_list entries at
// startup. Idempotent: re-authorizing an existing user is a no-op.
func (ps *PairingStore) Authorize(platform, userID string) {
	if userID == "" {
		return
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	key := platform + ":" + userID
	if _, exists := ps.authorized[key]; exists {
		return
	}
	ps.authorized[key] = &AuthorizedUser{
		Platform:     platform,
		UserID:       userID,
		AuthorizedAt: time.Now(),
		PairingCode:  "config",
	}
}

// RevokeUser removes a user's authorization.
func (ps *PairingStore) RevokeUser(platform, userID string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	key := platform + ":" + userID
	_, ok := ps.authorized[key]
	if ok {
		delete(ps.authorized, key)
	}
	return ok
}

// AuthorizeUser directly grants authorization to a user on a platform.
// This is used for administrative provisioning and testing.
func (ps *PairingStore) AuthorizeUser(platform, userID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	key := platform + ":" + userID
	ps.authorized[key] = &AuthorizedUser{
		Platform:     platform,
		UserID:       userID,
		AuthorizedAt: time.Now(),
	}
}

// AuthorizedUsers returns a list of all authorized users.
func (ps *PairingStore) AuthorizedUsers() []AuthorizedUser {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	result := make([]AuthorizedUser, 0, len(ps.authorized))
	for _, au := range ps.authorized {
		result = append(result, *au)
	}
	return result
}

// PendingCodes returns non-expired, unclaimed pairing codes.
func (ps *PairingStore) PendingCodes() []PairingCode {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	now := time.Now()
	var result []PairingCode
	for _, pc := range ps.codes {
		if !pc.Claimed && now.Before(pc.ExpiresAt) {
			result = append(result, *pc)
		}
	}
	return result
}

// CleanExpired removes expired codes.
func (ps *PairingStore) CleanExpired() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	now := time.Now()
	removed := 0
	for code, pc := range ps.codes {
		if now.After(pc.ExpiresAt) && !pc.Claimed {
			delete(ps.codes, code)
			removed++
		}
	}
	return removed
}

// VerifyOnMessage checks if a message sender is authorized. If the message
// text matches a pending pairing code, it claims it automatically.
func (ps *PairingStore) VerifyOnMessage(platform, userID, text string) bool {
	// First check if already authorized.
	if ps.IsAuthorized(platform, userID) {
		return true
	}

	// Try to claim the text as a pairing code.
	if ps.ClaimCode(text, platform, userID) {
		return true
	}

	return false
}
