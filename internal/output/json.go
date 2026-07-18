package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON outputs data as pretty-printed JSON to stdout.
func PrintJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}
