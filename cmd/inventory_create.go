package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var inventoryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an inventory",
	Example: `  semctl inventory create --name "Production" --type static --inventory "[all]\n10.0.0.1" --ssh-key-id 1
  semctl inventory create --name "Hosts" --type file --ssh-key-id 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		invType, _ := cmd.Flags().GetString("type")
		invContent, _ := cmd.Flags().GetString("inventory")
		sshKeyID, _ := cmd.Flags().GetInt64("ssh-key-id")
		becomeKeyID, _ := cmd.Flags().GetInt64("become-key-id")
		repoID, _ := cmd.Flags().GetInt64("repository-id")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if invType == "" {
			return fmt.Errorf("--type is required (static, static-yaml, file, terraform-workspace)")
		}

		req := &models.InventoryRequest{
			ProjectID:    int64(pid),
			Name:         name,
			Type:         invType,
			Inventory:    invContent,
			SSHKeyID:     sshKeyID,
			BecomeKeyID:  becomeKeyID,
			RepositoryID: repoID,
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := inventory.NewPostProjectProjectIDInventoryParams()
		params.ProjectID = int64(pid)
		params.Inventory = req

		resp, err := apiClient.Inventory.PostProjectProjectIDInventory(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create inventory: %w", err)
		}

		inv := resp.GetPayload()
		style.Success(fmt.Sprintf("Created inventory %q (ID: %d)", inv.Name, inv.ID))
		return nil
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryCreateCmd)

	inventoryCreateCmd.Flags().String("name", "", "inventory name (required)")
	inventoryCreateCmd.Flags().String("type", "", "inventory type: static, static-yaml, file, terraform-workspace (required)")
	inventoryCreateCmd.Flags().String("inventory", "", "inventory content")
	inventoryCreateCmd.Flags().Int64("ssh-key-id", 0, "SSH key ID")
	inventoryCreateCmd.Flags().Int64("become-key-id", 0, "become key ID")
	inventoryCreateCmd.Flags().Int64("repository-id", 0, "repository ID")
}
