package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/spf13/cobra"
)

var templateShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show template details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl template show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid template ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := template.NewGetProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = int64(pid)
		params.TemplateID = id

		resp, err := apiClient.Template.GetProjectProjectIDTemplatesTemplateID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		t := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(t, nil, nil)
			return nil
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
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

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	templateCmd.AddCommand(templateShowCmd)
}
