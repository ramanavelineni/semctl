package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/spf13/cobra"
)

var inventoryListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List inventories",
	Example: "  semctl inventory list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := inventory.NewGetProjectProjectIDInventoryParams()
		params.ProjectID = int64(pid)

		resp, err := apiClient.Inventory.GetProjectProjectIDInventory(params, nil)
		if err != nil {
			return fmt.Errorf("failed to list inventories: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name", "Type", "SSH Key ID"}
		var rows [][]string
		for _, inv := range items {
			rows = append(rows, []string{
				strconv.FormatInt(inv.ID, 10),
				inv.Name,
				inv.Type,
				strconv.FormatInt(inv.SSHKeyID, 10),
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no inventories found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryListCmd)
}
