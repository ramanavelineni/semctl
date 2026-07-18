package cmd

import (
	"fmt"
	"os"

	"github.com/ramanavelineni/semctl/internal/apply"
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
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

${VAR} references in config files are expanded from the environment.
Referencing an unset variable is an error; write $${VAR} for a literal ${VAR}.
Bare $WORD text (no braces) is left untouched.`,
	Example: `  semctl apply -f project.yaml
  semctl apply -f keys.yaml -f templates.yaml
  semctl apply -f ./semaphore/
  semctl apply -f project.yaml --dry-run
  semctl apply -f project.json --yes

  # GitOps drift gate: exit 0 = in sync, 2 = changes pending, 1 = error
  semctl apply -f ./semaphore/ --dry-run --detailed-exitcode
  # Machine-readable plan on stdout
  semctl apply -f project.yaml --dry-run --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePaths, _ := cmd.Flags().GetStringArray("file")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		skipSchedules, _ := cmd.Flags().GetBool("skip-schedules")
		detailedExit, _ := cmd.Flags().GetBool("detailed-exitcode")

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

		// Apply mutates many resources at once — worth one extra request to
		// flag a client/server version mismatch before anything changes.
		client.WarnIfVersionMismatch(apiClient)

		totalCreates, totalUpdates, totalDeletes := 0, 0, 0
		totalErrors := 0
		anyChanges := false
		var planDocs []apply.PlanJSON

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

			if skipSchedules {
				cfg.Schedules = nil
			}

			recon := apply.NewReconciler(apiClient, cfg)
			plan, err := recon.BuildPlan()
			if err != nil {
				style.Error(fmt.Sprintf("%s: %v", f, err))
				totalErrors++
				continue
			}

			planDocs = append(planDocs, plan.JSON(f))
			fmt.Fprint(os.Stderr, plan.FormatPlan())

			if !plan.HasChanges() {
				style.Success("No changes needed.")
				continue
			}
			anyChanges = true

			if dryRun {
				style.Info("Dry run — no changes applied.")
				continue
			}

			if err := confirmAction(cmd, "Apply these changes?"); err != nil {
				return err
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

		// Machine-readable plans on stdout (human plans went to stderr above).
		if output.GetFormat() != output.FormatTable {
			if err := output.Print(planDocs, nil, nil); err != nil {
				return err
			}
		}

		if totalErrors > 0 {
			return fmt.Errorf("apply completed with %d error(s) (%d created, %d updated, %d deleted)",
				totalErrors, totalCreates, totalUpdates, totalDeletes)
		}

		if !dryRun && (totalCreates+totalUpdates+totalDeletes) > 0 {
			style.Success(fmt.Sprintf("Apply complete: %d created, %d updated, %d deleted.",
				totalCreates, totalUpdates, totalDeletes))
		}

		// Terraform-plan convention for drift gates: 2 = changes present.
		if detailedExit && anyChanges {
			return withExitCode(fmt.Errorf("changes detected (--detailed-exitcode)"), exitDrift)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringArrayP("file", "f", nil, "config file or directory (can be specified multiple times)")
	applyCmd.Flags().Bool("dry-run", false, "preview changes without applying")
	applyCmd.Flags().Bool("skip-schedules", false, "leave schedule resources unmanaged by this apply")
	applyCmd.Flags().Bool("detailed-exitcode", false, "exit 2 when the plan has changes, 0 when in sync (1 = error); combine with --dry-run for drift gates")
}
