package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var keyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an access key",
	Example: `  semctl key create --name "None Key" --type none
  semctl key create --name "SSH Key" --type ssh --private-key "$(cat ~/.ssh/id_rsa)"
  semctl key create --name "Login" --type login_password --login admin --password secret`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		keyType, _ := cmd.Flags().GetString("type")
		login, _ := cmd.Flags().GetString("login")
		privateKey, _ := cmd.Flags().GetString("private-key")
		passphrase, _ := cmd.Flags().GetString("passphrase")
		password, _ := cmd.Flags().GetString("password")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if keyType == "" {
			return fmt.Errorf("--type is required (none, ssh, login_password)")
		}

		req := &models.AccessKeyRequest{
			ProjectID: int64(pid),
			Name:      name,
			Type:      keyType,
		}

		switch keyType {
		case "ssh":
			req.SSH = &models.AccessKeyRequestSSH{
				Login:      login,
				PrivateKey: privateKey,
				Passphrase: passphrase,
			}
		case "login_password":
			req.LoginPassword = &models.AccessKeyRequestLoginPassword{
				Login:    login,
				Password: password,
			}
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := key_store.NewPostProjectProjectIDKeysParams()
		params.ProjectID = int64(pid)
		params.AccessKey = req

		resp, err := apiClient.KeyStore.PostProjectProjectIDKeys(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create key: %w", err)
		}

		k := resp.GetPayload()
		style.Success(fmt.Sprintf("Created key %q (ID: %d)", k.Name, k.ID))
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyCreateCmd)

	keyCreateCmd.Flags().String("name", "", "key name (required)")
	keyCreateCmd.Flags().String("type", "", "key type: none, ssh, login_password (required)")
	keyCreateCmd.Flags().String("login", "", "login username (for ssh/login_password)")
	keyCreateCmd.Flags().String("private-key", "", "SSH private key content")
	keyCreateCmd.Flags().String("passphrase", "", "SSH key passphrase")
	keyCreateCmd.Flags().String("password", "", "password (for login_password type)")
}
