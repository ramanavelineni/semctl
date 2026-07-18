package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/spf13/cobra"
)

var templateDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a template",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl template delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "template")
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "template", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := template.NewDeleteProjectProjectIDTemplatesTemplateIDParams()
			params.ProjectID = int64(pid)
			params.TemplateID = id
			_, err = apiClient.Template.DeleteProjectProjectIDTemplatesTemplateID(params, nil)
			return err
		})
	},
}

func init() {
	templateCmd.AddCommand(templateDeleteCmd)
}
