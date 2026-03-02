package cmd

import (
	"github.com/spf13/cobra"
)

var inventoryCmd = &cobra.Command{
	Use:     "inventory",
	Aliases: []string{"inv"},
	Short:   "Manage inventories",
}

func init() {
	rootCmd.AddCommand(inventoryCmd)
}
