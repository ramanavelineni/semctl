package cmd

import (
	"testing"
)

func TestLogoutCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "logout" {
			found = true
			break
		}
	}
	if !found {
		t.Error("logout command not registered")
	}
}

func TestLogoutCmd_ContextFlag(t *testing.T) {
	f := logoutCmd.Flags().Lookup("context")
	if f == nil {
		t.Fatal("context flag not registered on logout command")
	}
	if f.DefValue != "" {
		t.Errorf("context default: got %q, want empty", f.DefValue)
	}
}

func TestLogoutCmd_Help(t *testing.T) {
	err := executeCommand("logout", "--help")
	if err != nil {
		t.Fatalf("logout --help should not error: %v", err)
	}
}
