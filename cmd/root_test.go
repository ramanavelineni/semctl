package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// executeCommand runs the root command with the given arguments and returns any error.
func executeCommand(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// resetFlags resets all flags on the command (and its parents) so each test starts clean.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	if cmd.HasParent() {
		resetFlags(cmd.Parent())
	}
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
		{"json", "", "false"},
		{"yaml", "", "false"},
		{"yes", "y", "false"},
		{"no-color", "", "false"},
		{"server", "s", ""},
		{"interactive", "I", "false"},
		{"no-interactive", "N", "false"},
		{"context", "", ""},
		{"project", "p", "0"},
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
	cmd.Flags().Int32P("project", "p", 0, "project ID")
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
	cmd.Flags().Int32P("project", "p", 0, "project ID")

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
	cmd.Flags().Int32P("project", "p", 0, "project ID")

	_, err := getProjectID(cmd)
	if err == nil {
		t.Fatal("expected error when project ID is missing")
	}
}
