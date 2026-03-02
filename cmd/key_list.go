package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
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

		params := key_store.NewGetProjectProjectIDKeysParams()
		params.ProjectID = int64(pid)

		resp, err := apiClient.KeyStore.GetProjectProjectIDKeys(params, nil)
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name", "Type"}
		var rows [][]string
		for _, k := range items {
			rows = append(rows, []string{
				strconv.FormatInt(k.ID, 10),
				k.Name,
				k.Type,
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no keys found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyListCmd)
}
