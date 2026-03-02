package cmd

import (
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:     "template",
	Aliases: []string{"tpl"},
	Short:   "Manage templates",
}

func init() {
	rootCmd.AddCommand(templateCmd)
}
