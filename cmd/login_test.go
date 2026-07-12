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
	flags := []string{"scheme", "username", "password", "password-stdin", "save-password"}
	for _, name := range flags {
		f := loginCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered on login command", name)
		}
	}
}

// Local --server/--context flags would shadow the root persistent ones and
// break the -s shorthand (pflag drops the shorthand of a shadowed flag).
func TestLoginCmd_NoShadowedPersistentFlags(t *testing.T) {
	for _, name := range []string{"server", "context"} {
		if loginCmd.LocalFlags().Lookup(name) != nil && rootCmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("login defines a local %q flag that shadows the root persistent flag", name)
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
