package batch

import (
	"context"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
)

// fakeProvider is a minimal llm.Provider for estimate tests.
type fakeProvider struct {
	meta llm.ModelMetadata
}

func (f *fakeProvider) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, nil
}
func (f *fakeProvider) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	return nil, nil
}
func (f *fakeProvider) CountTokens(_ []llm.Message) (int, error) {
	return 0, nil
}
func (f *fakeProvider) ModelInfo() llm.ModelMetadata {
	return f.meta
}

func TestEstimate_TokenCount(t *testing.T) {
	provider := &fakeProvider{
		meta: llm.ModelMetadata{
			ID:           "gpt-4o",
			Provider:     "openai",
			CostPer1kIn:  0.005,
			CostPer1kOut: 0.015,
		},
	}

	// 4 chars per token: "hello" = 1 token, "hello world" = 2 tokens
	prompts := []string{
		"abcd",     // 1 token
		"abcdefgh", // 2 tokens
	}
	result := Estimate(prompts, provider)

	if result.TotalPrompts != 2 {
		t.Errorf("TotalPrompts = %d, want 2", result.TotalPrompts)
	}
	if result.TotalTokensIn != 3 {
		t.Errorf("TotalTokensIn = %d, want 3", result.TotalTokensIn)
	}
	// Output = 2x input = 6
	if result.TotalTokensOut != 6 {
		t.Errorf("TotalTokensOut = %d, want 6", result.TotalTokensOut)
	}
}

func TestEstimate_Cost(t *testing.T) {
	provider := &fakeProvider{
		meta: llm.ModelMetadata{
			ID:           "gpt-4o",
			Provider:     "openai",
			CostPer1kIn:  1.0, // $1 per 1k tokens in
			CostPer1kOut: 2.0, // $2 per 1k tokens out
		},
	}
	// "abcdefgh" = 2 tokens in, 4 tokens out (ratio 2x)
	prompts := []string{"abcdefgh"}
	result := Estimate(prompts, provider)

	// costIn = 2/1000 * 1.0 = 0.002
	// costOut = 4/1000 * 2.0 = 0.008
	// total = 0.010
	if result.CostUSD < 0.0099 || result.CostUSD > 0.0101 {
		t.Errorf("CostUSD = %.6f, want ~0.010", result.CostUSD)
	}
}

func TestEstimate_Empty(t *testing.T) {
	provider := &fakeProvider{meta: llm.ModelMetadata{ID: "m", Provider: "p"}}
	result := Estimate(nil, provider)
	if result.TotalPrompts != 0 || result.TotalTokensIn != 0 || result.CostUSD != 0 {
		t.Errorf("empty prompts should produce zero result: %+v", result)
	}
}

func TestEstimateWithRatio(t *testing.T) {
	provider := &fakeProvider{
		meta: llm.ModelMetadata{
			ID:           "m",
			Provider:     "p",
			CostPer1kIn:  0.0,
			CostPer1kOut: 0.0,
		},
	}
	prompts := []string{"abcdefgh"} // 2 tokens
	result := EstimateWithRatio(prompts, provider, 3.0)
	if result.TotalTokensOut != 6 {
		t.Errorf("TotalTokensOut = %d, want 6 (2*3)", result.TotalTokensOut)
	}
}

func TestEstimateResult_String(t *testing.T) {
	r := EstimateResult{
		TotalPrompts:   5,
		TotalTokensIn:  100,
		TotalTokensOut: 200,
		CostUSD:        0.123,
		ModelID:        "gpt-4o",
		Provider:       "openai",
	}
	s := r.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}
