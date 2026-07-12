package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/spf13/cobra"
)

var userDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a user",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl user delete 2",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		if err := confirmAction(cmd, fmt.Sprintf("Delete user %d?", id)); err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := user.NewDeleteUsersUserIDParams()
		params.UserID = id

		if _, err := apiClient.User.DeleteUsersUserID(params, nil); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		style.Success(fmt.Sprintf("Deleted user %d", id))
		return nil
	},
}

func init() {
	userCmd.AddCommand(userDeleteCmd)
}
