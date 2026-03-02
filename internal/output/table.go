package output

import (
	"os"

	"github.com/olekukonko/tablewriter"
	tw "github.com/olekukonko/tablewriter/tw"
	"github.com/olekukonko/tablewriter/renderer"
)

// PrintTable outputs data as a formatted table to stdout.
func PrintTable(headers []string, rows [][]string) {
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows: tw.Off,
				},
			},
		})),
	)

	table.Header(toInterfaces(headers)...)
	for _, row := range rows {
		_ = table.Append(toInterfaces(row)...)
	}
	_ = table.Render()
}

func toInterfaces(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
