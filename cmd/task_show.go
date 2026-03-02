package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

var taskShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show task details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl task show 23",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := task.NewGetProjectProjectIDTasksTaskIDParams()
		params.ProjectID = int64(pid)
		params.TaskID = id

		resp, err := apiClient.Task.GetProjectProjectIDTasksTaskID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		t := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(t, nil, nil)
			return nil
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(t.ID, 10)},
			{"Template ID", strconv.FormatInt(t.TemplateID, 10)},
			{"Status", t.Status},
			{"Message", t.Message},
			{"Git Branch", t.GitBranch},
			{"Playbook", t.Playbook},
			{"Environment", t.Environment},
			{"Arguments", t.Arguments},
			{"Limit", t.Limit},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskShowCmd)
}
