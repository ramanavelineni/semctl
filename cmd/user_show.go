package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var userShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show a user (defaults to the authenticated user)",
	Args:  cobra.MaximumNArgs(1),
	Example: `  semctl user show        # current user
  semctl user show 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		var u *models.User
		if len(args) == 0 || args[0] == "me" {
			resp, err := apiClient.User.GetUser(user.NewGetUserParams(), nil)
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}
			u = resp.GetPayload()
		} else {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}
			params := user.NewGetUsersUserIDParams()
			params.UserID = id
			resp, err := apiClient.User.GetUsersUserID(params, nil)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}
			u = resp.GetPayload()
		}

		if output.GetFormat() != output.FormatTable {
			output.Print(u, nil, nil)
			return nil
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(u.ID, 10)},
			{"Username", u.Username},
			{"Name", u.Name},
			{"Email", u.Email},
			{"Admin", strconv.FormatBool(u.Admin)},
			{"External", strconv.FormatBool(u.External)},
			{"Alert", strconv.FormatBool(u.Alert)},
			{"Created", u.Created},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	userCmd.AddCommand(userShowCmd)
}
