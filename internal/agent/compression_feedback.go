package agent

import "fmt"

// CompressionFeedback holds user-facing feedback for manual context compression.
type CompressionFeedback struct {
	Noop      bool   `json:"noop"`
	Headline  string `json:"headline"`
	TokenLine string `json:"token_line"`
	Note      string `json:"note,omitempty"`
}

// SummarizeManualCompression produces consistent user-facing feedback
// for manual compression operations (e.g. /compress command).
func SummarizeManualCompression(
	beforeCount, afterCount int,
	beforeTokens, afterTokens int,
	messagesChanged bool,
) CompressionFeedback {
	noop := !messagesChanged

	var headline, tokenLine string

	if noop {
		headline = fmt.Sprintf("No changes from compression: %d messages", beforeCount)
		if afterTokens == beforeTokens {
			tokenLine = fmt.Sprintf("Rough transcript estimate: ~%d tokens (unchanged)", beforeTokens)
		} else {
			tokenLine = fmt.Sprintf("Rough transcript estimate: ~%d -> ~%d tokens", beforeTokens, afterTokens)
		}
	} else {
		headline = fmt.Sprintf("Compressed: %d -> %d messages", beforeCount, afterCount)
		tokenLine = fmt.Sprintf("Rough transcript estimate: ~%d -> ~%d tokens", beforeTokens, afterTokens)
	}

	var note string
	if !noop && afterCount < beforeCount && afterTokens > beforeTokens {
		note = "Note: fewer messages can still raise this rough transcript estimate " +
			"when compression rewrites the transcript into denser summaries."
	}

	return CompressionFeedback{
		Noop:      noop,
		Headline:  headline,
		TokenLine: tokenLine,
		Note:      note,
	}
}
