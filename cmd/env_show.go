package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/spf13/cobra"
)

var envShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show environment details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl env show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid environment ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := variable_group.NewGetProjectProjectIDEnvironmentEnvironmentIDParams()
		params.ProjectID = int64(pid)
		params.EnvironmentID = id

		resp, err := apiClient.VariableGroup.GetProjectProjectIDEnvironmentEnvironmentID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get environment: %w", err)
		}

		e := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(e, nil, nil)
			return nil
		}

		password := ""
		if e.Password != "" {
			password = strings.Repeat("*", 8)
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(e.ID, 10)},
			{"Name", e.Name},
			{"Project ID", strconv.FormatInt(e.ProjectID, 10)},
			{"JSON", e.JSON},
			{"Env", e.Env},
			{"Password", password},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	envCmd.AddCommand(envShowCmd)
}
