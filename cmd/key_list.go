package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var keyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List access keys",
	Example: "  semctl key list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runList("keys",
			[]string{"ID", "Name", "Type"},
			func() ([]*models.AccessKey, error) {
				params := key_store.NewGetProjectProjectIDKeysParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.KeyStore.GetProjectProjectIDKeys(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(k *models.AccessKey) []string {
				return []string{
					strconv.FormatInt(k.ID, 10),
					k.Name,
					k.Type,
				}
			})
	},
}

func init() {
	keyCmd.AddCommand(keyListCmd)
}
