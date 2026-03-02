package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

var taskStopCmd = &cobra.Command{
	Use:     "stop <id>",
	Short:   "Stop a running task",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl task stop 23",
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

		params := task.NewPostProjectProjectIDTasksTaskIDStopParams()
		params.ProjectID = int64(pid)
		params.TaskID = id

		_, err = apiClient.Task.PostProjectProjectIDTasksTaskIDStop(params, nil)
		if err != nil {
			return fmt.Errorf("failed to stop task: %w", err)
		}

		style.Success(fmt.Sprintf("Stopped task %d", id))
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskStopCmd)
}
