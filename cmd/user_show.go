package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var userShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show a user (defaults to the authenticated user)",
	Args:  cobra.MaximumNArgs(1),
	Example: `  semctl user show        # current user
  semctl user show me     # same
  semctl user show 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runShow("user",
			func() (*models.User, error) {
				if len(args) == 0 || args[0] == "me" {
					resp, err := apiClient.User.GetUser(user.NewGetUserParams(), nil)
					if err != nil {
						return nil, err
					}
					return resp.GetPayload(), nil
				}
				id, err := parseIDArg(args[0], "user")
				if err != nil {
					return nil, err
				}
				params := user.NewGetUsersUserIDParams()
				params.UserID = id
				resp, err := apiClient.User.GetUsersUserID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(u *models.User) [][]string {
				return [][]string{
					{"ID", strconv.FormatInt(u.ID, 10)},
					{"Username", u.Username},
					{"Name", u.Name},
					{"Email", u.Email},
					{"Admin", strconv.FormatBool(u.Admin)},
					{"External", strconv.FormatBool(u.External)},
					{"Alert", strconv.FormatBool(u.Alert)},
					{"Created", u.Created},
				}
			})
	},
}

func init() {
	userCmd.AddCommand(userShowCmd)
}
