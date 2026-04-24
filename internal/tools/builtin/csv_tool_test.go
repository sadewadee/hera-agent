package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVTool_Name(t *testing.T) {
	tool := &CSVTool{}
	assert.Equal(t, "csv", tool.Name())
}

func TestCSVTool_Description(t *testing.T) {
	tool := &CSVTool{}
	assert.Contains(t, tool.Description(), "CSV")
}

func TestCSVTool_InvalidArgs(t *testing.T) {
	tool := &CSVTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestCSVTool_ReadFromFile(t *testing.T) {
	dir := t.TempDir()
	csvFile := filepath.Join(dir, "test.csv")
	require.NoError(t, os.WriteFile(csvFile, []byte("name,age\nAlice,30\nBob,25\n"), 0o644))

	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "read", Path: csvFile})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Alice")
	assert.Contains(t, result.Content, "3 rows")
}

func TestCSVTool_ReadRequiresPath(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "read"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "path is required")
}

func TestCSVTool_Parse(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "parse", Data: "x,y\n1,2\n3,4"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "3 rows")
}

func TestCSVTool_ParseRequiresData(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "parse"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestCSVTool_ToJSON(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "to_json", Data: "name,age\nAlice,30\nBob,25"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var parsed []map[string]string
	require.NoError(t, json.Unmarshal([]byte(result.Content), &parsed))
	assert.Len(t, parsed, 2)
	assert.Equal(t, "Alice", parsed[0]["name"])
}

func TestCSVTool_ToJSONRequiresData(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "to_json"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestCSVTool_Generate(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{
		Action:  "generate",
		Headers: []string{"name", "score"},
		Rows:    [][]string{{"Alice", "95"}, {"Bob", "87"}},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "name,score")
	assert.Contains(t, result.Content, "Alice,95")
}

func TestCSVTool_GenerateToFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.csv")

	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{
		Action:  "generate",
		Headers: []string{"a", "b"},
		Rows:    [][]string{{"1", "2"}},
		Output:  outFile,
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, outFile)

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "a,b")
}

func TestCSVTool_GenerateEmpty(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "generate"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestCSVTool_CustomDelimiter(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "parse", Data: "a\tb\n1\t2", Delimiter: "\t"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "2 rows")
}

func TestCSVTool_UnknownAction(t *testing.T) {
	tool := &CSVTool{}
	args, _ := json.Marshal(csvToolArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRegisterCSV(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterCSV(registry)
	_, ok := registry.Get("csv")
	assert.True(t, ok)
}
