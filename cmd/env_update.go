package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var envUpdateCmd = &cobra.Command{
	Use:   "update <id> [field=value...]",
	Short: "Update an environment",
	Long:  `Update an environment. Fields: name, json, env, password.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl env update 1 name="Production v2"
  semctl env update 2 json='{"db_host":"10.0.0.5"}' password=newsecret`,
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

		// Fetch current environment
		getParams := variable_group.NewGetProjectProjectIDEnvironmentEnvironmentIDParams()
		getParams.ProjectID = int64(pid)
		getParams.EnvironmentID = id
		getResp, err := apiClient.VariableGroup.GetProjectProjectIDEnvironmentEnvironmentID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get environment: %w", err)
		}
		e := getResp.GetPayload()

		req := &models.EnvironmentRequest{
			ID:        e.ID,
			ProjectID: int64(pid),
			Name:      e.Name,
			JSON:      e.JSON,
			Env:       e.Env,
			Password:  e.Password,
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
			case "json":
				req.JSON = value
			case "env":
				req.Env = value
			case "password":
				req.Password = value
			default:
				return fmt.Errorf("unknown field %q — valid fields: name, json, env, password", key)
			}
		}

		putParams := variable_group.NewPutProjectProjectIDEnvironmentEnvironmentIDParams()
		putParams.ProjectID = int64(pid)
		putParams.EnvironmentID = id
		putParams.Environment = req

		_, err = apiClient.VariableGroup.PutProjectProjectIDEnvironmentEnvironmentID(putParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update environment: %w", err)
		}

		style.Success(fmt.Sprintf("Updated environment %d", id))
		return nil
	},
}

func init() {
	envCmd.AddCommand(envUpdateCmd)
}
