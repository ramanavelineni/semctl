package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var scheduleShowCmd = &cobra.Command{
	Use:     "show <id|name>",
	Short:   "Show schedule details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl schedule show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "schedule", scheduleNameIDs)
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

		return runShow("schedule",
			func() (*models.Schedule, error) {
				params := schedule.NewGetProjectProjectIDSchedulesScheduleIDParams()
				params.ProjectID = int64(pid)
				params.ScheduleID = id
				resp, err := apiClient.Schedule.GetProjectProjectIDSchedulesScheduleID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(s *models.Schedule) [][]string {
				runAt := ""
				if !s.RunAt.IsZero() {
					runAt = s.RunAt.String()
				}
				return [][]string{
					{"ID", strconv.FormatInt(s.ID, 10)},
					{"Name", s.Name},
					{"Template ID", strconv.FormatInt(s.TemplateID, 10)},
					{"Cron Format", s.CronFormat},
					{"Active", strconv.FormatBool(s.Active)},
					{"Type", s.Type},
					{"Run At", runAt},
				}
			})
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleShowCmd)
}
