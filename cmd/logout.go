package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove saved credentials",
	Long: `Revoke the API token server-side, delete the cached token, and remove
saved credentials from the config file for the current (or specified) context.`,
	Example: `  semctl logout
  semctl -y logout
  semctl logout --context prod
  semctl logout -I`,
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive, err := shouldAutoInteractive(cmd, false)
		if err != nil {
			return err
		}
		contextFlag, _ := cmd.Flags().GetString("context")

		// Load config (logout skips PersistentPreRunE)
		cfgFile, _ := cmd.Flags().GetString("config")
		_ = config.Load(cfgFile)

		targetContext := contextFlag
		if targetContext == "" {
			targetContext = config.GetCurrentContext()
		}

		if interactive {
			var confirm bool
			if err := newForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Log out?").
						Description(fmt.Sprintf("This will revoke the API token and remove credentials for context %q.", targetContext)).
						Value(&confirm),
				),
			).Run(); err != nil {
				return err
			}
			if !confirm {
				return errCancelled
			}
		} else {
			if err := confirmAction(cmd, fmt.Sprintf("Log out of context %q and remove saved credentials?", targetContext)); err != nil {
				return err
			}
		}

		// Revoke the token server-side before deleting the local cache, so a
		// leaked cache file cannot be used after logout.
		if token, err := client.LoadCachedTokenForContext(targetContext); err == nil && token != "" {
			serverDisplay := config.GetContextServerDisplay(targetContext)
			if serverDisplay != "" {
				if err := client.RevokeToken(serverDisplay+"/api", token); err != nil {
					style.Warning(fmt.Sprintf("Failed to revoke API token server-side: %s. The token may remain valid until it is deleted in the Semaphore UI.", err))
				} else {
					style.Info("API token revoked server-side.")
				}
			}
		}

		// Delete cached token for the target context
		cachePath := client.TokenCachePathForContext(targetContext)
		if _, err := os.Stat(cachePath); err == nil {
			if err := os.Remove(cachePath); err != nil {
				style.Warning(fmt.Sprintf("Failed to delete cached token: %s", err))
			}
		}

		// Remove the context from config
		if err := config.DeleteContext(targetContext); err != nil {
			if err := config.RemoveAuthConfig(); err != nil {
				return fmt.Errorf("failed to update config: %w", err)
			}
		}

		style.Success(fmt.Sprintf("Logged out of context %q. Credentials removed.", targetContext))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)

	logoutCmd.Flags().String("context", "", "context to log out of (default: current context)")
}
