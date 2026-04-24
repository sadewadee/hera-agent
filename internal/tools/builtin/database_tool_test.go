package builtin

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseTool_Name(t *testing.T) {
	tool := &DatabaseTool{}
	assert.Equal(t, "database", tool.Name())
}

func TestDatabaseTool_Description(t *testing.T) {
	tool := &DatabaseTool{}
	assert.Contains(t, tool.Description(), "SQL")
}

func TestDatabaseTool_InvalidArgs(t *testing.T) {
	tool := &DatabaseTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestDatabaseTool_UnknownAction(t *testing.T) {
	tool := &DatabaseTool{}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	args, _ := json.Marshal(databaseToolArgs{Action: "invalid", Path: dbPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestDatabaseTool_QueryRequiresQuery(t *testing.T) {
	tool := &DatabaseTool{}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	// Create the DB
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	db.Close()

	args, _ := json.Marshal(databaseToolArgs{Action: "query", Path: dbPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "query is required")
}

func TestDatabaseTool_Tables(t *testing.T) {
	tool := &DatabaseTool{}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	db.Close()

	args, _ := json.Marshal(databaseToolArgs{Action: "tables", Path: dbPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "users")
}

func TestDatabaseTool_QueryAndExec(t *testing.T) {
	tool := &DatabaseTool{}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	db.Close()

	// Insert via exec
	execArgs, _ := json.Marshal(databaseToolArgs{
		Action: "exec",
		Path:   dbPath,
		Query:  "INSERT INTO items (name) VALUES (?)",
		Params: []any{"test-item"},
	})
	execResult, err := tool.Execute(context.Background(), execArgs)
	require.NoError(t, err)
	assert.False(t, execResult.IsError)
	assert.Contains(t, execResult.Content, "Rows affected: 1")

	// Query
	queryArgs, _ := json.Marshal(databaseToolArgs{
		Action: "query",
		Path:   dbPath,
		Query:  "SELECT name FROM items",
	})
	queryResult, err := tool.Execute(context.Background(), queryArgs)
	require.NoError(t, err)
	assert.False(t, queryResult.IsError)
	assert.Contains(t, queryResult.Content, "test-item")
}

func TestDatabaseTool_Schema(t *testing.T) {
	tool := &DatabaseTool{}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE products (id INTEGER PRIMARY KEY, price REAL)")
	require.NoError(t, err)
	db.Close()

	args, _ := json.Marshal(databaseToolArgs{Action: "schema", Path: dbPath})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "CREATE TABLE")
}

func TestRegisterDatabase(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterDatabase(registry)
	_, ok := registry.Get("database")
	assert.True(t, ok)
}
