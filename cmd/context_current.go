package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/spf13/cobra"
)

var contextCurrentCmd = &cobra.Command{
	Use:     "current",
	Short:   "Show the current context",
	Example: "  semctl context current",
	RunE: func(cmd *cobra.Command, args []string) error {
		contexts := config.ListContexts()
		if len(contexts) == 0 {
			return fmt.Errorf("no contexts configured — run 'semctl login' to create one")
		}
		fmt.Println(config.GetCurrentContext())
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextCurrentCmd)
}
