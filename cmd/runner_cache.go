package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/spf13/cobra"
)

var runnerClearCacheCmd = &cobra.Command{
	Use:     "clear-cache <id>",
	Short:   "Clear a runner's cache",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner clear-cache 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid runner ID: %w", err)
		}

		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		if projectScoped {
			params := runner.NewDeleteProjectProjectIDRunnersRunnerIDCacheParams()
			params.ProjectID = pid
			params.RunnerID = id
			if _, err := apiClient.Runner.DeleteProjectProjectIDRunnersRunnerIDCache(params, nil); err != nil {
				return fmt.Errorf("failed to clear runner cache: %w", err)
			}
		} else {
			params := runner.NewDeleteRunnersRunnerIDCacheParams()
			params.RunnerID = id
			if _, err := apiClient.Runner.DeleteRunnersRunnerIDCache(params, nil); err != nil {
				return fmt.Errorf("failed to clear runner cache: %w", err)
			}
		}

		style.Success(fmt.Sprintf("Cleared cache for runner %d", id))
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerClearCacheCmd)
}
