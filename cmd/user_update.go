package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var userUpdateCmd = &cobra.Command{
	Use:   "update <id> <field=value>...",
	Short: "Update a user",
	Long: `Update user fields using field=value pairs.

Supported fields: username, name, email, admin, alert`,
	Args: cobra.MinimumNArgs(2),
	Example: `  semctl user update 2 name="Jane Doe"
  semctl user update 2 admin=true alert=false`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		getParams := user.NewGetUsersUserIDParams()
		getParams.UserID = id
		getResp, err := apiClient.User.GetUsersUserID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		current := getResp.GetPayload()

		req := &models.UserPutRequest{
			Username: current.Username,
			Name:     current.Name,
			Email:    current.Email,
			Admin:    current.Admin,
			Alert:    current.Alert,
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			switch key {
			case "username":
				req.Username = value
			case "name":
				req.Name = value
			case "email":
				req.Email = value
			case "admin":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for admin: %w", err)
				}
				req.Admin = b
			case "alert":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid value for alert: %w", err)
				}
				req.Alert = b
			default:
				return fmt.Errorf("unknown field %q (supported: username, name, email, admin, alert)", key)
			}
		}

		params := user.NewPutUsersUserIDParams()
		params.UserID = id
		params.User = req

		if _, err := apiClient.User.PutUsersUserID(params, nil); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		style.Success(fmt.Sprintf("Updated user %d", id))
		return nil
	},
}

func init() {
	userCmd.AddCommand(userUpdateCmd)
}
