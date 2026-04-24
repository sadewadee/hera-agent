// Package environments provides execution environment implementations.
//
// trajectory_compressor.go post-processes completed agent trajectories
// to compress them within a target token budget while preserving
// training signal quality. Compression strategy:
//
//  1. Protect first turns (system, human, first gpt, first tool)
//  2. Protect last N turns (final actions and conclusions)
//  3. Compress MIDDLE turns only, starting from 2nd tool response
//  4. Replace compressed region with a single summary message
//  5. Keep remaining tool calls intact
package environments

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CompressionConfig configures trajectory compression.
type CompressionConfig struct {
	// Compression targets.
	TargetMaxTokens     int `json:"target_max_tokens" yaml:"target_max_tokens"`
	SummaryTargetTokens int `json:"summary_target_tokens" yaml:"summary_target_tokens"`

	// Protected turns.
	ProtectFirstSystem bool `json:"protect_first_system" yaml:"protect_first_system"`
	ProtectFirstHuman  bool `json:"protect_first_human" yaml:"protect_first_human"`
	ProtectFirstGPT    bool `json:"protect_first_gpt" yaml:"protect_first_gpt"`
	ProtectFirstTool   bool `json:"protect_first_tool" yaml:"protect_first_tool"`
	ProtectLastNTurns  int  `json:"protect_last_n_turns" yaml:"protect_last_n_turns"`

	// Output.
	AddSummaryNotice  bool   `json:"add_summary_notice" yaml:"add_summary_notice"`
	SummaryNoticeText string `json:"summary_notice_text" yaml:"summary_notice_text"`
	OutputSuffix      string `json:"output_suffix" yaml:"output_suffix"`

	// Processing.
	SkipUnderTarget bool `json:"skip_under_target" yaml:"skip_under_target"`
	SaveOverLimit   bool `json:"save_over_limit" yaml:"save_over_limit"`
}

// DefaultCompressionConfig returns a sensible default configuration.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		TargetMaxTokens:     15250,
		SummaryTargetTokens: 750,
		ProtectFirstSystem:  true,
		ProtectFirstHuman:   true,
		ProtectFirstGPT:     true,
		ProtectFirstTool:    true,
		ProtectLastNTurns:   4,
		AddSummaryNotice:    true,
		SummaryNoticeText:   "\n\nSome of your previous tool responses may be summarised to preserve context.",
		OutputSuffix:        "_compressed",
		SkipUnderTarget:     true,
		SaveOverLimit:       true,
	}
}

// TrajectoryMetrics holds metrics for a single trajectory compression.
type TrajectoryMetrics struct {
	OriginalTokens     int     `json:"original_tokens"`
	CompressedTokens   int     `json:"compressed_tokens"`
	TokensSaved        int     `json:"tokens_saved"`
	CompressionRatio   float64 `json:"compression_ratio"`
	OriginalTurns      int     `json:"original_turns"`
	CompressedTurns    int     `json:"compressed_turns"`
	TurnsRemoved       int     `json:"turns_removed"`
	WasCompressed      bool    `json:"was_compressed"`
	StillOverLimit     bool    `json:"still_over_limit"`
	SkippedUnderTarget bool    `json:"skipped_under_target"`
}

// AggregateMetrics holds metrics across all compressed trajectories.
type AggregateMetrics struct {
	TotalTrajectories         int     `json:"total_trajectories"`
	TrajectoriesCompressed    int     `json:"trajectories_compressed"`
	TrajectoriesSkippedUnder  int     `json:"trajectories_skipped_under_target"`
	TrajectoriesStillOver     int     `json:"trajectories_still_over_limit"`
	TrajectoriesFailed        int     `json:"trajectories_failed"`
	TotalTokensBefore         int     `json:"total_tokens_before"`
	TotalTokensAfter          int     `json:"total_tokens_after"`
	TotalTokensSaved          int     `json:"total_tokens_saved"`
	TotalTurnsBefore          int     `json:"total_turns_before"`
	TotalTurnsAfter           int     `json:"total_turns_after"`
	TotalTurnsRemoved         int     `json:"total_turns_removed"`
	ProcessingDurationSeconds float64 `json:"processing_duration_seconds"`
}

// AddTrajectory adds a trajectory's metrics to the aggregate.
func (a *AggregateMetrics) AddTrajectory(m TrajectoryMetrics) {
	a.TotalTrajectories++
	a.TotalTokensBefore += m.OriginalTokens
	a.TotalTokensAfter += m.CompressedTokens
	a.TotalTokensSaved += m.TokensSaved
	a.TotalTurnsBefore += m.OriginalTurns
	a.TotalTurnsAfter += m.CompressedTurns
	a.TotalTurnsRemoved += m.TurnsRemoved

	if m.WasCompressed {
		a.TrajectoriesCompressed++
	}
	if m.SkippedUnderTarget {
		a.TrajectoriesSkippedUnder++
	}
	if m.StillOverLimit {
		a.TrajectoriesStillOver++
	}
}

