package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "semctl",
	Short: "Semaphore UI CLI",
	Long: `A command-line interface for managing Semaphore UI via its REST API.

Exit codes:
  0  success
  1  generic error
  2  changes pending (apply --detailed-exitcode)
  3  authentication failure
  4  resource not found
  5  cancelled by user
  6  task finished with error/stopped status (task run --wait)
  7  wait timeout expired (task run --wait-timeout)`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Session-level flags apply to every command, including those that
		// skip config loading (e.g. login).
		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			output.DisableColor()
		}
		if d, _ := cmd.Flags().GetDuration("timeout"); d > 0 {
			client.SetTimeout(d)
		}
		if insecure, _ := cmd.Flags().GetBool("insecure"); insecure {
			client.SetInsecureSkipVerify(true)
		}
		if caCert, _ := cmd.Flags().GetString("ca-cert"); caCert != "" {
			client.SetCACert(caCert)
		}
		if serverFlag, _ := cmd.Flags().GetString("server"); serverFlag != "" {
			config.SetServerOverride(serverFlag)
		}

		// Skip config loading for commands that don't need existing config
		switch cmd.Name() {
		case "completion", "version", "__complete", "login", "logout":
			return nil
		}

		// Also skip for context subcommands (they load config themselves)
		if cmd.Parent() != nil && cmd.Parent().Name() == "context" {
			return nil
		}

		cfgFile, _ := cmd.Flags().GetString("config")
		if err := config.Load(cfgFile); err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		// A config file in the working directory silently redirects commands
		// to whatever server it names — make that visible.
		if config.LoadedFromCWD() {
			style.Info(fmt.Sprintf("Using config from current directory: %s", config.ConfigFilePath()))
		}

		// Apply --context override if set
		if ctxFlag, _ := cmd.Flags().GetString("context"); ctxFlag != "" {
			if err := config.ApplyContext(ctxFlag); err != nil {
				return fmt.Errorf("context error: %w", err)
			}
		}

		// Set output format from flags
		if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
			output.SetFormat(output.FormatJSON)
		} else if yamlFlag, _ := cmd.Flags().GetBool("yaml"); yamlFlag {
			output.SetFormat(output.FormatYAML)
		} else {
			output.SetFormat(output.FormatFromConfig())
		}

		return nil
	},
}

// Execute runs the root command.
func Execute() {
	enforceSubcommands(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

// enforceSubcommands gives every parent command without its own Run a RunE
// that rejects unknown subcommands, so typos like "project lst" fail instead
// of printing help with exit 0.
func enforceSubcommands(cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		enforceSubcommands(c)
	}
	if cmd.HasSubCommands() && cmd.Run == nil && cmd.RunE == nil {
		cmd.RunE = requireSubcommand
	}
}

func requireSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2 // SuggestionsFor needs this; cobra only defaults it in its own error path
	}
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		msg += " — did you mean " + strings.Join(suggestions, " or ") + "?"
	}
	return fmt.Errorf("%s", msg)
}

// printEmptyList reports an empty collection on stderr. Empty is data, not
// an error: list commands exit 0 so scripts can chain on them.
func printEmptyList(what string) {
	style.Info(fmt.Sprintf("No %s found.", what))
}

// getProjectID resolves the project from the --project flag (numeric ID or
// project name), then the config default, or returns an error.
func getProjectID(cmd *cobra.Command) (int32, error) {
	if p, _ := cmd.Flags().GetString("project"); p != "" {
		if id, err := strconv.ParseInt(p, 10, 32); err == nil {
			if id <= 0 {
				return 0, fmt.Errorf("invalid project ID %q", p)
			}
			return int32(id), nil
		}
		return resolveProjectByName(p)
	}
	if p := config.GetDefaultProjectID(); p > 0 {
		return int32(p), nil
	}
	return 0, fmt.Errorf("project is required: use --project (ID or name) or set defaults.project_id in config")
}

// resolveProjectByName looks up a project ID by case-insensitive name.
func resolveProjectByName(name string) (int32, error) {
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return 0, err
	}
	resp, err := apiClient.Project.GetProjects(project.NewGetProjectsParams(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to list projects while resolving %q: %w", name, err)
	}
	for _, pr := range resp.GetPayload() {
		if strings.EqualFold(pr.Name, name) {
			return int32(pr.ID), nil
		}
	}
	return 0, fmt.Errorf("project %q not found", name)
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON")
	rootCmd.PersistentFlags().Bool("yaml", false, "output as YAML")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "auto-confirm prompts")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringP("server", "s", "", "override server host:port for this invocation")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "HTTP request timeout (e.g. 30s, 2m)")
	rootCmd.PersistentFlags().Bool("insecure", false, "skip TLS certificate verification (not recommended)")
	rootCmd.PersistentFlags().String("ca-cert", "", "path to a CA certificate file for TLS verification")
	rootCmd.PersistentFlags().BoolP("interactive", "I", false, "force interactive mode even when all inputs are provided")
	rootCmd.PersistentFlags().BoolP("no-interactive", "N", false, "disable interactive mode even when inputs are missing")
	rootCmd.PersistentFlags().String("context", "", "use a specific context for this command")
	rootCmd.PersistentFlags().StringP("project", "p", "", "project ID or name for project-scoped commands")
}
