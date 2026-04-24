package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// registerCompletionsCommand adds shell completion generation commands.
func registerCompletionsCommand(rootCmd *cobra.Command) {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for hera.

To load completions:

Bash:
  $ source <(hera completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ hera completion bash > /etc/bash_completion.d/hera
  # macOS:
  $ hera completion bash > $(brew --prefix)/etc/bash_completion.d/hera

Zsh:
  $ source <(hera completion zsh)
  # To load completions for each session:
  $ hera completion zsh > "${fpath[1]}/_hera"

Fish:
  $ hera completion fish | source
  # To load completions for each session:
  $ hera completion fish > ~/.config/fish/completions/hera.fish

PowerShell:
  PS> hera completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	rootCmd.AddCommand(completionCmd)
}
