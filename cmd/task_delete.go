package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

var taskDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a task",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl task delete 23",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "task")
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "task", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := task.NewDeleteProjectProjectIDTasksTaskIDParams()
			params.ProjectID = int64(pid)
			params.TaskID = id
			_, err = apiClient.Task.DeleteProjectProjectIDTasksTaskID(params, nil)
			return err
		})
	},
}

func init() {
	taskCmd.AddCommand(taskDeleteCmd)
}
