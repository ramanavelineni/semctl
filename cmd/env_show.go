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
				e := resp.GetPayload()
				// Redact in every output format, not just the table — the
				// mask keeps the "a password is set" signal without the value.
				if e.Password != "" {
					e.Password = strings.Repeat("*", 8)
				}
				return e, nil
			},
			func(e *models.Environment) [][]string {
				return [][]string{
					{"ID", strconv.FormatInt(e.ID, 10)},
					{"Name", e.Name},
					{"Project ID", strconv.FormatInt(e.ProjectID, 10)},
					{"JSON", e.JSON},
					{"Env", e.Env},
					{"Password", e.Password},
				}
			})
	},
}

func init() {
	envCmd.AddCommand(envShowCmd)
}
