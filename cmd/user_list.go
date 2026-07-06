package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
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

		resp, err := apiClient.User.GetUsers(user.NewGetUsersParams(), nil)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Username", "Name", "Email", "Admin", "External", "Alert"}
		var rows [][]string
		for _, u := range items {
			rows = append(rows, []string{
				strconv.FormatInt(u.ID, 10),
				u.Username,
				u.Name,
				u.Email,
				strconv.FormatBool(u.Admin),
				strconv.FormatBool(u.External),
				strconv.FormatBool(u.Alert),
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no users found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	userCmd.AddCommand(userListCmd)
}
