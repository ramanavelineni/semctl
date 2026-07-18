package cmd

import (
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/spf13/cobra"
)

var inventoryDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete an inventory",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl inventory delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg(args[0], "inventory")
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		return runDelete(cmd, "inventory", id, func() error {
			apiClient, err := client.NewAuthenticatedClient()
			if err != nil {
				return err
			}
			params := inventory.NewDeleteProjectProjectIDInventoryInventoryIDParams()
			params.ProjectID = int64(pid)
			params.InventoryID = id
			_, err = apiClient.Inventory.DeleteProjectProjectIDInventoryInventoryID(params, nil)
			return err
		})
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryDeleteCmd)
}
