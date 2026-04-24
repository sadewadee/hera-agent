package memory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// HealthReport is a structured summary of the memory-layer state. Emit
// as slog INFO at startup so silent misconfiguration (wrong db path,
// missing migrations, stale schema) shows up in the first few log lines
// instead of surfacing as "Hera forgets everything" days later.
type HealthReport struct {
	AbsPath       string   // resolved absolute path actually opened
	Cwd           string   // agent cwd — helps debug relative paths
	FileSize      int64    // 0 means freshly created
	Writable      bool     // could we create a temp key?
	Tables        []string // tables present in the opened DB
	MissingTables []string // required tables that are absent
	NoteCount     int
	FactCount     int
	ConvCount     int
	NotesPerUser  map[string]int
	Err           error // any error encountered during the check
}

var requiredTables = []string{
	"facts",
	"conversations",
	"summaries",
	"user_profiles",
	"memory_fts",
	"notes",
}

// HealthCheck runs a startup sanity check against an already-opened
// SQLiteProvider. It's read-only except for a one-row writability probe
// on a throwaway fact key, which it removes immediately. Callers log
// the returned report.
func (p *SQLiteProvider) HealthCheck(ctx context.Context, configuredPath string) HealthReport {
	r := HealthReport{NotesPerUser: map[string]int{}}

	abs, err := filepath.Abs(configuredPath)
	if err != nil {
		r.AbsPath = configuredPath
	} else {
		r.AbsPath = abs
	}
	if cwd, err := os.Getwd(); err == nil {
		r.Cwd = cwd
	}
	if st, err := os.Stat(r.AbsPath); err == nil {
		r.FileSize = st.Size()
	}

	// Which tables exist?
	rows, err := p.db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type IN ('table','virtual') ORDER BY name`)
	if err != nil {
		r.Err = fmt.Errorf("query sqlite_master: %w", err)
		return r
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			r.Err = fmt.Errorf("scan table name: %w", err)
			rows.Close()
			return r
		}
		r.Tables = append(r.Tables, name)
	}
	rows.Close()
	present := make(map[string]bool, len(r.Tables))
	for _, t := range r.Tables {
		present[t] = true
	}
	for _, want := range requiredTables {
		if !present[want] {
			r.MissingTables = append(r.MissingTables, want)
		}
	}

	// Writability probe: insert + delete a sentinel fact that no real
	// tool would produce. Uses the provider's own API so we respect
	// any future migrations.
	const probeKey = "__health_probe__"
	const probeUser = "__health__"
	if err := p.SaveFact(ctx, probeUser, probeKey, "ok"); err != nil {
		r.Err = fmt.Errorf("write probe: %w", err)
	} else {
		r.Writable = true
		// Best-effort cleanup. Not critical if it survives — it's
		// scoped under the __health__ user which no real user sees.
		_, _ = p.db.ExecContext(ctx, `DELETE FROM facts WHERE user_id = ? AND key = ?`, probeUser, probeKey)
	}

	// Counts — only for tables that actually exist, to avoid "no such
	// table" noise on a partially-migrated DB.
	if present["notes"] {
		_ = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes`).Scan(&r.NoteCount)
		noteRows, err := p.db.QueryContext(ctx, `SELECT user_id, COUNT(*) FROM notes GROUP BY user_id`)
		if err == nil {
			for noteRows.Next() {
				var u string
				var c int
				if err := noteRows.Scan(&u, &c); err == nil {
					r.NotesPerUser[u] = c
				}
			}
			noteRows.Close()
		}
	}
	if present["facts"] {
		_ = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facts`).Scan(&r.FactCount)
	}
	if present["conversations"] {
		_ = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations`).Scan(&r.ConvCount)
	}
	return r
}

// LogHealthReport emits an INFO line per salient field plus WARN lines
// for missing tables or writability problems. Structured so operators
// grep easily: `msg="memory healthcheck"`.
func LogHealthReport(r HealthReport) {
	slog.Info("memory healthcheck",
		"db_path", r.AbsPath,
		"cwd", r.Cwd,
		"file_size", r.FileSize,
		"writable", r.Writable,
		"tables", len(r.Tables),
		"notes", r.NoteCount,
		"facts", r.FactCount,
		"conversations", r.ConvCount,
	)
	if len(r.NotesPerUser) > 0 {
		slog.Info("memory healthcheck: notes per user", "counts", r.NotesPerUser)
	}
	if len(r.MissingTables) > 0 {
		slog.Warn("memory healthcheck: required tables missing",
			"missing", r.MissingTables,
			"hint", "schema migration likely failed; delete and re-create the db if this is a fresh install",
		)
	}
	if !r.Writable {
		slog.Warn("memory healthcheck: db not writable", "hint", "check file permissions and disk space")
	}
	if r.Err != nil {
		slog.Warn("memory healthcheck: error during probe", "err", r.Err)
	}
}
