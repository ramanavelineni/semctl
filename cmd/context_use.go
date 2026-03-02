package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var contextUseCmd = &cobra.Command{
	Use:     "use [name]",
	Short:   "Switch to a different context",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl context use production",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Verify context exists
		contexts := config.ListContexts()
		found := false
		for _, c := range contexts {
			if c == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("context %q not found (available: %v)", name, contexts)
		}

		if err := config.SetCurrentContext(name); err != nil {
			return fmt.Errorf("failed to switch context: %w", err)
		}

		style.Success(fmt.Sprintf("Switched to context %q", name))
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextUseCmd)
}
