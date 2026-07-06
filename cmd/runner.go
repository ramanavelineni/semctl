package cmd

import (
	"github.com/spf13/cobra"
)

var runnerCmd = &cobra.Command{
	Use:   "runner",
	Short: "Manage task runners (Semaphore UI 2.18+)",
	Long: `Manage Semaphore task runners.

Runners are GLOBAL by default. Pass --project explicitly to manage a
project-scoped runner instead; the defaults.project_id config value is
deliberately ignored here so global runners stay the default.`,
}

// runnerScope returns the project ID and whether the command targets
// project-scoped runners. Unlike getProjectID, only an explicit --project
// flag selects project scope — the config default does not apply.
func runnerScope(cmd *cobra.Command) (int64, bool, error) {
	if !cmd.Flags().Changed("project") {
		return 0, false, nil
	}
	pid, err := getProjectID(cmd)
	if err != nil {
		return 0, false, err
	}
	return int64(pid), true, nil
}

func init() {
	rootCmd.AddCommand(runnerCmd)
}
