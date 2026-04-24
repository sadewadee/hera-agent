package builtin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/tools"
)

// DatabaseTool executes queries against SQLite databases.
type DatabaseTool struct{}

type databaseToolArgs struct {
	Action string `json:"action"`
	Path   string `json:"path"`
	Query  string `json:"query,omitempty"`
	Params []any  `json:"params,omitempty"`
}

func (t *DatabaseTool) Name() string { return "database" }

func (t *DatabaseTool) Description() string {
	return "Executes SQL queries against SQLite databases. Supports query, exec, and schema inspection."
}

func (t *DatabaseTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["query", "exec", "tables", "schema"],
				"description": "Database action: query (SELECT), exec (INSERT/UPDATE/DELETE), tables (list tables), schema (show CREATE statements)."
			},
			"path": {
				"type": "string",
				"description": "Path to SQLite database file."
			},
			"query": {
				"type": "string",
				"description": "SQL query to execute."
			},
			"params": {
				"type": "array",
				"items": {},
				"description": "Query parameters for parameterized queries."
			}
		},
		"required": ["action", "path"]
	}`)
}

func (t *DatabaseTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a databaseToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	a.Path = paths.Normalize(a.Path)
	db, err := sql.Open("sqlite", a.Path)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("open database: %v", err), IsError: true}, nil
	}
	defer db.Close()

	switch a.Action {
	case "query":
		if a.Query == "" {
			return &tools.Result{Content: "query is required", IsError: true}, nil
		}
		return dbQuery(ctx, db, a.Query, a.Params)

	case "exec":
		if a.Query == "" {
			return &tools.Result{Content: "query is required", IsError: true}, nil
		}
		return dbExec(ctx, db, a.Query, a.Params)

	case "tables":
		return dbQuery(ctx, db, "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name", nil)

	case "schema":
		return dbQuery(ctx, db, "SELECT sql FROM sqlite_master WHERE sql IS NOT NULL ORDER BY name", nil)

	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

func dbQuery(ctx context.Context, db *sql.DB, query string, params []any) (*tools.Result, error) {
	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("query error: %v", err), IsError: true}, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("columns error: %v", err), IsError: true}, nil
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(cols, " | "))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(sb.String())-1))
	sb.WriteString("\n")

	values := make([]any, len(cols))
	scanArgs := make([]any, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	rowCount := 0
	const maxRows = 1000
	for rows.Next() {
		if rowCount >= maxRows {
			sb.WriteString(fmt.Sprintf("\n...[truncated at %d rows]", maxRows))
			break
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return &tools.Result{Content: fmt.Sprintf("scan error: %v", err), IsError: true}, nil
		}
		parts := make([]string, len(values))
		for i, v := range values {
			if v == nil {
				parts[i] = "NULL"
			} else {
				parts[i] = fmt.Sprintf("%v", v)
			}
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteString("\n")
		rowCount++
	}

	if rowCount == 0 {
		return &tools.Result{Content: "Query returned 0 rows"}, nil
	}

	return &tools.Result{Content: strings.TrimSpace(sb.String())}, nil
}

func dbExec(ctx context.Context, db *sql.DB, query string, params []any) (*tools.Result, error) {
	result, err := db.ExecContext(ctx, query, params...)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("exec error: %v", err), IsError: true}, nil
	}

	affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()

	return &tools.Result{
		Content: fmt.Sprintf("Rows affected: %d, Last insert ID: %d", affected, lastID),
	}, nil
}

// RegisterDatabase registers the database tool with the given registry.
func RegisterDatabase(registry *tools.Registry) {
	registry.Register(&DatabaseTool{})
}
