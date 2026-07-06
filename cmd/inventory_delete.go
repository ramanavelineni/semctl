package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
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
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid inventory ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		if err := confirmAction(cmd, fmt.Sprintf("Delete inventory %d?", id)); err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := inventory.NewDeleteProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = int64(pid)
		params.InventoryID = id

		_, err = apiClient.Inventory.DeleteProjectProjectIDInventoryInventoryID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to delete inventory: %w", err)
		}

		style.Success(fmt.Sprintf("Deleted inventory %d", id))
		return nil
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryDeleteCmd)
}
