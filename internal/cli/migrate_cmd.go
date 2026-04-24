package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/sadewadee/hera/internal/paths"
)

// registerMigrateCommands adds database migration commands.
func registerMigrateCommands(rootCmd *cobra.Command) {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
	}

	migrateCmd.AddCommand(migrateUpCmd())
	migrateCmd.AddCommand(migrateStatusCmd())
	migrateCmd.AddCommand(migrateCreateCmd())

	rootCmd.AddCommand(migrateCmd)
}

func migrateUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := getDBPath()
			if err != nil {
				return err
			}

			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			// Ensure migrations table exists
			if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
				version TEXT PRIMARY KEY,
				applied_at TEXT NOT NULL
			)`); err != nil {
				return fmt.Errorf("create migrations table: %w", err)
			}

			fmt.Printf("Migrations applied to %s\n", dbPath)
			return nil
		},
	}
}

func migrateStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := getDBPath()
			if err != nil {
				return err
			}

			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			rows, err := db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
			if err != nil {
				fmt.Println("No migrations have been applied yet.")
				return nil
			}
			defer rows.Close()

			fmt.Println("Applied migrations:")
			count := 0
			for rows.Next() {
				var version, appliedAt string
				if err := rows.Scan(&version, &appliedAt); err != nil {
					return fmt.Errorf("scan row: %w", err)
				}
				fmt.Printf("  %s (applied: %s)\n", version, appliedAt)
				count++
			}
			if count == 0 {
				fmt.Println("  (none)")
			}
			return nil
		},
	}
}

func migrateCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationsDir := filepath.Join(paths.HeraHome(), "migrations")
			if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
				return fmt.Errorf("create migrations dir: %w", err)
			}

			timestamp := time.Now().Format("20060102150405")
			filename := fmt.Sprintf("%s_%s.sql", timestamp, args[0])
			path := filepath.Join(migrationsDir, filename)

			content := fmt.Sprintf(`-- Migration: %s
-- Created: %s

-- +up
-- Write your UP migration SQL here

-- +down
-- Write your DOWN migration SQL here
`, args[0], time.Now().Format(time.RFC3339))

			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write migration: %w", err)
			}

			fmt.Printf("Created migration: %s\n", path)
			return nil
		},
	}
}

func getDBPath() (string, error) {
	return paths.UserDB(), nil
}
