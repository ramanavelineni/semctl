package cmd

import (
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List runners",
	Example: `  semctl runner list
  semctl runner list -p 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("runners",
			[]string{"ID", "Name", "Active", "Registered", "Default", "Max Parallel", "Tags"},
			func() ([]*models.Runner, error) {
				if projectScoped {
					params := runner.NewGetProjectProjectIDRunnersParams()
					params.ProjectID = pid
					resp, err := apiClient.Runner.GetProjectProjectIDRunners(params, nil)
					if err != nil {
						return nil, err
					}
					return resp.GetPayload(), nil
				}
				resp, err := apiClient.Runner.GetRunners(runner.NewGetRunnersParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(r *models.Runner) []string {
				return []string{
					strconv.FormatInt(r.ID, 10),
					r.Name,
					strconv.FormatBool(r.Active),
					strconv.FormatBool(r.Registered),
					strconv.FormatBool(r.IsDefault),
					strconv.FormatInt(r.MaxParallelTasks, 10),
					strings.Join(r.Tags, ","),
				}
			})
	},
}

func init() {
	runnerCmd.AddCommand(runnerListCmd)
}
