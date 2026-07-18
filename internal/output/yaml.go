package output

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PrintYAML outputs data as YAML to stdout.
func PrintYAML(data interface{}) error {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding YAML: %w", err)
	}
	return nil
}
