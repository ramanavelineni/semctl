package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/spf13/cobra"
)

var runnerDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a runner",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid runner ID: %w", err)
		}

		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		if err := confirmAction(cmd, fmt.Sprintf("Delete runner %d?", id)); err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		if projectScoped {
			params := runner.NewDeleteProjectProjectIDRunnersRunnerIDParams()
			params.ProjectID = pid
			params.RunnerID = id
			if _, err := apiClient.Runner.DeleteProjectProjectIDRunnersRunnerID(params, nil); err != nil {
				return fmt.Errorf("failed to delete runner: %w", err)
			}
		} else {
			params := runner.NewDeleteRunnersRunnerIDParams()
			params.RunnerID = id
			if _, err := apiClient.Runner.DeleteRunnersRunnerID(params, nil); err != nil {
				return fmt.Errorf("failed to delete runner: %w", err)
			}
		}

		style.Success(fmt.Sprintf("Deleted runner %d", id))
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerDeleteCmd)
}
