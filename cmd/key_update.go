package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var keyUpdateCmd = &cobra.Command{
	Use:   "update <id> [field=value...]",
	Short: "Update an access key",
	Long: `Update an access key. Fields: name, type, login, password, private_key, passphrase.

The API replaces the stored secret as a whole: updating any of login, password,
private_key, or passphrase requires the key's secret field to be included too
(private_key for ssh keys, password for login_password keys). Updating only
name or type leaves the stored secret untouched.`,
	Args: cobra.MinimumNArgs(1),
	Example: `  semctl key update 1 name="Renamed Key"
  semctl key update 2 login=newuser password=newpass`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		// No GET-by-ID for keys; find from list
		listParams := key_store.NewGetProjectProjectIDKeysParams()
		listParams.ProjectID = int64(pid)
		listResp, err := apiClient.KeyStore.GetProjectProjectIDKeys(listParams, nil)
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		var found *models.AccessKey
		for _, k := range listResp.GetPayload() {
			if k.ID == id {
				found = k
				break
			}
		}
		if found == nil {
			return fmt.Errorf("key %d not found", id)
		}

		req := &models.AccessKeyRequest{
			ID:        found.ID,
			ProjectID: int64(pid),
			Name:      found.Name,
			Type:      found.Type,
		}

		if len(args) < 2 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		// Track sub-struct fields
		var login, password, privateKey, passphrase string
		hasSecretFields := false

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			key = strings.ReplaceAll(key, "-", "_") // accept kebab-case like the create flags
			switch key {
			case "name":
				req.Name = value
			case "type":
				req.Type = value
			case "login":
				login = value
				hasSecretFields = true
			case "password":
				password = value
				hasSecretFields = true
			case "private_key":
				privateKey = value
				hasSecretFields = true
			case "passphrase":
				passphrase = value
				hasSecretFields = true
			default:
				return fmt.Errorf("unknown field %q — valid fields: name, type, login, password, private_key, passphrase", key)
			}
		}

		// The API cannot partially update secrets: with OverrideSecret set it
		// replaces the whole sub-struct, so a lone login= would silently wipe
		// the stored private key. Only override when the full secret is given;
		// otherwise (name/type-only updates) the server keeps the old secret.
		if hasSecretFields {
			switch req.Type {
			case "ssh":
				if privateKey == "" {
					return fmt.Errorf("updating ssh key fields replaces the stored secret: include private_key= alongside login=/passphrase=")
				}
				req.SSH = &models.AccessKeyRequestSSH{
					Login:      login,
					PrivateKey: privateKey,
					Passphrase: passphrase,
				}
			case "login_password":
				if password == "" {
					return fmt.Errorf("updating login_password key fields replaces the stored secret: include password= alongside login=")
				}
				req.LoginPassword = &models.AccessKeyRequestLoginPassword{
					Login:    login,
					Password: password,
				}
			default:
				return fmt.Errorf("key %d has type %q, which has no login/password/private_key/passphrase fields", id, req.Type)
			}
			req.OverrideSecret = true
		}

		putParams := key_store.NewPutProjectProjectIDKeysKeyIDParams()
		putParams.ProjectID = int64(pid)
		putParams.KeyID = id
		putParams.AccessKey = req

		_, err = apiClient.KeyStore.PutProjectProjectIDKeysKeyID(putParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update key: %w", err)
		}

		style.Success(fmt.Sprintf("Updated key %d", id))
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyUpdateCmd)
}
