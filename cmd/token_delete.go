package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/authentication"
	"github.com/spf13/cobra"
)

var tokenDeleteCmd = &cobra.Command{
	Use:     "delete <token-id>",
	Aliases: []string{"rm", "revoke"},
	Short:   "Expire an API token",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl token delete kycs1hbat3japquloopyoxxdiukj7flnh5e2ao9k",
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		return runDeleteNamed(cmd, "token", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := authentication.NewDeleteUserTokensAPITokenIDParams()
			params.APITokenID = id
			_, err = apiClient.Authentication.DeleteUserTokensAPITokenID(params, nil)
			return err
		})
	},
}

func init() {
	tokenCmd.AddCommand(tokenDeleteCmd)
}
