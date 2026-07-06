package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List runners",
	Example: `  semctl runner list
  semctl runner list -p 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		var items []*models.Runner
		if projectScoped {
			params := runner.NewGetProjectProjectIDRunnersParams()
			params.ProjectID = pid
			resp, err := apiClient.Runner.GetProjectProjectIDRunners(params, nil)
			if err != nil {
				return fmt.Errorf("failed to list project runners: %w", err)
			}
			items = resp.GetPayload()
		} else {
			resp, err := apiClient.Runner.GetRunners(runner.NewGetRunnersParams(), nil)
			if err != nil {
				return fmt.Errorf("failed to list runners: %w", err)
			}
			items = resp.GetPayload()
		}

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name", "Active", "Registered", "Default", "Max Parallel", "Tags"}
		var rows [][]string
		for _, r := range items {
			rows = append(rows, []string{
				strconv.FormatInt(r.ID, 10),
				r.Name,
				strconv.FormatBool(r.Active),
				strconv.FormatBool(r.Registered),
				strconv.FormatBool(r.IsDefault),
				strconv.FormatInt(r.MaxParallelTasks, 10),
				strings.Join(r.Tags, ","),
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no runners found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerListCmd)
}
