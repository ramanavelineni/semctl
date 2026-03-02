package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove saved credentials",
	Long:  `Remove saved credentials from the config file and delete the cached API token for the current (or specified) context.`,
	Example: `  semctl logout
  semctl -y logout
  semctl logout --context prod
  semctl logout -I`,
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive, err := shouldAutoInteractive(cmd, false)
		if err != nil {
			return err
		}
		autoConfirm, _ := cmd.Flags().GetBool("yes")
		contextFlag, _ := cmd.Flags().GetString("context")

		// Load config (logout skips PersistentPreRunE)
		cfgFile, _ := cmd.Flags().GetString("config")
		_ = config.Load(cfgFile)

		targetContext := contextFlag
		if targetContext == "" {
			targetContext = config.GetCurrentContext()
		}

		promptMsg := fmt.Sprintf("Log out of context %q and remove saved credentials?", targetContext)

		if interactive {
			var confirm bool
			if err := newForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Log out?").
						Description(fmt.Sprintf("This will remove credentials and cached token for context %q.", targetContext)).
						Value(&confirm),
				),
			).Run(); err != nil {
				return err
			}
			if !confirm {
				style.Info("Cancelled.")
				return nil
			}
		} else if !autoConfirm {
			fmt.Printf("%s [y/N] ", promptMsg)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				style.Info("Cancelled.")
				return nil
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
