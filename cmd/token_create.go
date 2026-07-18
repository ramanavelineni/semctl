package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/authentication"
	"github.com/spf13/cobra"
)

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an API token for the logged-in user",
	Long: `Create an API token for the logged-in user.

The token is printed to stdout so it can be captured; store it safely —
it grants the same access as your login.`,
	Example: `  semctl token create
  TOKEN=$(semctl token create)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		resp, err := apiClient.Authentication.PostUserTokens(authentication.NewPostUserTokensParams(), nil)
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}

		t := resp.GetPayload()
		style.Success("Created API token.")
		if output.GetFormat() != output.FormatTable {
			output.Print(t, nil, nil)
			return nil
		}
		// Bare token on stdout, pipeable (like runner token).
		fmt.Println(t.ID)
		return nil
	},
}

func init() {
	tokenCmd.AddCommand(tokenCreateCmd)
}
