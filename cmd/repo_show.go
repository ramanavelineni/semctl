package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var repoShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show repository details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl repo show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "repository")
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runShow("repository",
			func() (*models.Repository, error) {
				params := repository.NewGetProjectProjectIDRepositoriesRepositoryIDParams()
				params.ProjectID = int64(pid)
				params.RepositoryID = id
				resp, err := apiClient.Repository.GetProjectProjectIDRepositoriesRepositoryID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(r *models.Repository) [][]string {
				return [][]string{
					{"ID", strconv.FormatInt(r.ID, 10)},
					{"Name", r.Name},
					{"Git URL", r.GitURL},
					{"Git Branch", r.GitBranch},
					{"SSH Key ID", strconv.FormatInt(r.SSHKeyID, 10)},
				}
			})
	},
}

func init() {
	repoCmd.AddCommand(repoShowCmd)
}
