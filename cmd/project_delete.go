package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/spf13/cobra"
)

var projectDeleteCmd = &cobra.Command{
	Use:     "delete <id|name>",
	Aliases: []string{"rm"},
	Short:   "Delete a project",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl project delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "project", projectNameIDs)
		if err != nil {
			return err
		}

		return runDelete(cmd, "project", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := project.NewDeleteProjectProjectIDParams()
			params.ProjectID = id
			_, err = apiClient.Project.DeleteProjectProjectID(params, nil)
			return err
		})
	},
}

func init() {
	projectCmd.AddCommand(projectDeleteCmd)
}
