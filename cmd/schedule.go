package cmd

import (
	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:     "schedule",
	Aliases: []string{"sched"},
	Short:   "Manage schedules",
}

func init() {
	rootCmd.AddCommand(scheduleCmd)
}
