package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeManualCompression_Noop_SameTokens(t *testing.T) {
	fb := SummarizeManualCompression(10, 10, 5000, 5000, false)
	assert.True(t, fb.Noop)
	assert.Contains(t, fb.Headline, "No changes")
	assert.Contains(t, fb.Headline, "10 messages")
	assert.Contains(t, fb.TokenLine, "unchanged")
	assert.Empty(t, fb.Note)
}

func TestSummarizeManualCompression_Noop_DifferentTokens(t *testing.T) {
	fb := SummarizeManualCompression(10, 10, 5000, 4500, false)
	assert.True(t, fb.Noop)
	assert.Contains(t, fb.TokenLine, "5000")
	assert.Contains(t, fb.TokenLine, "4500")
	assert.NotContains(t, fb.TokenLine, "unchanged")
}

func TestSummarizeManualCompression_Changed(t *testing.T) {
	fb := SummarizeManualCompression(20, 5, 8000, 3000, true)
	assert.False(t, fb.Noop)
	assert.Contains(t, fb.Headline, "Compressed: 20 -> 5 messages")
	assert.Contains(t, fb.TokenLine, "8000")
	assert.Contains(t, fb.TokenLine, "3000")
	assert.Empty(t, fb.Note)
}

func TestSummarizeManualCompression_NoteWhenTokensIncrease(t *testing.T) {
	// Fewer messages but more tokens triggers an explanatory note.
	fb := SummarizeManualCompression(20, 5, 3000, 5000, true)
	assert.False(t, fb.Noop)
	assert.Contains(t, fb.Note, "fewer messages")
	assert.Contains(t, fb.Note, "denser summaries")
}

func TestSummarizeManualCompression_NoNoteWhenSameCount(t *testing.T) {
	// Same message count with more tokens should NOT trigger note.
	fb := SummarizeManualCompression(10, 10, 3000, 5000, true)
	assert.Empty(t, fb.Note)
}

func TestSummarizeManualCompression_NoNoteWhenTokensDecrease(t *testing.T) {
	fb := SummarizeManualCompression(20, 5, 8000, 3000, true)
	assert.Empty(t, fb.Note)
}
