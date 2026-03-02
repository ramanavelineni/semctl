package output

import (
	"github.com/ramanavelineni/semctl/internal/config"
)

// Format represents the output format.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

var currentFormat Format = FormatTable

// SetFormat sets the current output format.
func SetFormat(f Format) {
	currentFormat = f
}

// GetFormat returns the current output format.
func GetFormat() Format {
	return currentFormat
}

// FormatFromConfig returns the format configured in the config file.
func FormatFromConfig() Format {
	switch config.GetOutputFormat() {
	case "json":
		return FormatJSON
	case "yaml":
		return FormatYAML
	default:
		return FormatTable
	}
}

// DisableColor disables color in table output.
func DisableColor() {
	// fatih/color handles this globally
}

// Print outputs data in the current format.
// For table format, headers and rows are used.
// For JSON/YAML, data is serialized directly.
func Print(data interface{}, headers []string, rows [][]string) {
	switch currentFormat {
	case FormatJSON:
		PrintJSON(data)
	case FormatYAML:
		PrintYAML(data)
	default:
		PrintTable(headers, rows)
	}
}
