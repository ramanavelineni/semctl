package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/authentication"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var tokenListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List API tokens for the logged-in user",
	Long: `List API tokens for the logged-in user.

The token ID IS the bearer token — treat this output as sensitive.`,
	Example: "  semctl token list",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("tokens",
			[]string{"ID", "User ID", "Created", "Expired"},
			func() ([]*models.APIToken, error) {
				resp, err := apiClient.Authentication.GetUserTokens(authentication.NewGetUserTokensParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(t *models.APIToken) []string {
				return []string{
					t.ID,
					strconv.FormatInt(t.UserID, 10),
					t.Created,
					strconv.FormatBool(t.Expired),
				}
			})
	},
}

func init() {
	tokenCmd.AddCommand(tokenListCmd)
}