// ConversationTurn represents a single message in a trajectory.
type ConversationTurn struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolName  string `json:"tool_name,omitempty"`
	ToolCalls []any  `json:"tool_calls,omitempty"`
}

// TrajectoryCompressor compresses agent trajectories to fit within a
// target token budget.
type TrajectoryCompressor struct {
	config    CompressionConfig
	tokenizer TokenCounter
	aggregate AggregateMetrics
}

// TokenCounter estimates token counts for text. Implementations may
// use a proper tokenizer (tiktoken, sentencepiece) or a simple heuristic.
type TokenCounter interface {
	CountTokens(text string) int
}

// SimpleTokenCounter estimates tokens using a word/character heuristic.
type SimpleTokenCounter struct{}

// CountTokens estimates token count as roughly words * 1.3.
func (c *SimpleTokenCounter) CountTokens(text string) int {
	words := len(strings.Fields(text))
	if words == 0 {
		return len(text) / 4
	}
	return int(float64(words) * 1.3)
}

// NewTrajectoryCompressor creates a new trajectory compressor.
func NewTrajectoryCompressor(cfg CompressionConfig, tokenizer TokenCounter) *TrajectoryCompressor {
	if tokenizer == nil {
		tokenizer = &SimpleTokenCounter{}
	}
	return &TrajectoryCompressor{
		config:    cfg,
		tokenizer: tokenizer,
	}
}

// CompressTrajectory compresses a single trajectory (list of turns).
func (tc *TrajectoryCompressor) CompressTrajectory(turns []ConversationTurn) ([]ConversationTurn, TrajectoryMetrics) {
	metrics := TrajectoryMetrics{
		OriginalTurns: len(turns),
	}

	// Count original tokens.
	for _, t := range turns {
		metrics.OriginalTokens += tc.tokenizer.CountTokens(t.Content)
	}

	// Check if compression is needed.
	if tc.config.SkipUnderTarget && metrics.OriginalTokens <= tc.config.TargetMaxTokens {
		metrics.CompressedTokens = metrics.OriginalTokens
		metrics.CompressedTurns = len(turns)
		metrics.CompressionRatio = 1.0
		metrics.SkippedUnderTarget = true
		return turns, metrics
	}

	// Identify protected regions.
	headProtected := tc.identifyHeadProtected(turns)
	tailProtected := len(turns) - tc.config.ProtectLastNTurns
	if tailProtected < headProtected {
		tailProtected = headProtected
	}

	// Count tokens in protected regions.
	headTokens := 0
	for i := 0; i < headProtected && i < len(turns); i++ {
		headTokens += tc.tokenizer.CountTokens(turns[i].Content)
	}
	tailTokens := 0
	for i := tailProtected; i < len(turns); i++ {
		tailTokens += tc.tokenizer.CountTokens(turns[i].Content)
	}

	// Calculate how many tokens we need to remove from the middle.
	budget := tc.config.TargetMaxTokens - headTokens - tailTokens - tc.config.SummaryTargetTokens
	if budget < 0 {
		budget = 0
	}

	// Compress middle region: keep turns from the end of the middle
	// that fit in the remaining budget.
	middleTurns := turns[headProtected:tailProtected]
	var compressEnd int
	middleTokensAccum := 0
	for i := len(middleTurns) - 1; i >= 0; i-- {
		tokens := tc.tokenizer.CountTokens(middleTurns[i].Content)
		if middleTokensAccum+tokens > budget {
			compressEnd = i + 1
			break
		}
		middleTokensAccum += tokens
	}

	if compressEnd <= 0 {
		// Nothing to compress.
		metrics.CompressedTokens = metrics.OriginalTokens
		metrics.CompressedTurns = len(turns)
		metrics.CompressionRatio = 1.0
		return turns, metrics
	}

	// Build compressed trajectory.
	compressed := make([]ConversationTurn, 0, len(turns))
	compressed = append(compressed, turns[:headProtected]...)

	// Add summary placeholder for compressed region.
	compressedRegion := middleTurns[:compressEnd]
	summary := tc.buildSummaryMessage(compressedRegion)
	compressed = append(compressed, summary)

	// Add remaining middle turns.
	if compressEnd < len(middleTurns) {
		compressed = append(compressed, middleTurns[compressEnd:]...)
	}

	// Add tail turns.
	compressed = append(compressed, turns[tailProtected:]...)

	// Calculate metrics.
	metrics.WasCompressed = true
	metrics.CompressedTurns = len(compressed)
	metrics.TurnsRemoved = len(turns) - len(compressed)
	for _, t := range compressed {
		metrics.CompressedTokens += tc.tokenizer.CountTokens(t.Content)
	}
	metrics.TokensSaved = metrics.OriginalTokens - metrics.CompressedTokens
	if metrics.OriginalTokens > 0 {
		metrics.CompressionRatio = float64(metrics.CompressedTokens) / float64(metrics.OriginalTokens)
	}
	metrics.StillOverLimit = metrics.CompressedTokens > tc.config.TargetMaxTokens

	tc.aggregate.AddTrajectory(metrics)
	return compressed, metrics
}

