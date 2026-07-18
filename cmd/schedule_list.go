package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var scheduleListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List schedules",
	Example: "  semctl schedule list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("schedules",
			[]string{"ID", "Name", "Template", "Cron", "Active", "Type"},
			func() ([]*models.Schedule, error) {
				params := schedule.NewGetProjectProjectIDSchedulesParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.Schedule.GetProjectProjectIDSchedules(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(s *models.Schedule) []string {
				return []string{
					strconv.FormatInt(s.ID, 10),
					s.Name,
					s.TplName,
					s.CronFormat,
					strconv.FormatBool(s.Active),
					s.Type,
				}
			})
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleListCmd)
}
