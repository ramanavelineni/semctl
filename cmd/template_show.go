package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var templateShowCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show template details",
	Args:  cobra.ExactArgs(1),
	Example: `  semctl template show 1
  semctl template show "Deploy App"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "template", templateNameIDs)
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

		return runShow("template",
			func() (*models.Template, error) {
				params := template.NewGetProjectProjectIDTemplatesTemplateIDParams()
				params.ProjectID = int64(pid)
				params.TemplateID = id
				resp, err := apiClient.Template.GetProjectProjectIDTemplatesTemplateID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(t *models.Template) [][]string {
				return [][]string{
					{"ID", strconv.FormatInt(t.ID, 10)},
					{"Name", t.Name},
					{"Description", t.Description},
					{"Type", t.Type},
					{"App", t.App},
					{"Playbook", t.Playbook},
					{"Git Branch", t.GitBranch},
					{"Repository ID", strconv.FormatInt(t.RepositoryID, 10)},
					{"Environment ID", strconv.FormatInt(t.EnvironmentID, 10)},
					{"Inventory ID", strconv.FormatInt(t.InventoryID, 10)},
					{"Build Template ID", strconv.FormatInt(t.BuildTemplateID, 10)},
					{"View ID", strconv.FormatInt(t.ViewID, 10)},
					{"Autorun", strconv.FormatBool(t.Autorun)},
					{"Suppress Success Alerts", strconv.FormatBool(t.SuppressSuccessAlerts)},
				}
			})
	},
}

func init() {
	templateCmd.AddCommand(templateShowCmd)
}
