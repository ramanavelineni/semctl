package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Example: `  semctl project create --name "My Project"
  semctl project create --name "My Project" --alert --max-parallel-tasks 5`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		projType, _ := cmd.Flags().GetString("type")
		alert, _ := cmd.Flags().GetBool("alert")
		alertChat, _ := cmd.Flags().GetString("alert-chat")
		maxParallel, _ := cmd.Flags().GetInt64("max-parallel-tasks")

		interactive, err := shouldAutoInteractive(cmd, name == "")
		if err != nil {
			return err
		}
		if interactive {
			if err := runForm(newForm(
				huh.NewGroup(
					huh.NewInput().Title("Project name").Value(&name).
						Validate(requireValue("name")),
					huh.NewConfirm().Title("Enable alerts?").Value(&alert),
				).Title("New project"),
			)); err != nil {
				return err
			}
		}

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		req := &models.ProjectRequest{
			Name:      name,
			Type:      projType,
			Alert:     alert,
			AlertChat: &alertChat,
		}
		if cmd.Flags().Changed("max-parallel-tasks") {
			req.MaxParallelTasks = &maxParallel
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := project.NewPostProjectsParams()
		params.Project = req

		resp, err := apiClient.Project.PostProjects(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		p := resp.GetPayload()
		style.Success(fmt.Sprintf("Created project %q (ID: %d)", p.Name, p.ID))
		// Machine-readable resource on stdout so pipelines can capture the ID.
		if output.GetFormat() != output.FormatTable {
			output.Print(p, nil, nil)
		}
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectCreateCmd)

	projectCreateCmd.Flags().String("name", "", "project name (required)")
	projectCreateCmd.Flags().String("type", "", "project type")
	projectCreateCmd.Flags().Bool("alert", false, "enable alerts")
	projectCreateCmd.Flags().String("alert-chat", "", "alert chat channel")
	projectCreateCmd.Flags().Int64("max-parallel-tasks", 0, "max parallel tasks (0 = unlimited)")
}
