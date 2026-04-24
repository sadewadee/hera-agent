package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/internal/plugins"
)

// pluginsCmd returns the `hera plugins` subcommand tree.
//
//	hera plugins install <spec>   — clone a plugin from GitHub or full URL
//	hera plugins list             — print all installed plugins
//	hera plugins update  <name>   — git pull an installed plugin
//	hera plugins remove  <name>   — delete an installed plugin
//	hera plugins enable  <name>   — remove .disabled marker
//	hera plugins disable <name>   — add .disabled marker
func (a *App) pluginsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage third-party plugins",
		Long: `Install, update, remove, enable, and disable third-party plugins.

Plugins are git repositories cloned into $HERA_HOME/plugins/. Each plugin
may provide skills, hooks, tools, or an MCP server. Enabled plugins are
merged into the agent on the next startup.

Install by GitHub shorthand or full git URL:
  hera plugins install alice/my-plugin
  hera plugins install https://github.com/alice/my-plugin.git`,
	}

	cmd.AddCommand(pluginsInstallCmd())
	cmd.AddCommand(pluginsListCmd())
	cmd.AddCommand(pluginsUpdateCmd())
	cmd.AddCommand(pluginsRemoveCmd())
	cmd.AddCommand(pluginsEnableCmd())
	cmd.AddCommand(pluginsDisableCmd())

	return cmd
}

func pluginsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <spec>",
		Short: "Install a plugin (git clone)",
		Long: `Clone a plugin repository into $HERA_HOME/plugins/.

spec may be:
  owner/repo                       — GitHub shorthand
  https://github.com/owner/repo   — full HTTPS URL`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m := plugins.NewManager(paths.UserPlugins())
			info, err := m.Install(args[0])
			if err != nil {
				return fmt.Errorf("install plugin: %w", err)
			}
			name := info.DirName
			if info.Manifest != nil && info.Manifest.Name != "" {
				name = info.Manifest.Name + " (" + info.DirName + ")"
			}
			fmt.Fprintf(os.Stdout, "Installed plugin: %s\nPath: %s\n", name, info.Path)
			fmt.Fprintln(os.Stdout, "Restart hera to load the new plugin.")
			return nil
		},
	}
}

func pluginsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := plugins.NewManager(paths.UserPlugins())
			infos, err := m.List()
			if err != nil {
				return fmt.Errorf("list plugins: %w", err)
			}
			if len(infos) == 0 {
				fmt.Fprintln(os.Stdout, "No plugins installed.")
				fmt.Fprintln(os.Stdout, "Install one with: hera plugins install owner/repo")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tDESCRIPTION")
			for _, info := range infos {
				name := info.DirName
				version := "-"
				description := "-"
				status := "enabled"
				if !info.Enabled {
					status = "disabled"
				}
				if info.Manifest != nil {
					if info.Manifest.Name != "" {
						name = info.Manifest.Name
					}
					if info.Manifest.Version != "" {
						version = info.Manifest.Version
					}
					if info.Manifest.Description != "" {
						description = info.Manifest.Description
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, version, status, description)
			}
			return w.Flush()
		},
	}
}

func pluginsUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <name>",
		Short: "Update an installed plugin (git pull)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m := plugins.NewManager(paths.UserPlugins())
			info, err := m.Update(args[0])
			if err != nil {
				return fmt.Errorf("update plugin: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Updated plugin: %s\n", info.DirName)
			fmt.Fprintln(os.Stdout, "Restart hera to apply changes.")
			return nil
		},
	}
}

func pluginsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m := plugins.NewManager(paths.UserPlugins())
			if err := m.Remove(args[0]); err != nil {
				return fmt.Errorf("remove plugin: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Removed plugin: %s\n", args[0])
			return nil
		},
	}
}

func pluginsEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable a disabled plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m := plugins.NewManager(paths.UserPlugins())
			if err := m.Enable(args[0]); err != nil {
				return fmt.Errorf("enable plugin: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Enabled plugin: %s\n", args[0])
			fmt.Fprintln(os.Stdout, "Restart hera to load the plugin.")
			return nil
		},
	}
}

func pluginsDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable a plugin without removing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m := plugins.NewManager(paths.UserPlugins())
			if err := m.Disable(args[0]); err != nil {
				return fmt.Errorf("disable plugin: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Disabled plugin: %s\n", args[0])
			return nil
		},
	}
}
