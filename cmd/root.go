package cmd

import (
	"fmt"
	"os"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "semctl",
	Short: "Semaphore UI CLI",
	Long:         "A command-line interface for managing Semaphore UI via its REST API.",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			output.DisableColor()
		}

		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// getProjectID resolves the project ID from --project flag, config default, or returns an error.
func getProjectID(cmd *cobra.Command) (int32, error) {
	if p, _ := cmd.Flags().GetInt32("project"); p > 0 {
		return p, nil
	}
	if p := config.GetDefaultProjectID(); p > 0 {
		return int32(p), nil
	}
	return 0, fmt.Errorf("project ID is required: use --project flag or set defaults.project_id in config")
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to config file")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON")
	rootCmd.PersistentFlags().Bool("yaml", false, "output as YAML")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "auto-confirm prompts")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringP("server", "s", "", "override server host:port")
	rootCmd.PersistentFlags().BoolP("interactive", "I", false, "force interactive mode even when all inputs are provided")
	rootCmd.PersistentFlags().BoolP("no-interactive", "N", false, "disable interactive mode even when inputs are missing")
	rootCmd.PersistentFlags().String("context", "", "use a specific context for this command")
	rootCmd.PersistentFlags().Int32P("project", "p", 0, "project ID for project-scoped commands")
}
