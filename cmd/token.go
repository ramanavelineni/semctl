package cmd

import (
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage API tokens for the logged-in user",
}

func init() {
	rootCmd.AddCommand(tokenCmd)
}
