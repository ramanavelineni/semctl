package cmd

import (
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"repository"},
	Short:   "Manage repositories",
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
