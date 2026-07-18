package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var taskShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show task details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl task show 23",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "task")
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runShow("task",
			func() (*models.Task, error) {
				params := task.NewGetProjectProjectIDTasksTaskIDParams()
				params.ProjectID = int64(pid)
				params.TaskID = id
				resp, err := apiClient.Task.GetProjectProjectIDTasksTaskID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(t *models.Task) [][]string {
				return [][]string{
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
			})
	},
}

func init() {
	taskCmd.AddCommand(taskShowCmd)
}
