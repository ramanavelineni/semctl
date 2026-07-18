package cmd

import (
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var envShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show environment details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl env show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "environment")
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

		return runShow("environment",
			func() (*models.Environment, error) {
				params := variable_group.NewGetProjectProjectIDEnvironmentEnvironmentIDParams()
				params.ProjectID = int64(pid)
				params.EnvironmentID = id
				resp, err := apiClient.VariableGroup.GetProjectProjectIDEnvironmentEnvironmentID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(e *models.Environment) [][]string {
				password := ""
				if e.Password != "" {
					password = strings.Repeat("*", 8)
				}
				return [][]string{
					{"ID", strconv.FormatInt(e.ID, 10)},
					{"Name", e.Name},
					{"Project ID", strconv.FormatInt(e.ProjectID, 10)},
					{"JSON", e.JSON},
					{"Env", e.Env},
					{"Password", password},
				}
			})
	},
}

func init() {
	envCmd.AddCommand(envShowCmd)
}
