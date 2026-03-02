package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
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
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		autoConfirm, _ := cmd.Flags().GetBool("yes")
		if !autoConfirm {
			fmt.Fprintf(os.Stderr, "Delete task %d? [y/N] ", id)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				style.Info("Cancelled.")
				return nil
			}
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := task.NewDeleteProjectProjectIDTasksTaskIDParams()
		params.ProjectID = int64(pid)
		params.TaskID = id

		_, err = apiClient.Task.DeleteProjectProjectIDTasksTaskID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}

		style.Success(fmt.Sprintf("Deleted task %d", id))
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskDeleteCmd)
}
