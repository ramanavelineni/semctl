package cmd

import (
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:     "key",
	Aliases: []string{"keys"},
	Short:   "Manage access keys",
}

func init() {
	rootCmd.AddCommand(keyCmd)
}
