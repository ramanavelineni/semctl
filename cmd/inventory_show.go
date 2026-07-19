package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var inventoryShowCmd = &cobra.Command{
	Use:     "show <id|name>",
	Short:   "Show inventory details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl inventory show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "inventory", inventoryNameIDs)
		if err != nil {
			return err
		}
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runShow("inventory",
			func() (*models.Inventory, error) {
				params := inventory.NewGetProjectProjectIDInventoryInventoryIDParams()
				params.ProjectID = int64(pid)
				params.InventoryID = id
				resp, err := apiClient.Inventory.GetProjectProjectIDInventoryInventoryID(params, nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(inv *models.Inventory) [][]string {
				return [][]string{
					{"ID", strconv.FormatInt(inv.ID, 10)},
					{"Name", inv.Name},
					{"Type", inv.Type},
					{"SSH Key ID", strconv.FormatInt(inv.SSHKeyID, 10)},
					{"Become Key ID", strconv.FormatInt(inv.BecomeKeyID, 10)},
					{"Repository ID", strconv.FormatInt(inv.RepositoryID, 10)},
					{"Inventory", inv.Inventory},
				}
			})
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryShowCmd)
}
