package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON outputs data as pretty-printed JSON to stdout.
func PrintJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
