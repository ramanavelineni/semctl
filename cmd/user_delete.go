package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/spf13/cobra"
)

var userDeleteCmd = &cobra.Command{
	Use:     "delete <id|username>",
	Aliases: []string{"rm"},
	Short:   "Delete a user",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl user delete 2",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "user", userNameIDs)
		if err != nil {
			return err
		}

		return runDelete(cmd, "user", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := user.NewDeleteUsersUserIDParams()
			params.UserID = id
			_, err = apiClient.User.DeleteUsersUserID(params, nil)
			return err
		})
	},
}

func init() {
	userCmd.AddCommand(userDeleteCmd)
}
