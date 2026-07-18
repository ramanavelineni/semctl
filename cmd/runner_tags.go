package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var runnerTagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List runner tags",
	Example: `  semctl runner tags
  semctl runner tags -p 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, projectScoped, err := runnerScope(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("runner tags",
			[]string{"Tag", "Runners"},
			func() ([]*models.RunnerTag, error) {
				if projectScoped {
					params := runner.NewGetProjectProjectIDRunnerTagsParams()
					params.ProjectID = pid
					resp, err := apiClient.Runner.GetProjectProjectIDRunnerTags(params, nil)
					if err != nil {
						return nil, err
					}
					return resp.GetPayload(), nil
				}
				resp, err := apiClient.Runner.GetRunnerTags(runner.NewGetRunnerTagsParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(t *models.RunnerTag) []string {
				return []string{
					t.Tag,
					strconv.FormatInt(t.NumberOfRunners, 10),
				}
			})
	},
}

func init() {
	runnerCmd.AddCommand(runnerTagsCmd)
}
