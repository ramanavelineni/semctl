package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/spf13/cobra"
)

// executeCommand runs the root command with the given arguments and returns any error.
func executeCommand(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// writeTestConfig writes YAML content to a temp dir and returns the file path.
func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

func TestRootCmd_SilenceUsage(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Error("expected SilenceUsage to be true")
	}
}

func TestRootCmd_PersistentFlags(t *testing.T) {
	flags := []struct {
		name     string
		short    string
		defValue string
	}{
		{"config", "c", ""},
		{"output", "", ""},
		{"json", "", "false"},
		{"yaml", "", "false"},
		{"yes", "y", "false"},
		{"no-color", "", "false"},
		{"server", "s", ""},
		{"interactive", "I", "false"},
		{"no-interactive", "N", "false"},
		{"context", "", ""},
		{"project", "p", ""},
	}

	for _, tc := range flags {
		t.Run(tc.name, func(t *testing.T) {
			f := rootCmd.PersistentFlags().Lookup(tc.name)
			if f == nil {
				t.Fatalf("flag %q not registered", tc.name)
			}
			if tc.short != "" && f.Shorthand != tc.short {
				t.Errorf("flag %q shorthand: got %q, want %q", tc.name, f.Shorthand, tc.short)
			}
			if f.DefValue != tc.defValue {
				t.Errorf("flag %q default: got %q, want %q", tc.name, f.DefValue, tc.defValue)
			}
		})
	}
}

func TestRootCmd_PersistentPreRunE_SkipsLogin(t *testing.T) {
	err := executeCommand("login", "--help")
	if err != nil {
		t.Errorf("login --help should not error: %v", err)
	}
}

func TestRootCmd_PersistentPreRunE_SkipsVersion(t *testing.T) {
	err := executeCommand("version")
	if err != nil {
		t.Errorf("version should not error: %v", err)
	}
}

func TestRootCmd_PersistentPreRunE_SkipsCompletion(t *testing.T) {
	err := executeCommand("completion", "bash")
	if err != nil {
		t.Errorf("completion bash should not error: %v", err)
	}
}

func TestGetProjectID_FromFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project ID or name")
	_ = cmd.Flags().Set("project", "42")

	id, err := getProjectID(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("got %d, want 42", id)
	}
}

func TestGetProjectID_FromConfig(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
      port: 3000
defaults:
  project_id: 7
`)
	if err := config.Load(cfgPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project ID or name")

	id, err := getProjectID(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 7 {
		t.Errorf("got %d, want 7", id)
	}
}

func TestGetProjectID_MissingReturnsError(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	if err := config.Load(cfgPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project ID or name")

	_, err := getProjectID(cmd)
	if err == nil {
		t.Fatal("expected error when project ID is missing")
	}
}

func TestGetProjectID_InvalidNumeric(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project ID or name")
	_ = cmd.Flags().Set("project", "-3")

	_, err := getProjectID(cmd)
	if err == nil {
		t.Fatal("expected error for negative project ID")
	}
}

// formatTestCmd builds a root+child pair with the output-format flags and
// parses args on the child, mirroring how cobra merges persistent flags.
func formatTestCmd(t *testing.T, localOutputFlag bool, args ...string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("output", "", "")
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	child := &cobra.Command{Use: "child"}
	if localOutputFlag {
		child.Flags().String("output", "", "") // like export's file-path flag
	}
	root.AddCommand(child)
	if err := child.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags(%v): %v", args, err)
	}
	return child
}

func TestResolveOutputFormat(t *testing.T) {
	t.Cleanup(func() { output.SetFormat(output.FormatTable) })

	cases := []struct {
		name    string
		args    []string
		want    output.Format
		wantErr string
	}{
		{"json flag", []string{"--json"}, output.FormatJSON, ""},
		{"yaml flag", []string{"--yaml"}, output.FormatYAML, ""},
		{"output json", []string{"--output", "json"}, output.FormatJSON, ""},
		{"output yaml uppercase", []string{"--output", "YAML"}, output.FormatYAML, ""},
		{"output table", []string{"--output", "table"}, output.FormatTable, ""},
		{"agreeing flags", []string{"--output", "json", "--json"}, output.FormatJSON, ""},
		{"json vs yaml", []string{"--json", "--yaml"}, "", "conflicting output formats"},
		{"output vs json", []string{"--output", "yaml", "--json"}, "", "conflicting output formats"},
		{"invalid value", []string{"--output", "xml"}, "", "invalid output format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := formatTestCmd(t, false, tc.args...)
			err := resolveOutputFormat(cmd)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := output.GetFormat(); got != tc.want {
				t.Errorf("format = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveOutputFormat_LocalOutputFlagShadowsFormat(t *testing.T) {
	t.Cleanup(func() { output.SetFormat(output.FormatTable) })

	// export-style local --output (a file path) must not be read as a format...
	cmd := formatTestCmd(t, true, "--output", "backup.yaml")
	if err := resolveOutputFormat(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ...and --json alongside it still selects JSON without a conflict.
	cmd = formatTestCmd(t, true, "--output", "backup.yaml", "--json")
	if err := resolveOutputFormat(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.GetFormat() != output.FormatJSON {
		t.Errorf("format = %v, want json", output.GetFormat())
	}
}
