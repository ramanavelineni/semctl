package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

var taskOutputCmd = &cobra.Command{
	Use:     "output <id>",
	Aliases: []string{"log", "logs"},
	Short:   "Get task output",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl task output 23",
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

		params := task.NewGetProjectProjectIDTasksTaskIDOutputParams()
		params.ProjectID = int64(pid)
		params.TaskID = id

		resp, err := apiClient.Task.GetProjectProjectIDTasksTaskIDOutput(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get task output: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		for _, line := range items {
			fmt.Println(line.Output)
		}

		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskOutputCmd)
}
