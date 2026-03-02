package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

var taskListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tasks",
	Example: "  semctl task list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := task.NewGetProjectProjectIDTasksParams()
		params.ProjectID = int64(pid)

		resp, err := apiClient.Task.GetProjectProjectIDTasks(params, nil)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Template ID", "Status", "Message", "Git Branch"}
		var rows [][]string
		for _, t := range items {
			rows = append(rows, []string{
				strconv.FormatInt(t.ID, 10),
				strconv.FormatInt(t.TemplateID, 10),
				t.Status,
				t.Message,
				t.GitBranch,
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no tasks found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskListCmd)
}
