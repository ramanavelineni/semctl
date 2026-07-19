package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ramanavelineni/semctl/internal/config"
)

func TestCompleteUpdateFields(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("current_context: default\ncontexts:\n  default:\n    server: {host: localhost}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.Flags().String("config", cfg, "")
	t.Cleanup(func() { _ = config.Load("") }) // restore global config state

	noNames := func(*cobra.Command) ([]nameID, error) { return nil, errors.New("server unreachable") }
	oneName := func(*cobra.Command) ([]nameID, error) { return []nameID{{ID: 7, Name: "Nightly"}}, nil }

	fn := completeUpdateFields(noNames, true, "name", "cron_format", "active")

	// ID position with an unreachable server: no completions, no file fallback.
	out, directive := fn(cmd, nil, "")
	if len(out) != 0 || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("ID position: got %v (%v)", out, directive)
	}

	// ID position with a reachable server: resource names complete.
	fnNames := completeUpdateFields(oneName, true, "name")
	if out, _ := fnNames(cmd, nil, ""); len(out) != 1 || out[0] != "Nightly\tID 7" {
		t.Errorf("ID position names: got %v", out)
	}

	// Field position: names with '=' and no trailing space.
	out, directive = fn(cmd, []string{"5"}, "")
	if len(out) != 3 || out[0] != "name=" {
		t.Errorf("field position: got %v", out)
	}
	if directive&cobra.ShellCompDirectiveNoSpace == 0 {
		t.Error("field position must set NoSpace so the cursor stays after '='")
	}

	// Boolean value side.
	out, _ = fn(cmd, []string{"5"}, "active=")
	if len(out) != 2 || out[0] != "active=true" || out[1] != "active=false" {
		t.Errorf("bool value: got %v", out)
	}

	// Non-boolean value side: nothing.
	if out, _ := fn(cmd, []string{"5"}, "name="); out != nil {
		t.Errorf("string value: got %v, want nil", out)
	}

	// project update shape: fields AND names from position 0.
	fn0 := completeUpdateFields(oneName, false, "name")
	if out, _ := fn0(cmd, nil, ""); len(out) != 2 || out[0] != "name=" || out[1] != "Nightly\tID 7" {
		t.Errorf("no-ID shape: got %v", out)
	}
}

func TestCompleteContextNames(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("current_context: prod\ncontexts:\n  prod:\n    server: {host: a}\n  staging:\n    server: {host: b}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("config", cfg, "")

	out, directive := completeContextNames(cmd, nil, "")
	if strings.Join(out, ",") != "prod,staging" {
		t.Errorf("contexts: got %v", out)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive: got %v", directive)
	}

	// Second positional (e.g. rename's new name): nothing.
	if out, _ := completeContextNames(cmd, []string{"prod"}, ""); out != nil {
		t.Errorf("second arg: got %v, want nil", out)
	}
	_ = config.Load("") // restore global config state for other tests
}
