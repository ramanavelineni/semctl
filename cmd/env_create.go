package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var envCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an environment",
	Example: `  semctl env create --name "Production"
  semctl env create --name "Staging" --json-vars '{"key": "value"}' --env '{"VAR": "val"}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		jsonVars, _ := cmd.Flags().GetString("json-vars")
		env, _ := cmd.Flags().GetString("env")
		password, _ := cmd.Flags().GetString("password")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		req := &models.EnvironmentRequest{
			ProjectID: int64(pid),
			Name:      name,
			JSON:      jsonVars,
			Env:       env,
			Password:  password,
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := variable_group.NewPostProjectProjectIDEnvironmentParams()
		params.ProjectID = int64(pid)
		params.Environment = req

		resp, err := apiClient.VariableGroup.PostProjectProjectIDEnvironment(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create environment: %w", err)
		}

		e := resp.GetPayload()
		style.Success(fmt.Sprintf("Created environment %q (ID: %d)", e.Name, e.ID))
		return nil
	},
}

func init() {
	envCmd.AddCommand(envCreateCmd)

	envCreateCmd.Flags().String("name", "", "environment name (required)")
	envCreateCmd.Flags().String("json-vars", "", "environment JSON variables")
	envCreateCmd.Flags().String("env", "", "extra environment variables")
	envCreateCmd.Flags().String("password", "", "environment password")
}
