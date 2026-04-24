package batch

import (
	"fmt"

	"github.com/sadewadee/hera/internal/agent"
	"github.com/sadewadee/hera/internal/llm"
)

// EstimateResult holds the output of a cost estimation pass.
type EstimateResult struct {
	TotalPrompts   int
	TotalTokensIn  int
	TotalTokensOut int // estimated as 2× input tokens (conservative)
	CostUSD        float64
	ModelID        string
	Provider       string
}

// String returns a human-readable summary of the estimate.
func (e EstimateResult) String() string {
	return fmt.Sprintf(
		"model=%s provider=%s prompts=%d tokens_in=%d tokens_out_est=%d cost_usd=%.4f",
		e.ModelID, e.Provider, e.TotalPrompts, e.TotalTokensIn, e.TotalTokensOut, e.CostUSD,
	)
}

// Estimate calculates approximate token counts and cost for a slice of prompts
// using the given provider's model metadata.
//
// Token counting uses the agent.EstimateTokens heuristic (4 chars ≈ 1 token).
// Output tokens are estimated as 2× input tokens — adjust via EstimateWithRatio.
// Cost uses provider.ModelInfo().CostPer1kIn and CostPer1kOut.
func Estimate(prompts []string, provider llm.Provider) EstimateResult {
	return EstimateWithRatio(prompts, provider, 2.0)
}

// EstimateWithRatio is like Estimate but lets the caller specify the
// output/input token ratio (e.g. 1.5 for shorter responses, 3.0 for long).
func EstimateWithRatio(prompts []string, provider llm.Provider, outRatio float64) EstimateResult {
	meta := provider.ModelInfo()

	tokensIn := 0
	for _, p := range prompts {
		tokensIn += agent.EstimateTokens(p)
	}
	tokensOut := int(float64(tokensIn) * outRatio)

	costIn := float64(tokensIn) / 1000.0 * meta.CostPer1kIn
	costOut := float64(tokensOut) / 1000.0 * meta.CostPer1kOut
	totalCost := costIn + costOut

	return EstimateResult{
		TotalPrompts:   len(prompts),
		TotalTokensIn:  tokensIn,
		TotalTokensOut: tokensOut,
		CostUSD:        totalCost,
		ModelID:        meta.ID,
		Provider:       meta.Provider,
	}
}
