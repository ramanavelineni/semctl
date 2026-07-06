package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"register"},
	Short:   "Create (register) a runner",
	Long: `Create a runner and print its registration token.

The token (and private key, when the server generates one) is shown ONCE —
store it safely; it is what the runner process uses to authenticate.`,
	Example: `  semctl runner create --name build-runner
  semctl runner create --name gpu-runner --tags gpu,cuda --max-parallel-tasks 2
  semctl runner create --name proj-runner -p 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		maxParallel, _ := cmd.Flags().GetInt64("max-parallel-tasks")
		webhook, _ := cmd.Flags().GetString("webhook")
		inactive, _ := cmd.Flags().GetBool("inactive")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		req := &models.RunnerRequest{
			Name:             name,
			Active:           !inactive,
			Tags:             tags,
			MaxParallelTasks: maxParallel,
			Webhook:          webhook,
		}
		if req.Tags == nil {
			req.Tags = []string{}
		}

		var created *models.RunnerWithToken
		if projectScoped {
			req.ProjectID = pid
			params := runner.NewPostProjectProjectIDRunnersParams()
			params.ProjectID = pid
			params.Runner = req
			resp, err := apiClient.Runner.PostProjectProjectIDRunners(params, nil)
			if err != nil {
				return fmt.Errorf("failed to create runner: %w", err)
			}
			created = resp.GetPayload()
		} else {
			params := runner.NewPostRunnersParams()
			params.Runner = req
			resp, err := apiClient.Runner.PostRunners(params, nil)
			if err != nil {
				return fmt.Errorf("failed to create runner: %w", err)
			}
			created = resp.GetPayload()
		}

		style.Success(fmt.Sprintf("Created runner %q (ID: %d)", created.Name, created.ID))

		if output.GetFormat() != output.FormatTable {
			output.Print(created, nil, nil)
			return nil
		}

		if created.Token != "" {
			style.Warning("Registration token (shown once — store it safely):")
			fmt.Println(created.Token)
		}
		if created.PrivateKey != "" {
			style.Warning("Private key (shown once — store it safely):")
			fmt.Println(created.PrivateKey)
		}
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerCreateCmd)

	runnerCreateCmd.Flags().String("name", "", "runner name (required)")
	runnerCreateCmd.Flags().StringSlice("tags", nil, "comma-separated runner tags")
	runnerCreateCmd.Flags().Int64("max-parallel-tasks", 0, "maximum parallel tasks (0 = unlimited)")
	runnerCreateCmd.Flags().String("webhook", "", "webhook URL")
	runnerCreateCmd.Flags().Bool("inactive", false, "create the runner in inactive state")
}
