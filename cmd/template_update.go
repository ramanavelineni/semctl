package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/ramanavelineni/semctl/internal/apply"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var templateUpdateCmd = &cobra.Command{
	Use:   "update <id|name> [field=value...]",
	Short: "Update a template",
	Long:  `Update a template. Fields: name, description, type, app, playbook, git_branch, arguments, repository_id, environment_id, inventory_id, build_template_id, view_id, autorun, suppress_success_alerts.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl template update 1 name="New Name"
  semctl template update 5 playbook=deploy.yml git_branch=main
  semctl template update 3 autorun=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "template", templateNameIDs)
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
		}
		apply.PreserveUnmanagedTemplateFields(req, t)

		if len(args) < 2 {
			interactive, ferr := shouldAutoInteractive(cmd, true)
			if ferr != nil {
				return ferr
			}
			if !interactive {
				return fmt.Errorf("no fields to update — provide field=value pairs")
			}
			if err := templateUpdateForm(cmd, req); err != nil {
				return err
			}
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
				return fmt.Errorf("unknown field %q — valid fields: name, description, type, app, playbook, git_branch, arguments, repository_id, environment_id, inventory_id, build_template_id, view_id, autorun, suppress_success_alerts", key)
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

// templateUpdateForm edits req in place, pre-filled with the current values;
// linked resources are chosen from live server listings.
func templateUpdateForm(cmd *cobra.Command, req *models.TemplateRequest) error {
	repoOpts, err := nameIDOptions(cmd, repoNameIDs, false)
	if err != nil {
		return err
	}
	invOpts, err := nameIDOptions(cmd, inventoryNameIDs, true)
	if err != nil {
		return err
	}
	envOpts, err := nameIDOptions(cmd, envNameIDs, true)
	if err != nil {
		return err
	}
	return runForm(newForm(
		huh.NewGroup(
			huh.NewInput().Title("Name").Value(&req.Name).
				Validate(requireValue("name")),
			huh.NewInput().Title("Description").Value(&req.Description),
			huh.NewInput().Title("App").Value(&req.App),
			huh.NewInput().Title("Playbook").Value(&req.Playbook),
			huh.NewInput().Title("Git branch").Value(&req.GitBranch),
			huh.NewSelect[int64]().Title("Repository").Options(repoOpts...).Value(&req.RepositoryID),
			huh.NewSelect[int64]().Title("Inventory").Options(invOpts...).Value(&req.InventoryID),
			huh.NewSelect[int64]().Title("Variable group").Options(envOpts...).Value(&req.EnvironmentID),
			huh.NewConfirm().Title("Autorun").Value(&req.Autorun),
			huh.NewConfirm().Title("Suppress success alerts").Value(&req.SuppressSuccessAlerts),
		).Title("Edit template").Description(moreFlagsNote),
	))
}

func init() {
	templateCmd.AddCommand(templateUpdateCmd)
}
