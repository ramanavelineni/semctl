package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List repositories",
	Example: "  semctl repo list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("repositories",
			[]string{"ID", "Name", "Git URL", "Git Branch", "SSH Key ID"},
			func() ([]*models.Repository, error) {
				params := repository.NewGetProjectProjectIDRepositoriesParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.Repository.GetProjectProjectIDRepositories(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(r *models.Repository) []string {
				return []string{
					strconv.FormatInt(r.ID, 10),
					r.Name,
					r.GitURL,
					r.GitBranch,
					strconv.FormatInt(r.SSHKeyID, 10),
				}
			})
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
}
