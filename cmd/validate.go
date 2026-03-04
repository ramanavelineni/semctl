package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/apply"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate declarative configuration files",
	Long: `Validate YAML or JSON configuration files without connecting to Semaphore.

Checks file syntax, required fields, and cross-reference consistency.
No API calls are made.`,
	Example: `  semctl validate -f project.yaml
  semctl validate -f keys.yaml -f templates.yaml
  semctl validate -f ./semaphore/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePaths, _ := cmd.Flags().GetStringArray("file")

		if len(filePaths) == 0 {
			return fmt.Errorf("--file (-f) is required")
		}

		files, err := apply.CollectFiles(filePaths)
		if err != nil {
			return err
		}

		errors := 0
		for _, f := range files {
			cfg, err := apply.ParseFile(f)
			if err != nil {
				style.Error(fmt.Sprintf("%s: %v", f, err))
				errors++
				continue
			}
			if err := cfg.Validate(); err != nil {
				style.Error(fmt.Sprintf("%s: %v", f, err))
				errors++
				continue
			}

			style.Success(fmt.Sprintf("%s: valid (%s)", f, summarizeConfig(cfg)))
		}

		if errors > 0 {
			return fmt.Errorf("validation failed for %d file(s)", errors)
		}
		return nil
	},
}

func summarizeConfig(cfg *apply.ApplyConfig) string {
	parts := []string{fmt.Sprintf("project %q", cfg.Project)}
	if n := len(cfg.Keys); n > 0 {
		parts = append(parts, fmt.Sprintf("%d keys", n))
	}
	if n := len(cfg.VariableGroups); n > 0 {
		parts = append(parts, fmt.Sprintf("%d variable groups", n))
	}
	if n := len(cfg.Repositories); n > 0 {
		parts = append(parts, fmt.Sprintf("%d repositories", n))
	}
	if n := len(cfg.Inventories); n > 0 {
		parts = append(parts, fmt.Sprintf("%d inventories", n))
	}
	if n := len(cfg.Templates); n > 0 {
		parts = append(parts, fmt.Sprintf("%d templates", n))
	}
	if n := len(cfg.Schedules); n > 0 {
		parts = append(parts, fmt.Sprintf("%d schedules", n))
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringArrayP("file", "f", nil, "config file or directory (can be specified multiple times)")
}
