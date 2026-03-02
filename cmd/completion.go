package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for semctl.

To load completions:

Bash:
  $ source <(semctl completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ semctl completion bash > /etc/bash_completion.d/semctl
  # macOS:
  $ semctl completion bash > $(brew --prefix)/etc/bash_completion.d/semctl

Zsh:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  $ semctl completion zsh > "${fpath[1]}/_semctl"

Fish:
  $ semctl completion fish > ~/.config/fish/completions/semctl.fish

PowerShell:
  PS> semctl completion powershell | Out-String | Invoke-Expression
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
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
