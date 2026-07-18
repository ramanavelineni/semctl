package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var userOptionsCmd = &cobra.Command{
	Use:   "options [key]",
	Short: "Show the current user's stored options",
	Args:  cobra.MaximumNArgs(1),
	Example: `  semctl user options
  semctl user options nav.unpinnedItems`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		resp, err := apiClient.User.GetUserOptions(user.NewGetUserOptionsParams(), nil)
		if err != nil {
			return fmt.Errorf("failed to get user options: %w", err)
		}

		options := resp.GetPayload()

		if len(args) == 1 {
			value, ok := options[args[0]]
			if !ok {
				return fmt.Errorf("option %q not set", args[0])
			}
			fmt.Println(value)
			return nil
		}

		if output.GetFormat() != output.FormatTable {
			return output.Print(options, nil, nil)
		}

		if len(options) == 0 {
			style.Info("No options set.")
			return nil
		}

		keys := make([]string, 0, len(options))
		for k := range options {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		headers := []string{"Key", "Value"}
		var rows [][]string
		for _, k := range keys {
			rows = append(rows, []string{k, options[k]})
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

var userOptionsSetCmd = &cobra.Command{
	Use:     "set <key=value>",
	Short:   "Set a user option",
	Args:    cobra.ExactArgs(1),
	Example: `  semctl user options set nav.unpinnedItems='["dashboard"]'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value, ok := strings.Cut(args[0], "=")
		if !ok || key == "" {
			return fmt.Errorf("invalid argument %q — expected key=value", args[0])
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := user.NewPostUserOptionsParams()
		params.Body = &models.Option{Key: key, Value: value}

		if _, err := apiClient.User.PostUserOptions(params, nil); err != nil {
			return fmt.Errorf("failed to set user option (the server only accepts known option keys, e.g. nav.unpinnedItems): %w", err)
		}

		style.Success(fmt.Sprintf("Set option %q", key))
		return nil
	},
}

func init() {
	userCmd.AddCommand(userOptionsCmd)
	userOptionsCmd.AddCommand(userOptionsSetCmd)
}
