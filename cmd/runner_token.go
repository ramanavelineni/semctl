package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerTokenCmd = &cobra.Command{
	Use:     "token <id>",
	Short:   "Generate a new registration token for a runner",
	Long:    `Generate a new registration token, replacing the previous one.`,
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner token 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid runner ID: %w", err)
		}

		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		if err := confirmAction(cmd, fmt.Sprintf("Generate a new registration token for runner %d? The previous token stops working.", id)); err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		var tok *models.RunnerRegistrationToken
		if projectScoped {
			params := runner.NewPostProjectProjectIDRunnersRunnerIDRegistrationTokenParams()
			params.ProjectID = pid
			params.RunnerID = id
			resp, err := apiClient.Runner.PostProjectProjectIDRunnersRunnerIDRegistrationToken(params, nil)
			if err != nil {
				return fmt.Errorf("failed to generate registration token: %w", err)
			}
			tok = resp.GetPayload()
		} else {
			params := runner.NewPostRunnersRunnerIDRegistrationTokenParams()
			params.RunnerID = id
			resp, err := apiClient.Runner.PostRunnersRunnerIDRegistrationToken(params, nil)
			if err != nil {
				return fmt.Errorf("failed to generate registration token: %w", err)
			}
			tok = resp.GetPayload()
		}

		if output.GetFormat() != output.FormatTable {
			output.Print(tok, nil, nil)
			return nil
		}

		style.Success(fmt.Sprintf("New registration token for runner %d:", id))
		fmt.Println(tok.RegistrationToken)
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerTokenCmd)
}
