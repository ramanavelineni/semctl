package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var contextRenameCmd = &cobra.Command{
	Use:     "rename [old-name] [new-name]",
	Short:   "Rename a context",
	Args:    cobra.ExactArgs(2),
	Example: "  semctl context rename default production",
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		if err := config.RenameContext(oldName, newName); err != nil {
			return fmt.Errorf("failed to rename context: %w", err)
		}

		style.Success(fmt.Sprintf("Renamed context %q to %q", oldName, newName))
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextRenameCmd)
}
