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

		// Revoke the token server-side before deleting local state, like
		// logout does — otherwise it stays valid with no local record.
		if token, err := client.LoadCachedTokenForContext(name); err == nil && token != "" {
			if serverDisplay := config.GetContextServerDisplay(name); serverDisplay != "" {
				if err := client.RevokeToken(serverDisplay+"/api", token); err != nil {
					style.Warning(fmt.Sprintf("Failed to revoke API token server-side: %s. Delete it in the Semaphore UI if needed.", err))
				}
			}
		}

		// Delete cached token
		if cachePath, err := client.TokenCachePathForContext(name); err == nil {
			if _, err := os.Stat(cachePath); err == nil {
				_ = os.Remove(cachePath)
			}
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
