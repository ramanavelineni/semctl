package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
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

		params := template.NewGetProjectProjectIDTemplatesParams()
		params.ProjectID = int64(pid)

		resp, err := apiClient.Template.GetProjectProjectIDTemplates(params, nil)
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name", "Type", "App", "Playbook", "Repository ID"}
		var rows [][]string
		for _, t := range items {
			rows = append(rows, []string{
				strconv.FormatInt(t.ID, 10),
				t.Name,
				t.Type,
				t.App,
				t.Playbook,
				strconv.FormatInt(t.RepositoryID, 10),
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no templates found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	templateCmd.AddCommand(templateListCmd)
}
