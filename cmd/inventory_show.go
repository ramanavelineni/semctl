package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/spf13/cobra"
)

var inventoryShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show inventory details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl inventory show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid inventory ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := inventory.NewGetProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = int64(pid)
		params.InventoryID = id

		resp, err := apiClient.Inventory.GetProjectProjectIDInventoryInventoryID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get inventory: %w", err)
		}

		inv := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(inv, nil, nil)
			return nil
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(inv.ID, 10)},
			{"Name", inv.Name},
			{"Type", inv.Type},
			{"SSH Key ID", strconv.FormatInt(inv.SSHKeyID, 10)},
			{"Become Key ID", strconv.FormatInt(inv.BecomeKeyID, 10)},
			{"Repository ID", strconv.FormatInt(inv.RepositoryID, 10)},
			{"Inventory", inv.Inventory},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryShowCmd)
}
