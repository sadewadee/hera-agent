package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslationTool_Name(t *testing.T) {
	tool := &TranslationTool{}
	assert.Equal(t, "translation", tool.Name())
}

func TestTranslationTool_Description(t *testing.T) {
	tool := &TranslationTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestTranslationTool_InvalidArgs(t *testing.T) {
	tool := &TranslationTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTranslationTool_DetectLatin(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "detect", Text: "Hello world"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Latin")
}

func TestTranslationTool_DetectCJK(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "detect", Text: "你好世界"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "CJK")
}

func TestTranslationTool_DetectCyrillic(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "detect", Text: "Привет мир"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Cyrillic")
}

func TestTranslationTool_DetectEmpty(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "detect", Text: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "empty")
}

func TestTranslationTool_CharInfo(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "info", Text: "Hello 123"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Characters:")
	assert.Contains(t, result.Content, "Uppercase:")
	assert.Contains(t, result.Content, "Digits:")
}

func TestTranslationTool_CharCount(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "char_count", Text: "Hello"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Characters: 5")
}

func TestTranslationTool_WordCount(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "word_count", Text: "one two three"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Words: 3")
}

func TestTranslationTool_UnknownAction(t *testing.T) {
	tool := &TranslationTool{}
	args, _ := json.Marshal(translationArgs{Action: "invalid", Text: "test"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRegisterTranslation(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTranslation(registry)
	_, ok := registry.Get("translation")
	assert.True(t, ok)
}
