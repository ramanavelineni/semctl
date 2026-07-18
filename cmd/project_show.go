package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var projectShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show project details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl project show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "project")
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runShow("project",
			func() (*models.Project, error) {
				params := project.NewGetProjectProjectIDParams()
				params.ProjectID = id
				resp, err := apiClient.Project.GetProjectProjectID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(p *models.Project) [][]string {
				maxParallel := "" // nil = unset; "0" would fake an explicit value
				if p.MaxParallelTasks != nil {
					maxParallel = strconv.FormatInt(*p.MaxParallelTasks, 10)
				}
				return [][]string{
					{"ID", strconv.FormatInt(p.ID, 10)},
					{"Name", p.Name},
					{"Type", p.Type},
					{"Alert", strconv.FormatBool(p.Alert)},
					{"Alert Chat", strDeref(p.AlertChat)},
					{"Max Parallel Tasks", maxParallel},
					{"Created", p.Created},
				}
			})
	},
}

func init() {
	projectCmd.AddCommand(projectShowCmd)
}

// strDeref returns the string behind p, or "" when nil.
func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
