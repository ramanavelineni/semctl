package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var templateListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List templates",
	Example: "  semctl template list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("templates",
			[]string{"ID", "Name", "Type", "App", "Playbook", "Repository ID"},
			func() ([]*models.Template, error) {
				params := template.NewGetProjectProjectIDTemplatesParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.Template.GetProjectProjectIDTemplates(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(t *models.Template) []string {
				return []string{
					strconv.FormatInt(t.ID, 10),
					t.Name,
					t.Type,
					t.App,
					t.Playbook,
					strconv.FormatInt(t.RepositoryID, 10),
				}
			})
	},
}

func init() {
	templateCmd.AddCommand(templateListCmd)
}
