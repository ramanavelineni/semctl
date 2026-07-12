package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to a Semaphore server",
	Long: `Authenticate with a Semaphore server, create an API token, and save the
server context to the config file.

The password is used once to obtain an API token and is NOT stored unless
--save-password is given. Prefer --password-stdin over --password to keep
the password out of shell history and process listings.`,
	Example: `  semctl login
  semctl login --server 10.0.0.1:3000 --username admin --password-stdin < pass.txt
  echo "$SEM_PASS" | semctl login --server sem.example.com:443 --scheme https --username admin --password-stdin
  semctl login --server 10.0.0.1:3000 --username admin --context prod`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		scheme, _ := cmd.Flags().GetString("scheme")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		passwordStdin, _ := cmd.Flags().GetBool("password-stdin")
		savePassword, _ := cmd.Flags().GetBool("save-password")
		contextName, _ := cmd.Flags().GetString("context")

		if password != "" && passwordStdin {
			return fmt.Errorf("cannot use --password and --password-stdin together")
		}
		if passwordStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			password = strings.TrimRight(string(data), "\r\n")
			if password == "" {
				return fmt.Errorf("empty password on stdin")
			}
		}

		// Interactive mode: prompt for missing values
		inputsMissing := server == "" || username == "" || password == ""
		interactive, err := shouldAutoInteractive(cmd, inputsMissing)
		if err != nil {
			return err
		}
		if interactive && !passwordStdin {
			if contextName == "" {
				contextName = "default"
			}

			form := newForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Server").
						Description("host:port (e.g. 10.0.0.1:3000)").
						Value(&server).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("server is required")
							}
							return nil
						}),
					huh.NewSelect[string]().
						Title("Scheme").
						Options(
							huh.NewOption("http", "http"),
							huh.NewOption("https", "https"),
						).
						Value(&scheme),
				).Title("Server"),

				huh.NewGroup(
					huh.NewInput().
						Title("Username").
						Value(&username).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("username is required")
							}
							return nil
						}),
					huh.NewInput().
						Title("Password").
						EchoMode(huh.EchoModePassword).
						Value(&password).
						Validate(func(s string) error {
							if s == "" {
								return fmt.Errorf("password is required")
							}
							return nil
						}),
				).Title("Authentication"),

				huh.NewGroup(
					huh.NewInput().
						Title("Context name").
						Description("Name for this server context (e.g. production, staging)").
						Value(&contextName),
				).Title("Options"),
			)

			if err := runForm(form); err != nil {
				return err
			}
		}

		if server == "" || username == "" || password == "" {
			return fmt.Errorf("--server, --username, and a password (--password-stdin or --password) are required in non-interactive mode")
		}

		// Default context name
		if contextName == "" {
			contextName = "default"
		}
		if err := config.ValidateContextName(contextName); err != nil {
			return err
		}

		// Parse host and port from server string
		host, port, err := config.ParseHostPort(server)
		if err != nil {
			return err
		}

		client.WarnIfPlaintext(scheme, host)

		// Build API URL
		serverURL := fmt.Sprintf("%s://%s:%d/api", scheme, host, port)

		// Attempt to authenticate: login → create API token
		token, err := client.LoginAndCreateToken(serverURL, username, password)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Cache the token for this context, bound to the server that issued it
		if err := client.SaveTokenCacheForContext(contextName, client.ServerID(scheme, host, port), token); err != nil {
			style.Warning(fmt.Sprintf("Failed to cache API token: %s", err))
		}

		// Load config (login skips PersistentPreRunE)
		cfgFile, _ := cmd.Flags().GetString("config")
		_ = config.Load(cfgFile)

		// Save context config. The password is only persisted with --save-password;
		// the cached API token handles subsequent authentication.
		serverData := map[string]interface{}{
			"host":   host,
			"port":   port,
			"scheme": scheme,
		}
		authData := map[string]interface{}{
			"username": username,
		}
		if savePassword {
			authData["password"] = password
		}

		if err := config.SaveContext(contextName, serverData, authData); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Set as current context
		if err := config.SetCurrentContext(contextName); err != nil {
			return fmt.Errorf("failed to set current context: %w", err)
		}

		style.Success(fmt.Sprintf("Logged in to %s://%s:%d as %s (context: %s)", scheme, host, port, username, contextName))
		style.Info(fmt.Sprintf("Config saved to %s", config.ConfigFilePath()))
		if !savePassword {
			style.Info("Password not stored (API token cached). Re-run 'semctl login' if the token is revoked, or use --save-password to enable automatic re-login.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// --server and --context are inherited from the root persistent flags.
	// Defining local ones here shadows them and breaks the -s shorthand.
	loginCmd.Flags().String("scheme", "http", "server scheme (http or https)")
	loginCmd.Flags().String("username", "", "auth username")
	loginCmd.Flags().String("password", "", "auth password (prefer --password-stdin)")
	loginCmd.Flags().Bool("password-stdin", false, "read the password from stdin")
	loginCmd.Flags().Bool("save-password", false, "store the password in the config file for automatic re-login")
}
