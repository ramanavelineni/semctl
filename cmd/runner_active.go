package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerActivateCmd = &cobra.Command{
	Use:     "activate <id>",
	Short:   "Activate a runner",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner activate 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setRunnerActive(cmd, args[0], true)
	},
}

var runnerDeactivateCmd = &cobra.Command{
	Use:     "deactivate <id>",
	Short:   "Deactivate a runner",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl runner deactivate 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setRunnerActive(cmd, args[0], false)
	},
}

func setRunnerActive(cmd *cobra.Command, idArg string, active bool) error {
	id, err := strconv.ParseInt(idArg, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid runner ID: %w", err)
	}

	pid, projectScoped, err := runnerScope(cmd)
	if err != nil {
		return err
	}

	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return err
	}

	body := &models.RunnerActive{Active: active}

	if projectScoped {
		params := runner.NewPostProjectProjectIDRunnersRunnerIDActiveParams()
		params.ProjectID = pid
		params.RunnerID = id
		params.Active = body
		if _, err := apiClient.Runner.PostProjectProjectIDRunnersRunnerIDActive(params, nil); err != nil {
			return fmt.Errorf("failed to set runner active state: %w", err)
		}
	} else {
		params := runner.NewPostRunnersRunnerIDActiveParams()
		params.RunnerID = id
		params.Active = body
		if _, err := apiClient.Runner.PostRunnersRunnerIDActive(params, nil); err != nil {
			return fmt.Errorf("failed to set runner active state: %w", err)
		}
	}

	state := "activated"
	if !active {
		state = "deactivated"
	}
	style.Success(fmt.Sprintf("Runner %d %s", id, state))
	return nil
}

func init() {
	runnerCmd.AddCommand(runnerActivateCmd)
	runnerCmd.AddCommand(runnerDeactivateCmd)
}
