package tool_call_parsers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ToolCall type ---

func TestToolCallStruct(t *testing.T) {
	t.Parallel()

	tc := ToolCall{
		Name: "search",
		Args: map[string]string{"query": "hello"},
		Raw:  "raw data",
	}
	assert.Equal(t, "search", tc.Name)
	assert.Equal(t, "hello", tc.Args["query"])
	assert.Equal(t, "raw data", tc.Raw)
}

// --- JSONToolCallParser ---

func TestJSONToolCallParser_Name(t *testing.T) {
	t.Parallel()
	p := &JSONToolCallParser{}
	assert.Equal(t, "json_tool_call", p.Name())
}

func TestJSONToolCallParser_CanParse(t *testing.T) {
	t.Parallel()

	p := &JSONToolCallParser{}
	assert.True(t, p.CanParse("<tool_call>something</tool_call>"))
	assert.True(t, p.CanParse("text before <tool_call> text after"))
	assert.False(t, p.CanParse("no tool call here"))
	assert.False(t, p.CanParse(""))
}

func TestJSONToolCallParser_Parse_SingleCall(t *testing.T) {
	t.Parallel()

	p := &JSONToolCallParser{}
	text := `<tool_call>{"name":"search","arguments":{"q":"test"}}</tool_call>`
	calls, err := p.Parse(text)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].Raw, "search")
}

func TestJSONToolCallParser_Parse_MultipleCalls(t *testing.T) {
	t.Parallel()

	p := &JSONToolCallParser{}
	text := `<tool_call>first</tool_call> some text <tool_call>second</tool_call>`
	calls, err := p.Parse(text)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, "first", calls[0].Raw)
	assert.Equal(t, "second", calls[1].Raw)
}

func TestJSONToolCallParser_Parse_NoMatch(t *testing.T) {
	t.Parallel()

	p := &JSONToolCallParser{}
	calls, err := p.Parse("no tool calls")
	require.NoError(t, err)
	assert.Empty(t, calls)
}

// --- XMLParser ---

func TestXMLParser_Name(t *testing.T) {
	t.Parallel()
	p := &XMLParser{}
	assert.Equal(t, "xml", p.Name())
}

func TestXMLParser_CanParse(t *testing.T) {
	t.Parallel()

	p := &XMLParser{}
	assert.True(t, p.CanParse(`<tool_call name="search">args</tool_call>`))
	assert.True(t, p.CanParse(`<function_call name="search">args</function_call>`))
	assert.False(t, p.CanParse("no xml here"))
	assert.False(t, p.CanParse(""))
}

func TestXMLParser_Parse_SingleCall(t *testing.T) {
	t.Parallel()

	p := &XMLParser{}
	text := `<tool_call name="search">{"query":"test"}</tool_call>`
	calls, err := p.Parse(text)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "search", calls[0].Name)
	assert.Contains(t, calls[0].Raw, "search")
}

func TestXMLParser_Parse_MultipleCalls(t *testing.T) {
	t.Parallel()

	p := &XMLParser{}
	text := `<tool_call name="first">a</tool_call><tool_call name="second">b</tool_call>`
	calls, err := p.Parse(text)
	require.NoError(t, err)
	require.Len(t, calls, 2)
	assert.Equal(t, "first", calls[0].Name)
	assert.Equal(t, "second", calls[1].Name)
}

func TestXMLParser_Parse_NoMatch(t *testing.T) {
	t.Parallel()

	p := &XMLParser{}
	calls, err := p.Parse("no xml tool calls")
	require.NoError(t, err)
	assert.Empty(t, calls)
}

// --- ParseAll ---

func TestParseAll_MatchesFirstParser(t *testing.T) {
	t.Parallel()

	parsers := []ToolCallParser{
		&JSONToolCallParser{},
		&XMLParser{},
	}

	text := `<tool_call>{"name":"search"}</tool_call>`
	calls, name := ParseAll(parsers, text)
	assert.NotEmpty(t, calls)
	assert.Equal(t, "json_tool_call", name)
}

func TestParseAll_FallsThrough(t *testing.T) {
	t.Parallel()

	parsers := []ToolCallParser{
		&JSONToolCallParser{},
		&XMLParser{},
	}

	text := `<tool_call name="search">args</tool_call>`
	calls, name := ParseAll(parsers, text)
	assert.NotEmpty(t, calls)
	assert.Equal(t, "xml", name)
}

func TestParseAll_NoMatch(t *testing.T) {
	t.Parallel()

	parsers := []ToolCallParser{
		&JSONToolCallParser{},
		&XMLParser{},
	}

	calls, name := ParseAll(parsers, "plain text with no tool calls")
	assert.Nil(t, calls)
	assert.Empty(t, name)
}

func TestParseAll_EmptyParsers(t *testing.T) {
	t.Parallel()

	calls, name := ParseAll(nil, "some text")
	assert.Nil(t, calls)
	assert.Empty(t, name)
}

func TestParseAll_EmptyText(t *testing.T) {
	t.Parallel()

	parsers := []ToolCallParser{
		&JSONToolCallParser{},
		&XMLParser{},
	}

	calls, name := ParseAll(parsers, "")
	assert.Nil(t, calls)
	assert.Empty(t, name)
}
