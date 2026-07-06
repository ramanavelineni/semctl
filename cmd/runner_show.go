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

var runnerShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show runner details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid runner ID: %w", err)
		}

		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		var r *models.Runner
		if projectScoped {
			params := runner.NewGetProjectProjectIDRunnersRunnerIDParams()
			params.ProjectID = pid
			params.RunnerID = id
			resp, err := apiClient.Runner.GetProjectProjectIDRunnersRunnerID(params, nil)
			if err != nil {
				return fmt.Errorf("failed to get runner: %w", err)
			}
			r = resp.GetPayload()
		} else {
			params := runner.NewGetRunnersRunnerIDParams()
			params.RunnerID = id
			resp, err := apiClient.Runner.GetRunnersRunnerID(params, nil)
			if err != nil {
				return fmt.Errorf("failed to get runner: %w", err)
			}
			r = resp.GetPayload()
		}

		if output.GetFormat() != output.FormatTable {
			output.Print(r, nil, nil)
			return nil
		}

		projectID := ""
		if r.ProjectID != nil {
			projectID = strconv.FormatInt(*r.ProjectID, 10)
		}
		touched := ""
		if r.Touched != nil {
			touched = r.Touched.String()
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(r.ID, 10)},
			{"Name", r.Name},
			{"Active", strconv.FormatBool(r.Active)},
			{"Registered", strconv.FormatBool(r.Registered)},
			{"Default", strconv.FormatBool(r.IsDefault)},
			{"Max Parallel Tasks", strconv.FormatInt(r.MaxParallelTasks, 10)},
			{"Tags", strings.Join(r.Tags, ",")},
			{"Webhook", r.Webhook},
			{"Project ID", projectID},
			{"Last Seen", touched},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerShowCmd)
}
