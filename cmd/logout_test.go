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

// logout must rely on the root persistent --context flag; a local one would
// shadow it (see the login -s shorthand bug).
func TestLogoutCmd_NoLocalContextFlag(t *testing.T) {
	if logoutCmd.LocalFlags().Lookup("context") != nil {
		t.Error("logout defines a local context flag that shadows the root persistent flag")
	}
	if rootCmd.PersistentFlags().Lookup("context") == nil {
		t.Error("root persistent context flag missing")
	}
}

func TestLogoutCmd_Help(t *testing.T) {
	err := executeCommand("logout", "--help")
	if err != nil {
		t.Fatalf("logout --help should not error: %v", err)
	}
}
