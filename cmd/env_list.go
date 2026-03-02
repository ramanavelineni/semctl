package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/spf13/cobra"
)

var envListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List environments",
	Example: "  semctl env list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := variable_group.NewGetProjectProjectIDEnvironmentParams()
		params.ProjectID = int64(pid)

		resp, err := apiClient.VariableGroup.GetProjectProjectIDEnvironment(params, nil)
		if err != nil {
			return fmt.Errorf("failed to list environments: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name"}
		var rows [][]string
		for _, e := range items {
			rows = append(rows, []string{
				strconv.FormatInt(e.ID, 10),
				e.Name,
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no environments found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	envCmd.AddCommand(envListCmd)
}
