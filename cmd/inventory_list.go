package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
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

		return runList("inventories",
			[]string{"ID", "Name", "Type", "SSH Key ID"},
			func() ([]*models.Inventory, error) {
				params := inventory.NewGetProjectProjectIDInventoryParams()
				params.ProjectID = int64(pid)
				resp, err := apiClient.Inventory.GetProjectProjectIDInventory(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(inv *models.Inventory) []string {
				return []string{
					strconv.FormatInt(inv.ID, 10),
					inv.Name,
					inv.Type,
					strconv.FormatInt(inv.SSHKeyID, 10),
				}
			})
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryListCmd)
}
