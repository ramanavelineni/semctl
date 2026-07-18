package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
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

		return runList("tasks",
			[]string{"ID", "Template ID", "Status", "Message", "Git Branch"},
			func() ([]*models.Task, error) {
				params := task.NewGetProjectProjectIDTasksParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.Task.GetProjectProjectIDTasks(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(t *models.Task) []string {
				return []string{
					strconv.FormatInt(t.ID, 10),
					strconv.FormatInt(t.TemplateID, 10),
					t.Status,
					t.Message,
					t.GitBranch,
				}
			})
	},
}

func init() {
	taskCmd.AddCommand(taskListCmd)
}
