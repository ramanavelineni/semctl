package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all projects",
	Example: "  semctl project list",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("projects",
			[]string{"ID", "Name", "Type", "Alert", "Created"},
			func() ([]*models.Project, error) {
				resp, err := apiClient.Project.GetProjects(project.NewGetProjectsParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(p *models.Project) []string {
				return []string{
					strconv.FormatInt(p.ID, 10),
					p.Name,
					p.Type,
					strconv.FormatBool(p.Alert),
					p.Created,
				}
			})
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}
