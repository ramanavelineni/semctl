package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var projectUpdateCmd = &cobra.Command{
	Use:   "update [field=value...]",
	Short: "Update a project",
	Long:  `Update a project. Uses --project flag or config default for project ID. Fields: name, type, alert, alert_chat, max_parallel_tasks.`,
	Example: `  semctl project update name="New Name"
  semctl project update -p 1 alert=true alert_chat="#ops"
  semctl project update max_parallel_tasks=5`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		id := int64(pid)

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		// Fetch current project
		getParams := project.NewGetProjectProjectIDParams()
		getParams.ProjectID = id
		getResp, err := apiClient.Project.GetProjectProjectID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}
		p := getResp.GetPayload()

		// Build request from current values
		req := models.ProjectRequest{
			Name:             p.Name,
			Type:             p.Type,
			Alert:            p.Alert,
			AlertChat:        p.AlertChat,
			MaxParallelTasks: p.MaxParallelTasks,
		}

		// Apply field=value overrides
		if len(args) == 0 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		for _, arg := range args {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			switch key {
			case "name":
				req.Name = value
			case "type":
				req.Type = value
			case "alert":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for alert: %w", err)
				}
				req.Alert = b
			case "alert_chat":
				req.AlertChat = value
			case "max_parallel_tasks":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for max_parallel_tasks: %w", err)
				}
				req.MaxParallelTasks = &n
			default:
				return fmt.Errorf("unknown field %q — valid fields: name, type, alert, alert_chat, max_parallel_tasks", key)
			}
		}

		body := project.PutProjectProjectIDBody{
			ProjectRequest: req,
			ID:             id,
		}

		putParams := project.NewPutProjectProjectIDParams()
		putParams.ProjectID = id
		putParams.Project = body

		_, err = apiClient.Project.PutProjectProjectID(putParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update project: %w", err)
		}

		style.Success(fmt.Sprintf("Updated project %d", id))
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectUpdateCmd)
}
