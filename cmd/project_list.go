package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
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

		resp, err := apiClient.Project.GetProjects(project.NewGetProjectsParams(), nil)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name", "Type", "Alert", "Created"}
		var rows [][]string
		for _, p := range items {
			rows = append(rows, []string{
				strconv.FormatInt(p.ID, 10),
				p.Name,
				p.Type,
				strconv.FormatBool(p.Alert),
				p.Created,
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no projects found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}
