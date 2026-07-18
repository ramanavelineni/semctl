package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var envListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List environments",
	Example: "  semctl env list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("environments",
			[]string{"ID", "Name"},
			func() ([]*models.Environment, error) {
				params := variable_group.NewGetProjectProjectIDEnvironmentParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.VariableGroup.GetProjectProjectIDEnvironment(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(e *models.Environment) []string {
				return []string{
					strconv.FormatInt(e.ID, 10),
					e.Name,
				}
			})
	},
}

func init() {
	envCmd.AddCommand(envListCmd)
}
