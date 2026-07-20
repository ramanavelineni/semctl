package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/go-openapi/strfmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/spf13/cobra"
)

var userPasswordCmd = &cobra.Command{
	Use:   "password <id|username>",
	Short: "Change a user's password",
	Args:  cobra.ExactArgs(1),
	Example: `  semctl user password 2                       # prompts on a terminal
  semctl user password 2 --password-stdin < pass.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "user", userNameIDs)
		if err != nil {
			return err
		}

		passwordStdin, _ := cmd.Flags().GetBool("password-stdin")

		var password string
		if passwordStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			password = strings.TrimRight(string(data), "\r\n")
			if password == "" {
				return fmt.Errorf("empty password on stdin")
			}
		} else {
			interactive, err := shouldAutoInteractive(cmd, true)
			if err != nil {
				return err
			}
			if !interactive {
				return fmt.Errorf("no terminal available: pass the new password via --password-stdin")
			}
			if err := runForm(newForm(
				huh.NewGroup(
					huh.NewInput().Title("New password").EchoMode(huh.EchoModePassword).Value(&password).
						Validate(requireValue("password")),
				),
			)); err != nil {
				return err
			}
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := user.NewPostUsersUserIDPasswordParams()
		params.UserID = id
		params.Password = user.PostUsersUserIDPasswordBody{Password: strfmt.Password(password)}

		if _, err := apiClient.User.PostUsersUserIDPassword(params, nil); err != nil {
			return fmt.Errorf("failed to change password: %w", err)
		}

		style.Success(fmt.Sprintf("Password changed for user %d", id))
		return nil
	},
}

func init() {
	userCmd.AddCommand(userPasswordCmd)

	userPasswordCmd.Flags().Bool("password-stdin", false, "read the new password from stdin")
}
