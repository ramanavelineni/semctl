package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/spf13/cobra"
)

var keyDeleteCmd = &cobra.Command{
	Use:     "delete <id|name>",
	Aliases: []string{"rm"},
	Short:   "Delete an access key",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl key delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "key", keyNameIDs)
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "key", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := key_store.NewDeleteProjectProjectIDKeysKeyIDParams()
			params.ProjectID = int64(pid)
			params.KeyID = id
			_, err = apiClient.KeyStore.DeleteProjectProjectIDKeysKeyID(params, nil)
			return err
		})
	},
}

func init() {
	keyCmd.AddCommand(keyDeleteCmd)
}
