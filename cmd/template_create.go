package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var templateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new template",
	Example: `  semctl template create --name "Deploy" --playbook deploy.yml --repository-id 1
  semctl template create --name "Build" --type build --app ansible --playbook build.yml --repository-id 1 --environment-id 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		tplType, _ := cmd.Flags().GetString("type")
		app, _ := cmd.Flags().GetString("app")
		playbook, _ := cmd.Flags().GetString("playbook")
		repoID, _ := cmd.Flags().GetInt64("repository-id")
		envID, _ := cmd.Flags().GetInt64("environment-id")
		invID, _ := cmd.Flags().GetInt64("inventory-id")
		gitBranch, _ := cmd.Flags().GetString("git-branch")
		description, _ := cmd.Flags().GetString("description")
		autorun, _ := cmd.Flags().GetBool("autorun")
		buildTplID, _ := cmd.Flags().GetInt64("build-template-id")
		viewID, _ := cmd.Flags().GetInt64("view-id")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		req := &models.TemplateRequest{
			ProjectID:       int64(pid),
			Name:            name,
			Type:            tplType,
			App:             app,
			Playbook:        playbook,
			RepositoryID:    repoID,
			EnvironmentID:   envID,
			InventoryID:     invID,
			GitBranch:       gitBranch,
			Description:     description,
			Autorun:         autorun,
			BuildTemplateID: buildTplID,
			ViewID:          viewID,
			SurveyVars:     []*models.TemplateSurveyVar{},
			Vaults:         []*models.TemplateVault{},
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := template.NewPostProjectProjectIDTemplatesParams()
		params.ProjectID = int64(pid)
		params.Template = req

		resp, err := apiClient.Template.PostProjectProjectIDTemplates(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create template: %w", err)
		}

		t := resp.GetPayload()
		style.Success(fmt.Sprintf("Created template %q (ID: %d)", t.Name, t.ID))
		return nil
	},
}

func init() {
	templateCmd.AddCommand(templateCreateCmd)

	templateCreateCmd.Flags().String("name", "", "template name (required)")
	templateCreateCmd.Flags().String("type", "", "template type (build, deploy)")
	templateCreateCmd.Flags().String("app", "ansible", "app type")
	templateCreateCmd.Flags().String("playbook", "", "playbook filename")
	templateCreateCmd.Flags().Int64("repository-id", 0, "repository ID")
	templateCreateCmd.Flags().Int64("environment-id", 0, "environment ID")
	templateCreateCmd.Flags().Int64("inventory-id", 0, "inventory ID")
	templateCreateCmd.Flags().String("git-branch", "", "git branch")
	templateCreateCmd.Flags().String("description", "", "template description")
	templateCreateCmd.Flags().Bool("autorun", false, "enable autorun")
	templateCreateCmd.Flags().Int64("build-template-id", 0, "build template ID")
	templateCreateCmd.Flags().Int64("view-id", 0, "view ID")
}
