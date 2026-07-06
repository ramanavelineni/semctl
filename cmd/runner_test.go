package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRunnerCmd_Subcommands(t *testing.T) {
	want := []string{"list", "show", "create", "update", "delete", "activate", "deactivate", "clear-cache", "token", "tags"}
	got := map[string]bool{}
	for _, c := range runnerCmd.Commands() {
		got[c.Name()] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("runner subcommand %q not registered", name)
		}
	}
}

func TestRunnerScope_GlobalByDefault(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project ID or name")

	pid, scoped, err := runnerScope(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scoped || pid != 0 {
		t.Errorf("expected global scope by default, got scoped=%v pid=%d", scoped, pid)
	}
}

func TestRunnerScope_ExplicitProject(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("project", "p", "", "project ID or name")
	_ = cmd.Flags().Set("project", "42")

	pid, scoped, err := runnerScope(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !scoped || pid != 42 {
		t.Errorf("expected project scope with pid=42, got scoped=%v pid=%d", scoped, pid)
	}
}