// identifyHeadProtected determines how many turns at the start are protected.
func (tc *TrajectoryCompressor) identifyHeadProtected(turns []ConversationTurn) int {
	protected := 0
	seenSystem := false
	seenHuman := false
	seenGPT := false
	seenTool := false

	for _, t := range turns {
		shouldProtect := false
		switch t.Role {
		case "system":
			if tc.config.ProtectFirstSystem && !seenSystem {
				shouldProtect = true
				seenSystem = true
			}
		case "user", "human":
			if tc.config.ProtectFirstHuman && !seenHuman {
				shouldProtect = true
				seenHuman = true
			}
		case "assistant", "gpt":
			if tc.config.ProtectFirstGPT && !seenGPT {
				shouldProtect = true
				seenGPT = true
			}
		case "tool":
			if tc.config.ProtectFirstTool && !seenTool {
				shouldProtect = true
				seenTool = true
			}
		}

		if shouldProtect {
			protected++
		} else {
			break
		}
	}
	return protected
}

// buildSummaryMessage creates a placeholder summary for a compressed region.
func (tc *TrajectoryCompressor) buildSummaryMessage(turns []ConversationTurn) ConversationTurn {
	var toolNames []string
	seen := make(map[string]bool)
	for _, t := range turns {
		if t.ToolName != "" && !seen[t.ToolName] {
			toolNames = append(toolNames, t.ToolName)
			seen[t.ToolName] = true
		}
	}

	summary := fmt.Sprintf("[%d conversation turns compressed to save context. ", len(turns))
	if len(toolNames) > 0 {
		summary += fmt.Sprintf("Tools used: %s. ", strings.Join(toolNames, ", "))
	}
	summary += "The agent continued working on the task.]"

	if tc.config.AddSummaryNotice {
		summary += tc.config.SummaryNoticeText
	}

	return ConversationTurn{
		Role:    "user",
		Content: summary,
	}
}

// GetAggregateMetrics returns the accumulated compression metrics.
func (tc *TrajectoryCompressor) GetAggregateMetrics() AggregateMetrics {
	return tc.aggregate
}

// CompressFile compresses all trajectories in a JSONL file.
func (tc *TrajectoryCompressor) CompressFile(inputPath, outputPath string) error {
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + tc.config.OutputSuffix + ext
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	start := time.Now()

	var outputLines []string
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var turns []ConversationTurn
		if err := json.Unmarshal([]byte(line), &turns); err != nil {
			slog.Warn("failed to parse trajectory line",
				"line", i+1,
				"error", err,
			)
			outputLines = append(outputLines, line)
			continue
		}

		compressed, metrics := tc.CompressTrajectory(turns)
		slog.Debug("compressed trajectory",
			"line", i+1,
			"original_tokens", metrics.OriginalTokens,
			"compressed_tokens", metrics.CompressedTokens,
			"ratio", fmt.Sprintf("%.2f", metrics.CompressionRatio),
		)

		outData, err := json.Marshal(compressed)
		if err != nil {
			slog.Warn("failed to marshal compressed trajectory", "line", i+1, "error", err)
			outputLines = append(outputLines, line)
			continue
		}
		outputLines = append(outputLines, string(outData))
	}

	tc.aggregate.ProcessingDurationSeconds = time.Since(start).Seconds()

	output := strings.Join(outputLines, "\n") + "\n"
	if err := os.WriteFile(outputPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	slog.Info("compression complete",
		"input", inputPath,
		"output", outputPath,
		"trajectories", tc.aggregate.TotalTrajectories,
		"compressed", tc.aggregate.TrajectoriesCompressed,
		"duration", fmt.Sprintf("%.1fs", tc.aggregate.ProcessingDurationSeconds),
	)
	return nil
}
