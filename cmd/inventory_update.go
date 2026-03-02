package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var inventoryUpdateCmd = &cobra.Command{
	Use:   "update <id> [field=value...]",
	Short: "Update an inventory",
	Long:  `Update an inventory. Fields: name, type, inventory, ssh_key_id, become_key_id, repository_id.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl inventory update 1 name="Staging Hosts"
  semctl inventory update 2 type=static-yaml inventory="all:\n  hosts:\n    10.0.0.1:"`,
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

		// Fetch current inventory
		getParams := inventory.NewGetProjectProjectIDInventoryInventoryIDParams()
		getParams.ProjectID = int64(pid)
		getParams.InventoryID = id
		getResp, err := apiClient.Inventory.GetProjectProjectIDInventoryInventoryID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get inventory: %w", err)
		}
		inv := getResp.GetPayload()

		req := &models.InventoryRequest{
			ID:           inv.ID,
			ProjectID:    int64(pid),
			Name:         inv.Name,
			Type:         inv.Type,
			Inventory:    inv.Inventory,
			SSHKeyID:     inv.SSHKeyID,
			BecomeKeyID:  inv.BecomeKeyID,
			RepositoryID: inv.RepositoryID,
		}

		if len(args) < 2 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			switch key {
			case "name":
				req.Name = value
			case "type":
				req.Type = value
			case "inventory":
				req.Inventory = value
			case "ssh_key_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for ssh_key_id: %w", err)
				}
				req.SSHKeyID = n
			case "become_key_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for become_key_id: %w", err)
				}
				req.BecomeKeyID = n
			case "repository_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for repository_id: %w", err)
				}
				req.RepositoryID = n
			default:
				return fmt.Errorf("unknown field %q — valid fields: name, type, inventory, ssh_key_id, become_key_id, repository_id", key)
			}
		}

		putParams := inventory.NewPutProjectProjectIDInventoryInventoryIDParams()
		putParams.ProjectID = int64(pid)
		putParams.InventoryID = id
		putParams.Inventory = req

		_, err = apiClient.Inventory.PutProjectProjectIDInventoryInventoryID(putParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update inventory: %w", err)
		}

		style.Success(fmt.Sprintf("Updated inventory %d", id))
		return nil
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryUpdateCmd)
}
