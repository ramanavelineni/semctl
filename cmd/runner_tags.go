package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerTagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List runner tags",
	Example: `  semctl runner tags
  semctl runner tags -p 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		var items []*models.RunnerTag
		if projectScoped {
			params := runner.NewGetProjectProjectIDRunnerTagsParams()
			params.ProjectID = pid
			resp, err := apiClient.Runner.GetProjectProjectIDRunnerTags(params, nil)
			if err != nil {
				return fmt.Errorf("failed to list runner tags: %w", err)
			}
			items = resp.GetPayload()
		} else {
			resp, err := apiClient.Runner.GetRunnerTags(runner.NewGetRunnerTagsParams(), nil)
			if err != nil {
				return fmt.Errorf("failed to list runner tags: %w", err)
			}
			items = resp.GetPayload()
		}

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"Tag", "Runners"}
		var rows [][]string
		for _, t := range items {
			rows = append(rows, []string{
				t.Tag,
				strconv.FormatInt(t.NumberOfRunners, 10),
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no runner tags found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerTagsCmd)
}
