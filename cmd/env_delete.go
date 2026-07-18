package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/spf13/cobra"
)

var envDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete an environment",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl env delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "environment")
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "environment", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := variable_group.NewDeleteProjectProjectIDEnvironmentEnvironmentIDParams()
			params.ProjectID = int64(pid)
			params.EnvironmentID = id
			_, err = apiClient.VariableGroup.DeleteProjectProjectIDEnvironmentEnvironmentID(params, nil)
			return err
		})
	},
}

func init() {
	envCmd.AddCommand(envDeleteCmd)
}
