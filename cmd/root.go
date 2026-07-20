package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
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
	Example: `  semctl login --server 10.0.0.1:3000 --username admin
  semctl project list
  semctl -p "My Project" template list
  semctl -p 1 task run --template-id 5 --wait`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Session-level flags apply to every command, including those that
		// skip config loading (e.g. login).
		client.SetRootContext(cmd.Context())
		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			output.DisableColor()
		}
		if q, _ := cmd.Flags().GetBool("quiet"); q {
			style.SetQuiet(true)
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
		// (trust manages the trust store, not the config itself)
		switch cmd.Name() {
		case "completion", "version", "__complete", "login", "logout", "trust":
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
		warnIfUntrustedSkipped()

		// Apply --context override if set
		if ctxFlag, _ := cmd.Flags().GetString("context"); ctxFlag != "" {
			if err := config.ApplyContext(ctxFlag); err != nil {
				return fmt.Errorf("context error: %w", err)
			}
		}

		return resolveOutputFormat(cmd)
	},
}

// resolveOutputFormat applies --output/--json/--yaml, erroring when they
// disagree (previously --json --yaml silently picked JSON). export shadows
// the global --output with its deprecated file-path flag, so the string flag
// only counts as a format when it is the root's own.
func resolveOutputFormat(cmd *cobra.Command) error {
	formats := map[output.Format]bool{}
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		formats[output.FormatJSON] = true
	}
	if yamlFlag, _ := cmd.Flags().GetBool("yaml"); yamlFlag {
		formats[output.FormatYAML] = true
	}
	if f := cmd.Flags().Lookup("output"); f != nil && f.Changed && f == cmd.Root().PersistentFlags().Lookup("output") {
		switch v := output.Format(strings.ToLower(f.Value.String())); v {
		case output.FormatTable, output.FormatJSON, output.FormatYAML:
			formats[v] = true
		default:
			return fmt.Errorf("invalid output format %q (valid: table, json, yaml)", f.Value.String())
		}
	}
	switch len(formats) {
	case 0:
		output.SetFormat(output.FormatFromConfig())
	case 1:
		for f := range formats {
			output.SetFormat(f)
		}
	default:
		return fmt.Errorf("conflicting output formats: pass only one of --output, --json, --yaml")
	}
	return nil
}

// Execute runs the root command.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		// After the first signal cancels ctx, restore the default handler so
		// a second Ctrl-C kills the process even if a command ignores ctx.
		<-ctx.Done()
		stop()
	}()
	config.TrustPrompt = promptTrustCWDConfig
	enforceSubcommands(rootCmd)
	registerDynamicCompletions()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
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
	items, err := projectNameIDs(nil)
	if err != nil {
		return 0, err
	}
	id, err := matchNameID(items, name, "project")
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.PersistentFlags().String("output", "", "output format: table, json, or yaml")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON (same as --output json)")
	rootCmd.PersistentFlags().Bool("yaml", false, "output as YAML (same as --output yaml)")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "auto-confirm prompts")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress success/info messages (warnings and errors still print)")
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
