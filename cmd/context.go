package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage server contexts",
	Long:  `Manage multiple Semaphore server contexts. Contexts allow you to switch between different servers without re-entering credentials.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfgFile, _ := cmd.Flags().GetString("config")
		if err := config.Load(cfgFile); err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
			output.SetFormat(output.FormatJSON)
		} else if yamlFlag, _ := cmd.Flags().GetBool("yaml"); yamlFlag {
			output.SetFormat(output.FormatYAML)
		} else {
			output.SetFormat(output.FormatFromConfig())
		}

		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			output.DisableColor()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
}
