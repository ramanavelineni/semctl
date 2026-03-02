package cmd

import (
	"testing"
)

func TestVersionCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("version command not registered")
	}
}

func TestVersionCmd_Runs(t *testing.T) {
	err := executeCommand("version")
	if err != nil {
		t.Fatalf("version command should not error: %v", err)
	}
}
