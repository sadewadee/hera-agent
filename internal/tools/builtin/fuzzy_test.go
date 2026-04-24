package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuzzyTool_Name(t *testing.T) {
	tool := &FuzzyTool{}
	assert.Equal(t, "fuzzy", tool.Name())
}

func TestFuzzyTool_Description(t *testing.T) {
	tool := &FuzzyTool{}
	assert.Contains(t, tool.Description(), "fuzzy")
}

func TestFuzzyTool_InvalidArgs(t *testing.T) {
	tool := &FuzzyTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestFuzzyTool_MatchFound(t *testing.T) {
	tool := &FuzzyTool{}
	args, _ := json.Marshal(fuzzyArgs{
		Query: "app",
		Items: []string{"application", "apple", "banana", "grape"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "application")
	assert.Contains(t, result.Content, "apple")
	assert.Contains(t, result.Content, "Matches (2)")
}

func TestFuzzyTool_NoMatches(t *testing.T) {
	tool := &FuzzyTool{}
	args, _ := json.Marshal(fuzzyArgs{
		Query: "xyz",
		Items: []string{"alpha", "beta", "gamma"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "No matches")
}

func TestFuzzyTool_CaseInsensitive(t *testing.T) {
	tool := &FuzzyTool{}
	args, _ := json.Marshal(fuzzyArgs{
		Query: "ABC",
		Items: []string{"abcdef", "XYZ"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "abcdef")
}

func TestRegisterFuzzy(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterFuzzy(registry)
	_, ok := registry.Get("fuzzy")
	assert.True(t, ok)
}
