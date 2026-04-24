package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sadewadee/hera/internal/paths"
)

// registerConfigCommands adds configuration management commands.
func registerConfigCommands(rootCmd *cobra.Command) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Hera configuration",
	}

	configCmd.AddCommand(configShowCmd())
	configCmd.AddCommand(configSetCmd())
	configCmd.AddCommand(configGetCmd())
	configCmd.AddCommand(configPathCmd())
	configCmd.AddCommand(configInitCmd())

	rootCmd.AddCommand(configCmd)
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings := viper.AllSettings()
			out, err := json.MarshalIndent(settings, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			fmt.Println(string(out))
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set(args[0], args[1])
			if err := viper.WriteConfig(); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			fmt.Printf("Set %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val := viper.Get(args[0])
			if val == nil {
				fmt.Printf("%s: (not set)\n", args[0])
			} else {
				fmt.Printf("%s: %v\n", args[0], val)
			}
			return nil
		},
	}
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Display the configuration file path",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile := viper.ConfigFileUsed()
			if cfgFile == "" {
				cfgFile = paths.UserConfig()
			}
			fmt.Println(cfgFile)
		},
	}
}

func configInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a default configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := paths.HeraHome()
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}

			configPath := filepath.Join(configDir, "config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				fmt.Printf("Config already exists: %s\n", configPath)
				return nil
			}

			defaultConfig := `# Hera Configuration
agent:
  provider: openai
  model: gpt-4
  max_turns: 50
  temperature: 0.7

security:
  dangerous_approve: false

gateway:
  human_delay: false
`
			if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			fmt.Printf("Config created: %s\n", configPath)
			return nil
		},
	}
}
