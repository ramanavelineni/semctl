package cmd

import (
	"testing"
)

func TestCompletionCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "completion" {
			found = true
			break
		}
	}
	if !found {
		t.Error("completion command not registered")
	}
}

func TestCompletionCmd_Bash(t *testing.T) {
	err := executeCommand("completion", "bash")
	if err != nil {
		t.Fatalf("completion bash should not error: %v", err)
	}
}

func TestCompletionCmd_Zsh(t *testing.T) {
	err := executeCommand("completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh should not error: %v", err)
	}
}

func TestCompletionCmd_Fish(t *testing.T) {
	err := executeCommand("completion", "fish")
	if err != nil {
		t.Fatalf("completion fish should not error: %v", err)
	}
}

func TestCompletionCmd_Powershell(t *testing.T) {
	err := executeCommand("completion", "powershell")
	if err != nil {
		t.Fatalf("completion powershell should not error: %v", err)
	}
}

func TestCompletionCmd_InvalidShell(t *testing.T) {
	err := executeCommand("completion", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid shell")
	}
}

func TestCompletionCmd_NoArgs(t *testing.T) {
	err := executeCommand("completion")
	if err == nil {
		t.Fatal("expected error for missing shell argument")
	}
}
