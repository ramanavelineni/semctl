package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramanavelineni/semctl/internal/apply"
	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export project configuration to YAML or JSON",
	Long: `Export the current state of a Semaphore project as a declarative config file.

The exported file can be edited and re-applied with 'semctl apply'.
Secrets (SSH keys, passwords) are replaced with "<set-me>" placeholders.
Schedules are not exported (no list API).`,
	Example: `  semctl export -p 1 -o project.yaml
  semctl export -p 1 --json
  semctl export -p 1 --only keys,templates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		outputFile, _ := cmd.Flags().GetString("output")
		onlyFlag, _ := cmd.Flags().GetString("only")
		jsonFlag, _ := cmd.Flags().GetBool("json")
		yamlFlag, _ := cmd.Flags().GetBool("yaml")

		// Parse resource filter
		filter, err := apply.ParseResourceFilter(onlyFlag)
		if err != nil {
			return err
		}

		// Authenticate
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		// Get project name
		getParams := project.NewGetProjectProjectIDParams()
		getParams.ProjectID = int64(pid)
		resp, err := apiClient.Project.GetProjectProjectID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}
		projectName := resp.GetPayload().Name

		// Export
		exporter := apply.NewExporter(apiClient, int64(pid), filter)
		cfg, err := exporter.Export(projectName)
		if err != nil {
			return err
		}

		// Determine format
		useJSON := jsonFlag
		if !useJSON && !yamlFlag && outputFile != "" {
			ext := strings.ToLower(filepath.Ext(outputFile))
			useJSON = ext == ".json"
		}

		var data []byte
		if useJSON {
			data, err = apply.MarshalJSON(cfg)
		} else {
			data, err = apply.MarshalYAML(cfg)
		}
		if err != nil {
			return fmt.Errorf("failed to serialize: %w", err)
		}

		// Write output
		if outputFile != "" {
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			style.Success(fmt.Sprintf("Exported to %s", outputFile))
		} else {
			fmt.Print(string(data))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("output", "o", "", "output file path (auto-detects format from extension)")
	exportCmd.Flags().String("only", "", "export only specific resource types (comma-separated: keys,variable_groups,repositories,inventories,templates)")
}
