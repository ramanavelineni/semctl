package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/spf13/cobra"
)

var repoDeleteCmd = &cobra.Command{
	Use:     "delete <id|name>",
	Aliases: []string{"rm"},
	Short:   "Delete a repository",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl repo delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "repository", repoNameIDs)
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "repository", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := repository.NewDeleteProjectProjectIDRepositoriesRepositoryIDParams()
			params.ProjectID = int64(pid)
			params.RepositoryID = id
			_, err = apiClient.Repository.DeleteProjectProjectIDRepositoriesRepositoryID(params, nil)
			return err
		})
	},
}

func init() {
	repoCmd.AddCommand(repoDeleteCmd)
}
