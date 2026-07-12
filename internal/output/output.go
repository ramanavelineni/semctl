package output

import (
	"reflect"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
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

// DisableColor disables colored and emoji output globally.
func DisableColor() {
	style.DisableColor()
	style.SetEmojiEnabled(false)
}

// Print outputs data in the current format.
// For table format, headers and rows are used.
// For JSON/YAML, data is serialized directly.
func Print(data interface{}, headers []string, rows [][]string) {
	switch currentFormat {
	case FormatJSON:
		PrintJSON(normalizeNilSlice(data))
	case FormatYAML:
		PrintYAML(normalizeNilSlice(data))
	default:
		PrintTable(headers, rows)
	}
}

// normalizeNilSlice turns a nil slice into an empty one so empty lists
// serialize as [] rather than null.
func normalizeNilSlice(data interface{}) interface{} {
	if v := reflect.ValueOf(data); v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return data
}
