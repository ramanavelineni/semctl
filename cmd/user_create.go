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
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a user (admin only)",
	Example: `  semctl user create
  semctl user create --username jdoe --name "Jane Doe" --email jdoe@example.com --password-stdin < pass.txt
  semctl user create --username ops --name Ops --email ops@example.com --admin --password-stdin < pass.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		name, _ := cmd.Flags().GetString("name")
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		passwordStdin, _ := cmd.Flags().GetBool("password-stdin")
		admin, _ := cmd.Flags().GetBool("admin")

		if password != "" && passwordStdin {
			return fmt.Errorf("cannot use --password and --password-stdin together")
		}
		if passwordStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			password = strings.TrimRight(string(data), "\r\n")
		}

		inputsMissing := username == "" || name == "" || email == "" || password == ""
		interactive, err := shouldAutoInteractive(cmd, inputsMissing)
		if err != nil {
			return err
		}
		if interactive && !passwordStdin {
			form := newForm(
				huh.NewGroup(
					huh.NewInput().Title("Username").Value(&username).
						Validate(requireValue("username")),
					huh.NewInput().Title("Full name").Value(&name).
						Validate(requireValue("name")),
					huh.NewInput().Title("Email").Value(&email).
						Validate(requireValue("email")),
					huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&password).
						Validate(requireValue("password")),
					huh.NewConfirm().Title("Admin?").Value(&admin),
				).Title("New user"),
			)
			if err := form.Run(); err != nil {
				return err
			}
		}

		if username == "" || name == "" || email == "" || password == "" {
			return fmt.Errorf("--username, --name, --email, and a password (--password-stdin or --password) are required in non-interactive mode")
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		req := &models.UserRequest{
			Username: username,
			Name:     name,
			Email:    email,
			Password: strfmt.Password(password),
			Admin:    admin,
		}

		params := user.NewPostUsersParams()
		params.User = req

		resp, err := apiClient.User.PostUsers(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		u := resp.GetPayload()
		style.Success(fmt.Sprintf("Created user %q (ID: %d)", u.Username, u.ID))
		return nil
	},
}

// requireValue returns a huh validator that rejects blank input.
func requireValue(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

func init() {
	userCmd.AddCommand(userCreateCmd)

	userCreateCmd.Flags().String("username", "", "login username (required)")
	userCreateCmd.Flags().String("name", "", "full name (required)")
	userCreateCmd.Flags().String("email", "", "email address (required)")
	userCreateCmd.Flags().String("password", "", "password (prefer --password-stdin)")
	userCreateCmd.Flags().Bool("password-stdin", false, "read the password from stdin")
	userCreateCmd.Flags().Bool("admin", false, "grant admin privileges")
}
