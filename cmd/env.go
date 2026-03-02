package cmd

import (
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:     "env",
	Aliases: []string{"environment"},
	Short:   "Manage environments",
}

func init() {
	rootCmd.AddCommand(envCmd)
}
