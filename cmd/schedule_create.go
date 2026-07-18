package cmd

import (
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var scheduleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a schedule",
	Example: `  semctl schedule create --name "Nightly Deploy" --template-id 1 --cron-format "0 2 * * *"
  semctl schedule create --name "Once" --template-id 1 --type run_at --run-at 2026-08-01T02:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		templateID, _ := cmd.Flags().GetInt64("template-id")
		cronFormat, _ := cmd.Flags().GetString("cron-format")
		active, _ := cmd.Flags().GetBool("active")
		schedType, _ := cmd.Flags().GetString("type")
		runAt, _ := cmd.Flags().GetString("run-at")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if templateID == 0 {
			return fmt.Errorf("--template-id is required")
		}
		if cronFormat == "" && runAt == "" {
			return fmt.Errorf("--cron-format is required (or --type run_at with --run-at)")
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		req := &models.ScheduleRequest{
			Name:       name,
			ProjectID:  int64(pid),
			TemplateID: templateID,
			CronFormat: cronFormat,
			Active:     active,
			Type:       schedType,
		}
		if runAt != "" {
			t, err := strfmt.ParseDateTime(runAt)
			if err != nil {
				return fmt.Errorf("invalid --run-at %q (use RFC3339, e.g. 2026-08-01T02:00:00Z): %w", runAt, err)
			}
			req.RunAt = t
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := schedule.NewPostProjectProjectIDSchedulesParams()
		params.ProjectID = int64(pid)
		params.Schedule = req

		resp, err := apiClient.Schedule.PostProjectProjectIDSchedules(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create schedule: %w", err)
		}

		s := resp.GetPayload()
		style.Success(fmt.Sprintf("Created schedule %q (ID: %d)", s.Name, s.ID))
		// Machine-readable resource on stdout so pipelines can capture the ID.
		if output.GetFormat() != output.FormatTable {
			output.Print(s, nil, nil)
		}
		return nil
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleCreateCmd)

	scheduleCreateCmd.Flags().String("name", "", "schedule name (required)")
	scheduleCreateCmd.Flags().Int64("template-id", 0, "template ID to run (required)")
	scheduleCreateCmd.Flags().String("cron-format", "", "cron expression (e.g. \"0 2 * * *\")")
	scheduleCreateCmd.Flags().Bool("active", true, "schedule is active")
	scheduleCreateCmd.Flags().String("type", "", "schedule type (\"\" for cron, \"run_at\" for one-shot)")
	scheduleCreateCmd.Flags().String("run-at", "", "one-shot run time (RFC3339, with --type run_at)")
}
