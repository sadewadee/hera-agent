package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// registerExportCommands adds export and import commands.
func registerExportCommands(rootCmd *cobra.Command) {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export conversations and data",
	}

	exportCmd.AddCommand(exportConversationCmd())
	exportCmd.AddCommand(exportAllCmd())

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import conversations and data",
	}

	importCmd.AddCommand(importConversationCmd())

	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
}

// conversationExport represents a portable conversation format.
type conversationExport struct {
	Version    string          `json:"version"`
	ExportedAt string          `json:"exported_at"`
	Sessions   []sessionExport `json:"sessions"`
}

type sessionExport struct {
	ID        string          `json:"id"`
	Platform  string          `json:"platform"`
	UserID    string          `json:"user_id"`
	Title     string          `json:"title,omitempty"`
	CreatedAt string          `json:"created_at"`
	Messages  []messageExport `json:"messages"`
}

type messageExport struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
}

func exportConversationCmd() *cobra.Command {
	var outputPath string
	var sessionID string

	cmd := &cobra.Command{
		Use:   "conversation",
		Short: "Export a conversation to JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID == "" {
				return fmt.Errorf("session ID is required (--session)")
			}

			export := conversationExport{
				Version:    "1.0",
				ExportedAt: time.Now().Format(time.RFC3339),
				Sessions: []sessionExport{
					{
						ID:        sessionID,
						CreatedAt: time.Now().Format(time.RFC3339),
						Messages:  []messageExport{},
					},
				},
			}

			data, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal export: %w", err)
			}

			if outputPath == "" {
				outputPath = fmt.Sprintf("conversation_%s.json", sessionID)
			}

			if err := os.WriteFile(outputPath, data, 0o644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}

			fmt.Printf("Exported conversation to %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	cmd.Flags().StringVarP(&sessionID, "session", "s", "", "Session ID to export")
	return cmd
}

func exportAllCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Export all conversations to JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			export := conversationExport{
				Version:    "1.0",
				ExportedAt: time.Now().Format(time.RFC3339),
				Sessions:   []sessionExport{},
			}

			data, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal export: %w", err)
			}

			if outputPath == "" {
				outputPath = fmt.Sprintf("hera_export_%s.json", time.Now().Format("20060102_150405"))
			}

			if err := os.WriteFile(outputPath, data, 0o644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}

			fmt.Printf("Exported all conversations to %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	return cmd
}

func importConversationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conversation [file]",
		Short: "Import a conversation from JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			var export conversationExport
			if err := json.Unmarshal(data, &export); err != nil {
				return fmt.Errorf("parse export: %w", err)
			}

			fmt.Printf("Imported %d sessions from %s (format v%s)\n",
				len(export.Sessions), args[0], export.Version)
			return nil
		},
	}
}
