package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var scheduleUpdateCmd = &cobra.Command{
	Use:   "update <id|name> [field=value...]",
	Short: "Update a schedule",
	Long:  `Update a schedule. Fields: name, template_id, cron_format, active, type, run_at.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl schedule update 1 cron_format="0 4 * * *"
  semctl schedule update 1 active=false name="Paused nightly"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "schedule", scheduleNameIDs)
		if err != nil {
			return err
		}
		if len(args) < 2 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		getParams := schedule.NewGetProjectProjectIDSchedulesScheduleIDParams()
		getParams.ProjectID = int64(pid)
		getParams.ScheduleID = id
		getResp, err := apiClient.Schedule.GetProjectProjectIDSchedulesScheduleID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}
		cur := getResp.GetPayload()

		req := &models.ScheduleRequest{
			ID:         cur.ID,
			ProjectID:  int64(pid),
			Name:       cur.Name,
			TemplateID: cur.TemplateID,
			CronFormat: cur.CronFormat,
			Active:     cur.Active,
			Type:       cur.Type,
			RunAt:      cur.RunAt,
			TaskParams: cur.TaskParams,
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			key = strings.ReplaceAll(key, "-", "_") // accept kebab-case like the create flags
			switch key {
			case "name":
				req.Name = value
			case "template_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for template_id: %w", err)
				}
				req.TemplateID = n
			case "cron_format":
				req.CronFormat = value
			case "active":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for active: %w", err)
				}
				req.Active = b
			case "type":
				req.Type = value
			case "run_at":
				t, err := strfmt.ParseDateTime(value)
				if err != nil {
					return fmt.Errorf("invalid value for run_at (use RFC3339): %w", err)
				}
				req.RunAt = t
			default:
				return fmt.Errorf("unknown field %q — valid fields: name, template_id, cron_format, active, type, run_at", key)
			}
		}

		putParams := schedule.NewPutProjectProjectIDSchedulesScheduleIDParams()
		putParams.ProjectID = int64(pid)
		putParams.ScheduleID = id
		putParams.Schedule = req

		if _, err := apiClient.Schedule.PutProjectProjectIDSchedulesScheduleID(putParams, nil); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}

		style.Success(fmt.Sprintf("Updated schedule %d", id))
		return nil
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleUpdateCmd)
}
