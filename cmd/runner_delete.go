package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/spf13/cobra"
)

var runnerDeleteCmd = &cobra.Command{
	Use:     "delete <id|name>",
	Aliases: []string{"rm"},
	Short:   "Delete a runner",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "runner", runnerNameIDs)
		if err != nil {
			return err
		}
		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "runner", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			if projectScoped {
				params := runner.NewDeleteProjectProjectIDRunnersRunnerIDParams()
				params.ProjectID = pid
				params.RunnerID = id
				_, err = apiClient.Runner.DeleteProjectProjectIDRunnersRunnerID(params, nil)
				return err
			}
			params := runner.NewDeleteRunnersRunnerIDParams()
			params.RunnerID = id
			_, err = apiClient.Runner.DeleteRunnersRunnerID(params, nil)
			return err
		})
	},
}

func init() {
	runnerCmd.AddCommand(runnerDeleteCmd)
}
