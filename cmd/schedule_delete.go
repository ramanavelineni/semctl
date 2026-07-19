package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/spf13/cobra"
)

var scheduleDeleteCmd = &cobra.Command{
	Use:     "delete <id|name>",
	Aliases: []string{"rm"},
	Short:   "Delete a schedule",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl schedule delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "schedule", scheduleNameIDs)
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "schedule", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := schedule.NewDeleteProjectProjectIDSchedulesScheduleIDParams()
			params.ProjectID = int64(pid)
			params.ScheduleID = id
			_, err = apiClient.Schedule.DeleteProjectProjectIDSchedulesScheduleID(params, nil)
			return err
		})
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleDeleteCmd)
}
