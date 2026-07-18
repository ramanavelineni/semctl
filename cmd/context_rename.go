package cmd

import (
	"fmt"
	"os"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var contextRenameCmd = &cobra.Command{
	Use:     "rename <old-name> <new-name>",
	Short:   "Rename a context",
	Args:    cobra.ExactArgs(2),
	Example: "  semctl context rename default production",
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		if err := config.RenameContext(oldName, newName); err != nil {
			return fmt.Errorf("failed to rename context: %w", err)
		}

		// Move the cached token along with the context — otherwise the old
		// cache file orphans a still-valid token and the next command has to
		// re-login under the new name.
		oldPath, errOld := client.TokenCachePathForContext(oldName)
		newPath, errNew := client.TokenCachePathForContext(newName)
		if errOld == nil && errNew == nil {
			if _, err := os.Stat(oldPath); err == nil {
				if err := os.Rename(oldPath, newPath); err != nil {
					style.Warning(fmt.Sprintf("Failed to move cached token: %s. Run 'semctl login' to refresh it.", err))
				}
			}
		}

		style.Success(fmt.Sprintf("Renamed context %q to %q", oldName, newName))
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextRenameCmd)
}
