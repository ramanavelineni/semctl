package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var userListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List users (admin only)",
	Example: "  semctl user list",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("users",
			[]string{"ID", "Username", "Name", "Email", "Admin", "External", "Alert"},
			func() ([]*models.User, error) {
				resp, err := apiClient.User.GetUsers(user.NewGetUsersParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(u *models.User) []string {
				return []string{
					strconv.FormatInt(u.ID, 10),
					u.Username,
					u.Name,
					u.Email,
					strconv.FormatBool(u.Admin),
					strconv.FormatBool(u.External),
					strconv.FormatBool(u.Alert),
				}
			})
	},
}

func init() {
	userCmd.AddCommand(userListCmd)
}
