package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// newTestCommand creates a minimal cobra command with the interactive flags
// registered, matching the rootCmd persistent flags pattern.
func newTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	cmd.Flags().BoolP("interactive", "I", false, "force interactive mode")
	cmd.Flags().BoolP("no-interactive", "N", false, "disable interactive mode")
	cmd.Flags().Bool("json", false, "output as JSON")
	cmd.Flags().Bool("yaml", false, "output as YAML")
	return cmd
}

func TestShouldAutoInteractive_ConflictingFlags(t *testing.T) {
	cmd := newTestCommand()
	_ = cmd.Flags().Set("interactive", "true")
	_ = cmd.Flags().Set("no-interactive", "true")

	_, err := shouldAutoInteractive(cmd, true)
	if err == nil {
		t.Fatal("expected error for conflicting flags")
	}
}

func TestShouldAutoInteractive_ExplicitInteractiveNoTTY(t *testing.T) {
	cmd := newTestCommand()
	_ = cmd.Flags().Set("interactive", "true")

	// In tests, stdout is not a TTY, so --interactive should error
	_, err := shouldAutoInteractive(cmd, true)
	if err == nil {
		t.Fatal("expected error when --interactive is set without TTY")
	}
}

func TestShouldAutoInteractive_NoInteractive(t *testing.T) {
	cmd := newTestCommand()
	_ = cmd.Flags().Set("no-interactive", "true")

	got, err := shouldAutoInteractive(cmd, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when --no-interactive is set")
	}
}

func TestShouldAutoInteractive_JSONSuppresses(t *testing.T) {
	cmd := newTestCommand()
	_ = cmd.Flags().Set("json", "true")

	got, err := shouldAutoInteractive(cmd, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when --json is set")
	}
}

func TestShouldAutoInteractive_YAMLSuppresses(t *testing.T) {
	cmd := newTestCommand()
	_ = cmd.Flags().Set("yaml", "true")

	got, err := shouldAutoInteractive(cmd, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when --yaml is set")
	}
}

func TestShouldAutoInteractive_NoTTYReturnsFalse(t *testing.T) {
	cmd := newTestCommand()

	// In tests, stdout is not a TTY
	got, err := shouldAutoInteractive(cmd, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when no TTY is available")
	}
}

func TestShouldAutoInteractive_InputsNotMissing(t *testing.T) {
	cmd := newTestCommand()

	got, err := shouldAutoInteractive(cmd, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when inputs are not missing")
	}
}
