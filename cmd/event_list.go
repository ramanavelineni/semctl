package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/operations"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var eventListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List events for your projects",
	Long:    `List events for your projects. Shows the last 200 events by default; use --all for the full history.`,
	Example: `  semctl event list
  semctl event list --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("events",
			[]string{"Project ID", "User ID", "Object Type", "Object ID", "Description"},
			func() ([]*models.Event, error) {
				if all {
					resp, err := apiClient.Operations.GetEvents(operations.NewGetEventsParams(), nil)
					if err != nil {
						return nil, err
					}
					return resp.GetPayload(), nil
				}
				resp, err := apiClient.Operations.GetEventsLast(operations.NewGetEventsLastParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(e *models.Event) []string {
				return []string{
					strconv.FormatInt(e.ProjectID, 10),
					strconv.FormatInt(e.UserID, 10),
					e.ObjectType,
					strconv.FormatInt(e.ObjectID, 10),
					e.Description,
				}
			})
	},
}

func init() {
	eventCmd.AddCommand(eventListCmd)

	eventListCmd.Flags().Bool("all", false, "list all events instead of the last 200")
}
