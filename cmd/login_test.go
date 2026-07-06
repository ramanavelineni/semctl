package cmd

import (
	"testing"
)

func TestLoginCmd_SchemeDefault(t *testing.T) {
	f := loginCmd.Flags().Lookup("scheme")
	if f == nil {
		t.Fatal("scheme flag not registered")
	}
	if f.DefValue != "http" {
		t.Errorf("scheme default: got %q, want %q", f.DefValue, "http")
	}
}

func TestLoginCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "login" {
			found = true
			break
		}
	}
	if !found {
		t.Error("login command not registered")
	}
}

func TestLoginCmd_Flags(t *testing.T) {
	flags := []string{"server", "scheme", "username", "password", "password-stdin", "save-password", "context"}
	for _, name := range flags {
		f := loginCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered on login command", name)
		}
	}
}

func TestRootCmd_SessionFlags(t *testing.T) {
	flags := []string{"server", "timeout", "insecure", "ca-cert", "config", "context", "project", "yes"}
	for _, name := range flags {
		f := rootCmd.PersistentFlags().Lookup(name)
		if f == nil {
			t.Errorf("persistent flag %q not registered on root command", name)
		}
	}
}
