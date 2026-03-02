package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextList_NoContexts(t *testing.T) {
	cfgPath := writeTestConfig(t, `current_context: default`)
	err := executeCommand("context", "list", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error when no contexts configured")
	}
}

func TestContextList_WithContexts(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
      port: 3000
    auth:
      username: admin
`)
	err := executeCommand("context", "list", "--config", cfgPath, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextList_AliasLs(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
      port: 3000
    auth:
      username: admin
`)
	err := executeCommand("context", "ls", "--config", cfgPath, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextCurrent_NoContexts(t *testing.T) {
	cfgPath := writeTestConfig(t, `current_context: default`)
	err := executeCommand("context", "current", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error when no contexts configured")
	}
}

func TestContextCurrent_WithConfig(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: myctx
contexts:
  myctx:
    server:
      host: localhost
`)
	err := executeCommand("context", "current", "--config", cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextUse_Exists(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
  staging:
    server:
      host: staging.example.com
`)
	err := executeCommand("context", "use", "staging", "--config", cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextUse_NotFound(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	err := executeCommand("context", "use", "nonexistent", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error for nonexistent context")
	}
}

func TestContextUse_NoArgs(t *testing.T) {
	cfgPath := writeTestConfig(t, `current_context: default`)
	err := executeCommand("context", "use", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestContextDelete_NoArgs(t *testing.T) {
	cfgPath := writeTestConfig(t, `current_context: default`)
	err := executeCommand("context", "delete", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestContextDelete_WithYes(t *testing.T) {
	cfgPath := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
  staging:
    server:
      host: staging.example.com
`)
	err := executeCommand("context", "delete", "staging", "--yes", "--config", cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify context was deleted
	raw, _ := os.ReadFile(cfgPath)
	if strings.Contains(string(raw), "staging") {
		t.Error("staging context should have been deleted from config")
	}
}

func TestContextRename_NoArgs(t *testing.T) {
	cfgPath := writeTestConfig(t, `current_context: default`)
	err := executeCommand("context", "rename", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestContextRename_OneArg(t *testing.T) {
	cfgPath := writeTestConfig(t, `current_context: default`)
	err := executeCommand("context", "rename", "old", "--config", cfgPath)
	if err == nil {
		t.Fatal("expected error for missing second arg")
	}
}

func TestContextRename_Success(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `current_context: default
contexts:
  default:
    server:
      host: localhost
      port: 3000
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	err := executeCommand("context", "rename", "default", "production", "--config", cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(raw), "production") {
		t.Error("production context should exist after rename")
	}
}

func TestContextCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "context" {
			found = true
			break
		}
	}
	if !found {
		t.Error("context command not registered")
	}
}

func TestContextCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "current", "use", "delete", "rename"}
	for _, name := range expected {
		found := false
		for _, cmd := range contextCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("context subcommand %q not registered", name)
		}
	}
}
