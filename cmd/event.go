package cmd

import (
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "View Semaphore events",
}

func init() {
	rootCmd.AddCommand(eventCmd)
}
