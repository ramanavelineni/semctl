package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/spf13/cobra"
)

var contextListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all contexts",
	Example: "  semctl context list",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := config.ListContexts()
		current := config.GetCurrentContext()

		if output.GetFormat() != output.FormatTable {
			var items []map[string]interface{}
			for _, name := range names {
				item := map[string]interface{}{
					"name":    name,
					"current": name == current,
				}
				cc, err := config.GetContextConfig(name)
				if err == nil {
					item["server"] = config.GetContextServerDisplay(name)
					item["username"] = cc.AuthUsername
				}
				items = append(items, item)
			}
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"", "Name", "Server", "Username"}
		var rows [][]string
		for _, name := range names {
			marker := ""
			if name == current {
				marker = "*"
			}
			server := config.GetContextServerDisplay(name)
			username := ""
			if cc, err := config.GetContextConfig(name); err == nil {
				username = cc.AuthUsername
			}
			rows = append(rows, []string{marker, name, server, username})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no contexts configured — run 'semctl login' to create one")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextListCmd)
}
