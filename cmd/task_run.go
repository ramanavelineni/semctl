package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

var taskRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a task",
	Example: `  semctl task run --template-id 1
  semctl task run --template-id 1 --message "Deploy v1.2" --git-branch main`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		templateID, _ := cmd.Flags().GetInt64("template-id")
		message, _ := cmd.Flags().GetString("message")
		gitBranch, _ := cmd.Flags().GetString("git-branch")
		arguments, _ := cmd.Flags().GetString("arguments")
		environment, _ := cmd.Flags().GetString("environment")
		limit, _ := cmd.Flags().GetString("limit")
		playbook, _ := cmd.Flags().GetString("playbook")
		debug, _ := cmd.Flags().GetBool("debug")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		diff, _ := cmd.Flags().GetBool("diff")

		if templateID == 0 {
			return fmt.Errorf("--template-id is required")
		}

		body := task.PostProjectProjectIDTasksBody{
			TemplateID:  templateID,
			Message:     message,
			GitBranch:   gitBranch,
			Arguments:   arguments,
			Environment: environment,
			Limit:       limit,
			Playbook:    playbook,
			Debug:       debug,
			DryRun:      dryRun,
			Diff:        diff,
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := task.NewPostProjectProjectIDTasksParams()
		params.ProjectID = int64(pid)
		params.Task = body

		resp, err := apiClient.Task.PostProjectProjectIDTasks(params, nil)
		if err != nil {
			return fmt.Errorf("failed to run task: %w", err)
		}

		t := resp.GetPayload()
		style.Success(fmt.Sprintf("Started task %d (template: %d)", t.ID, t.TemplateID))
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskRunCmd)

	taskRunCmd.Flags().Int64("template-id", 0, "template ID (required)")
	taskRunCmd.Flags().String("message", "", "task message")
	taskRunCmd.Flags().String("git-branch", "", "git branch override")
	taskRunCmd.Flags().String("arguments", "", "extra arguments (JSON)")
	taskRunCmd.Flags().String("environment", "", "environment override (JSON)")
	taskRunCmd.Flags().String("limit", "", "limit hosts")
	taskRunCmd.Flags().String("playbook", "", "playbook override")
	taskRunCmd.Flags().Bool("debug", false, "enable debug mode")
	taskRunCmd.Flags().Bool("dry-run", false, "dry run mode")
	taskRunCmd.Flags().Bool("diff", false, "show diff")
}
