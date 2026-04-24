package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFact_Fields(t *testing.T) {
	now := time.Now()
	f := Fact{
		ID:        "f1",
		UserID:    "user1",
		Key:       "name",
		Value:     "Alice",
		CreatedAt: now,
		UpdatedAt: now,
	}
	assert.Equal(t, "f1", f.ID)
	assert.Equal(t, "user1", f.UserID)
	assert.Equal(t, "name", f.Key)
	assert.Equal(t, "Alice", f.Value)
	assert.Equal(t, now, f.CreatedAt)
	assert.Equal(t, now, f.UpdatedAt)
}

func TestMemoryResult_Fields(t *testing.T) {
	mr := MemoryResult{
		Content:  "User likes Go programming",
		Source:   "fact",
		SourceID: "f1",
		Score:    0.95,
	}
	assert.Equal(t, "User likes Go programming", mr.Content)
	assert.Equal(t, "fact", mr.Source)
	assert.Equal(t, "f1", mr.SourceID)
	assert.InDelta(t, 0.95, mr.Score, 0.001)
}

func TestSummary_Fields(t *testing.T) {
	now := time.Now()
	s := Summary{
		ID:        "s1",
		SessionID: "session1",
		Content:   "Discussion about Go testing",
		Model:     "gpt-4o",
		CreatedAt: now,
	}
	assert.Equal(t, "s1", s.ID)
	assert.Equal(t, "session1", s.SessionID)
	assert.Equal(t, "Discussion about Go testing", s.Content)
	assert.Equal(t, "gpt-4o", s.Model)
}

func TestSessionResult_Fields(t *testing.T) {
	now := time.Now()
	sr := SessionResult{
		SessionID: "sess1",
		Preview:   "Talked about testing",
		Score:     0.8,
		CreatedAt: now,
	}
	assert.Equal(t, "sess1", sr.SessionID)
	assert.InDelta(t, 0.8, sr.Score, 0.001)
}

func TestSearchOpts_Defaults(t *testing.T) {
	opts := SearchOpts{}
	assert.Equal(t, 0, opts.Limit)
	assert.Empty(t, opts.Source)
	assert.Empty(t, opts.UserID)
}

func TestSearchOpts_WithValues(t *testing.T) {
	opts := SearchOpts{
		Limit:  10,
		Source: "fact",
		UserID: "user1",
	}
	assert.Equal(t, 10, opts.Limit)
	assert.Equal(t, "fact", opts.Source)
	assert.Equal(t, "user1", opts.UserID)
}
