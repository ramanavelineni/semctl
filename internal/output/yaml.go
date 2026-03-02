package output

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PrintYAML outputs data as YAML to stdout.
func PrintYAML(data interface{}) {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding YAML: %v\n", err)
		os.Exit(1)
	}
}
