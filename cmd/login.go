package cmd

import (
	"fmt"
	"strconv"
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
	Long:  `Authenticate with a Semaphore server and save credentials to the config file.`,
	Example: `  semctl login
  semctl login --server 10.0.0.1:3000 --username admin --password secret
  semctl login --scheme https --server sem.example.com:443 --username admin --password secret
  semctl login --server 10.0.0.1:3000 --username admin --password secret --context prod`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		scheme, _ := cmd.Flags().GetString("scheme")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		contextName, _ := cmd.Flags().GetString("context")

		// Interactive mode: prompt for missing values
		inputsMissing := server == "" || username == "" || password == ""
		interactive, err := shouldAutoInteractive(cmd, inputsMissing)
		if err != nil {
			return err
		}
		if interactive {
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

			if err := form.Run(); err != nil {
				return err
			}
		}

		// Default context name
		if contextName == "" {
			contextName = "default"
		}

		// Parse host and port from server string
		host, port, err := parseServer(server)
		if err != nil {
			return err
		}

		// Build API URL
		serverURL := fmt.Sprintf("%s://%s:%d/api", scheme, host, port)

		// Attempt to authenticate: login → create API token
		token, err := client.LoginAndCreateToken(serverURL, username, password)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Cache the token for this context
		_ = client.SaveTokenCacheForContext(contextName, token)

		// Load config (login skips PersistentPreRunE)
		cfgFile, _ := cmd.Flags().GetString("config")
		_ = config.Load(cfgFile)

		// Save context config
		serverData := map[string]interface{}{
			"host":   host,
			"port":   port,
			"scheme": scheme,
		}
		authData := map[string]interface{}{
			"username": username,
			"password": password,
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
		return nil
	},
}

// parseServer splits a "host:port" string, defaulting port to 3000 if omitted.
func parseServer(s string) (string, int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, fmt.Errorf("server is required")
	}

	if idx := strings.LastIndex(s, ":"); idx > 0 {
		host := s[:idx]
		portStr := s[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil && port > 0 {
			return host, port, nil
		}
	}

	return s, 3000, nil
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().String("server", "", "server host:port (e.g. 10.0.0.1:3000)")
	loginCmd.Flags().String("scheme", "http", "server scheme (http or https)")
	loginCmd.Flags().String("username", "", "auth username")
	loginCmd.Flags().String("password", "", "auth password")
	loginCmd.Flags().String("context", "", "context name for this login (default: \"default\")")
}
