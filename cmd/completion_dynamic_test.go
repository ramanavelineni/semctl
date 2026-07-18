package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ramanavelineni/semctl/internal/config"
)

func TestCompleteUpdateFields(t *testing.T) {
	fn := completeUpdateFields(true, "name", "cron_format", "active")

	// ID position: no completions, no file fallback.
	out, directive := fn(nil, nil, "")
	if out != nil || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("ID position: got %v (%v)", out, directive)
	}

	// Field position: names with '=' and no trailing space.
	out, directive = fn(nil, []string{"5"}, "")
	if len(out) != 3 || out[0] != "name=" {
		t.Errorf("field position: got %v", out)
	}
	if directive&cobra.ShellCompDirectiveNoSpace == 0 {
		t.Error("field position must set NoSpace so the cursor stays after '='")
	}

	// Boolean value side.
	out, _ = fn(nil, []string{"5"}, "active=")
	if len(out) != 2 || out[0] != "active=true" || out[1] != "active=false" {
		t.Errorf("bool value: got %v", out)
	}

	// Non-boolean value side: nothing.
	if out, _ := fn(nil, []string{"5"}, "name="); out != nil {
		t.Errorf("string value: got %v, want nil", out)
	}

	// project update shape: fields from position 0.
	fn0 := completeUpdateFields(false, "name")
	if out, _ := fn0(nil, nil, ""); len(out) != 1 || out[0] != "name=" {
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
