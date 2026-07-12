package cmd

import (
	"fmt"
	"os"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var contextDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Aliases: []string{"rm"},
	Short:   "Delete a context",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl context delete staging",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		// Validate before the cache-file removal below builds a path from it.
		if err := config.ValidateContextName(name); err != nil {
			return err
		}
		if err := confirmAction(cmd, fmt.Sprintf("Delete context %q?", name)); err != nil {
			return err
		}

		// Delete cached token
		cachePath := client.TokenCachePathForContext(name)
		if _, err := os.Stat(cachePath); err == nil {
			_ = os.Remove(cachePath)
		}

		if err := config.DeleteContext(name); err != nil {
			return fmt.Errorf("failed to delete context: %w", err)
		}

		style.Success(fmt.Sprintf("Deleted context %q", name))
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextDeleteCmd)
}
