package cmd

import (
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users and per-user options",
}

func init() {
	rootCmd.AddCommand(userCmd)
}
