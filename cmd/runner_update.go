package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerUpdateCmd = &cobra.Command{
	Use:   "update <id|name> <field=value>...",
	Short: "Update a runner",
	Long: `Update runner fields using field=value pairs.

Supported fields: name, active, max_parallel_tasks, tags (comma-separated), webhook`,
	Args: cobra.MinimumNArgs(1),
	Example: `  semctl runner update 1 name=build-runner
  semctl runner update 1 active=false max_parallel_tasks=4
  semctl runner update 1 tags=gpu,cuda`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "runner", runnerNameIDs)
		if err != nil {
			return err
		}
		if len(args) < 2 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		// Fetch current state, apply overrides, PUT back
		var current *models.Runner
		if projectScoped {
			getParams := runner.NewGetProjectProjectIDRunnersRunnerIDParams()
			getParams.ProjectID = pid
			getParams.RunnerID = id
			resp, err := apiClient.Runner.GetProjectProjectIDRunnersRunnerID(getParams, nil)
			if err != nil {
				return fmt.Errorf("failed to get runner: %w", err)
			}
			current = resp.GetPayload()
		} else {
			getParams := runner.NewGetRunnersRunnerIDParams()
			getParams.RunnerID = id
			resp, err := apiClient.Runner.GetRunnersRunnerID(getParams, nil)
			if err != nil {
				return fmt.Errorf("failed to get runner: %w", err)
			}
			current = resp.GetPayload()
		}

		req := &models.RunnerRequest{
			Name:             current.Name,
			Active:           current.Active,
			Registered:       current.Registered,
			MaxParallelTasks: current.MaxParallelTasks,
			Tags:             current.Tags,
			Webhook:          current.Webhook,
		}
		if req.Tags == nil {
			req.Tags = []string{}
		}
		if projectScoped {
			req.ProjectID = pid
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			key = strings.ReplaceAll(key, "-", "_") // accept kebab-case like the create flags
			switch key {
			case "name":
				req.Name = value
			case "active":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for active: %w", err)
				}
				req.Active = b
			case "max_parallel_tasks":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for max_parallel_tasks: %w", err)
				}
				req.MaxParallelTasks = n
			case "tags":
				if value == "" {
					req.Tags = []string{}
				} else {
					req.Tags = strings.Split(value, ",")
				}
			case "webhook":
				req.Webhook = value
			default:
				return fmt.Errorf("unknown field %q (supported: name, active, max_parallel_tasks, tags, webhook)", key)
			}
		}

		if projectScoped {
			params := runner.NewPutProjectProjectIDRunnersRunnerIDParams()
			params.ProjectID = pid
			params.RunnerID = id
			params.Runner = req
			if _, err := apiClient.Runner.PutProjectProjectIDRunnersRunnerID(params, nil); err != nil {
				return fmt.Errorf("failed to update runner: %w", err)
			}
		} else {
			params := runner.NewPutRunnersRunnerIDParams()
			params.RunnerID = id
			params.Runner = req
			if _, err := apiClient.Runner.PutRunnersRunnerID(params, nil); err != nil {
				return fmt.Errorf("failed to update runner: %w", err)
			}
		}

		style.Success(fmt.Sprintf("Updated runner %d", id))
		return nil
	},
}

func init() {
	runnerCmd.AddCommand(runnerUpdateCmd)
}
