package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var templateUpdateCmd = &cobra.Command{
	Use:   "update <id> [field=value...]",
	Short: "Update a template",
	Long:  `Update a template. Fields: name, description, type, app, playbook, git_branch, repository_id, environment_id, inventory_id, build_template_id, view_id, autorun, suppress_success_alerts.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl template update 1 name="New Name"
  semctl template update 5 playbook=deploy.yml git_branch=main
  semctl template update 3 autorun=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid template ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		// Fetch current template
		getParams := template.NewGetProjectProjectIDTemplatesTemplateIDParams()
		getParams.ProjectID = int64(pid)
		getParams.TemplateID = id
		getResp, err := apiClient.Template.GetProjectProjectIDTemplatesTemplateID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
		t := getResp.GetPayload()

		// Build request from current values
		req := &models.TemplateRequest{
			ID:                      t.ID,
			ProjectID:               t.ProjectID,
			Name:                    t.Name,
			Description:             t.Description,
			Type:                    t.Type,
			App:                     t.App,
			Playbook:                t.Playbook,
			GitBranch:               t.GitBranch,
			RepositoryID:            t.RepositoryID,
			EnvironmentID:           t.EnvironmentID,
			InventoryID:             t.InventoryID,
			BuildTemplateID:         t.BuildTemplateID,
			ViewID:                  t.ViewID,
			Autorun:                 t.Autorun,
			SuppressSuccessAlerts:   t.SuppressSuccessAlerts,
			AllowOverrideArgsInTask: t.AllowOverrideArgsInTask,
			StartVersion:            t.StartVersion,
			Arguments:               t.Arguments,
			SurveyVars:             []*models.TemplateSurveyVar{},
			Vaults:                 []*models.TemplateVault{},
		}

		if len(args) < 2 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			switch key {
			case "name":
				req.Name = value
			case "description":
				req.Description = value
			case "type":
				req.Type = value
			case "app":
				req.App = value
			case "playbook":
				req.Playbook = value
			case "git_branch":
				req.GitBranch = value
			case "arguments":
				req.Arguments = value
			case "repository_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for repository_id: %w", err)
				}
				req.RepositoryID = n
			case "environment_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for environment_id: %w", err)
				}
				req.EnvironmentID = n
			case "inventory_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for inventory_id: %w", err)
				}
				req.InventoryID = n
			case "build_template_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for build_template_id: %w", err)
				}
				req.BuildTemplateID = n
			case "view_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for view_id: %w", err)
				}
				req.ViewID = n
			case "autorun":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for autorun: %w", err)
				}
				req.Autorun = b
			case "suppress_success_alerts":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for suppress_success_alerts: %w", err)
				}
				req.SuppressSuccessAlerts = b
			default:
				return fmt.Errorf("unknown field %q", key)
			}
		}

		putParams := template.NewPutProjectProjectIDTemplatesTemplateIDParams()
		putParams.ProjectID = int64(pid)
		putParams.TemplateID = id
		putParams.Template = req

		_, err = apiClient.Template.PutProjectProjectIDTemplatesTemplateID(putParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update template: %w", err)
		}

		style.Success(fmt.Sprintf("Updated template %d", id))
		return nil
	},
}

func init() {
	templateCmd.AddCommand(templateUpdateCmd)
}
