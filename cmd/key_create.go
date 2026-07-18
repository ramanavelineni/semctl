package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var keyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an access key",
	Example: `  semctl key create --name "None Key" --type none
  semctl key create --name "SSH Key" --type ssh --private-key-file ~/.ssh/id_deploy
  echo "$PASS" | semctl key create --name "Login" --type login_password --login admin --password-stdin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		keyType, _ := cmd.Flags().GetString("type")
		login, _ := cmd.Flags().GetString("login")
		privateKey, _ := cmd.Flags().GetString("private-key")
		privateKeyFile, _ := cmd.Flags().GetString("private-key-file")
		passphrase, _ := cmd.Flags().GetString("passphrase")
		password, _ := cmd.Flags().GetString("password")
		passwordStdin, _ := cmd.Flags().GetBool("password-stdin")

		// File/stdin variants keep secrets out of argv, ps, and shell history.
		if privateKey != "" && privateKeyFile != "" {
			return fmt.Errorf("cannot use --private-key and --private-key-file together")
		}
		if privateKeyFile != "" {
			data, err := os.ReadFile(privateKeyFile)
			if err != nil {
				return fmt.Errorf("reading private key file: %w", err)
			}
			privateKey = string(data)
		}
		if password != "" && passwordStdin {
			return fmt.Errorf("cannot use --password and --password-stdin together")
		}
		if passwordStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			password = strings.TrimRight(string(data), "\r\n")
		}

		interactive, err := shouldAutoInteractive(cmd, name == "" || keyType == "")
		if err != nil {
			return err
		}
		if interactive {
			if err := runForm(newForm(
				huh.NewGroup(
					huh.NewInput().Title("Key name").Value(&name).
						Validate(requireValue("name")),
					huh.NewSelect[string]().Title("Type").
						Options(
							huh.NewOption("none", "none"),
							huh.NewOption("ssh", "ssh"),
							huh.NewOption("login_password", "login_password"),
						).
						Value(&keyType),
				).Title("New access key"),
			)); err != nil {
				return err
			}

			// Type-specific fields in a second form, once the type is known
			switch keyType {
			case "ssh":
				if err := runForm(newForm(
					huh.NewGroup(
						huh.NewInput().Title("Login (optional)").Value(&login),
						huh.NewText().Title("Private key").Value(&privateKey).
							Validate(requireValue("private key")),
						huh.NewInput().Title("Passphrase (optional)").EchoMode(huh.EchoModePassword).Value(&passphrase),
					).Title("SSH key"),
				)); err != nil {
					return err
				}
			case "login_password":
				if err := runForm(newForm(
					huh.NewGroup(
						huh.NewInput().Title("Login").Value(&login).
							Validate(requireValue("login")),
						huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&password).
							Validate(requireValue("password")),
					).Title("Login/password credentials"),
				)); err != nil {
					return err
				}
			}
		}

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
		// Machine-readable resource on stdout so pipelines can capture the ID.
		if output.GetFormat() != output.FormatTable {
			return output.Print(k, nil, nil)
		}
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyCreateCmd)

	keyCreateCmd.Flags().String("name", "", "key name (required)")
	keyCreateCmd.Flags().String("type", "", "key type: none, ssh, login_password (required)")
	keyCreateCmd.Flags().String("login", "", "login username (for ssh/login_password)")
	keyCreateCmd.Flags().String("private-key", "", "SSH private key content (prefer --private-key-file)")
	keyCreateCmd.Flags().String("private-key-file", "", "path to an SSH private key file")
	keyCreateCmd.Flags().String("passphrase", "", "SSH key passphrase")
	keyCreateCmd.Flags().String("password", "", "password (for login_password type; prefer --password-stdin)")
	keyCreateCmd.Flags().Bool("password-stdin", false, "read the password from stdin")
}
