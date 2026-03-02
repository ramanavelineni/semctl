package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/spf13/cobra"
)

var projectShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show project details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl project show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := project.NewGetProjectProjectIDParams()
		params.ProjectID = id

		resp, err := apiClient.Project.GetProjectProjectID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		p := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(p, nil, nil)
			return nil
		}

		maxParallel := "0"
		if p.MaxParallelTasks != nil {
			maxParallel = strconv.FormatInt(*p.MaxParallelTasks, 10)
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(p.ID, 10)},
			{"Name", p.Name},
			{"Type", p.Type},
			{"Alert", strconv.FormatBool(p.Alert)},
			{"Alert Chat", p.AlertChat},
			{"Max Parallel Tasks", maxParallel},
			{"Created", p.Created},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectShowCmd)
}
