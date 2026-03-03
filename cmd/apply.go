package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ramanavelineni/semctl/internal/apply"
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply declarative configuration to Semaphore",
	Long: `Apply YAML or JSON configuration files to reconcile Semaphore project state.

Resources are matched by name and created, updated, or deleted as needed.
Use --dry-run to preview changes without applying them.

Multiple files can be specified with repeated -f flags. Each file is applied
independently in order. Directories are scanned for .yaml, .yml, and .json files.

Environment variables in config files are expanded (e.g. ${SSH_PRIVATE_KEY}).`,
	Example: `  semctl apply -f project.yaml
  semctl apply -f keys.yaml -f templates.yaml
  semctl apply -f ./semaphore/
  semctl apply -f project.yaml --dry-run
  semctl apply -f project.json --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePaths, _ := cmd.Flags().GetStringArray("file")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		autoConfirm, _ := cmd.Flags().GetBool("yes")

		if len(filePaths) == 0 {
			return fmt.Errorf("--file (-f) is required")
		}

		// Resolve directories and collect files
		files, err := apply.CollectFiles(filePaths)
		if err != nil {
			return err
		}

		// Authenticate once for all files
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		totalCreates, totalUpdates, totalDeletes := 0, 0, 0
		totalErrors := 0

		for _, f := range files {
			if len(files) > 1 {
				style.Info(fmt.Sprintf("Applying %s", f))
			}

			cfg, err := apply.ParseFile(f)
			if err != nil {
				style.Error(fmt.Sprintf("%s: %v", f, err))
				totalErrors++
				continue
			}
			if err := cfg.Validate(); err != nil {
				style.Error(fmt.Sprintf("%s: validation error: %v", f, err))
				totalErrors++
				continue
			}

			recon := apply.NewReconciler(apiClient, cfg)
			plan, err := recon.BuildPlan()
			if err != nil {
				style.Error(fmt.Sprintf("%s: %v", f, err))
				totalErrors++
				continue
			}

			fmt.Fprint(os.Stderr, plan.FormatPlan())

			if !plan.HasChanges() {
				style.Success("No changes needed.")
				continue
			}

			if dryRun {
				style.Info("Dry run — no changes applied.")
				continue
			}

			if !autoConfirm {
				fmt.Fprint(os.Stderr, "Apply these changes? [y/N] ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					style.Info("Cancelled.")
					continue
				}
			}

			executor := apply.NewExecutor(apiClient, cfg, recon)
			errorCount := executor.Execute(plan)
			totalErrors += errorCount

			creates, updates, deletes, _ := plan.Summary()
			totalCreates += creates
			totalUpdates += updates
			totalDeletes += deletes
		}

		if len(files) > 1 {
			fmt.Fprintln(os.Stderr)
		}

		if totalErrors > 0 {
			return fmt.Errorf("apply completed with %d error(s) (%d created, %d updated, %d deleted)",
				totalErrors, totalCreates, totalUpdates, totalDeletes)
		}

		if !dryRun && (totalCreates+totalUpdates+totalDeletes) > 0 {
			style.Success(fmt.Sprintf("Apply complete: %d created, %d updated, %d deleted.",
				totalCreates, totalUpdates, totalDeletes))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringArrayP("file", "f", nil, "config file or directory (can be specified multiple times)")
	applyCmd.Flags().Bool("dry-run", false, "preview changes without applying")
}
