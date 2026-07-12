package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

// optionalInt is a huh validator accepting empty input or an integer.
func optionalInt(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		if _, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err != nil {
			return fmt.Errorf("%s must be a number", field)
		}
		return nil
	}
}

// parseOptionalInt parses an optionalInt-validated input ("" → 0).
func parseOptionalInt(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// newForm creates a huh.Form with Esc added to the quit key binding,
// allowing users to press Escape to abort any interactive form.
func newForm(groups ...*huh.Group) *huh.Form {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	return huh.NewForm(groups...).WithKeyMap(km)
}

// runForm runs a form, translating huh's abort error into errCancelled so
// aborting a form and declining a [y/N] prompt exit identically.
func runForm(f *huh.Form) error {
	if err := f.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errCancelled
		}
		return err
	}
	return nil
}

// shouldAutoInteractive decides whether a command should launch interactive mode.
// Decision order:
//  1. --interactive + --no-interactive → error
//  2. --interactive explicitly set → require TTY, return true
//  3. --no-interactive explicitly set → return false
//  4. --json or --yaml flag set → return false (scripting context)
//  5. inputsMissing && IsTTY() → return true (auto-interactive)
//  6. Otherwise → return false
func shouldAutoInteractive(cmd *cobra.Command, inputsMissing bool) (bool, error) {
	iFlag := cmd.Flags().Lookup("interactive")
	nFlag := cmd.Flags().Lookup("no-interactive")

	interactiveSet := iFlag != nil && iFlag.Changed
	noInteractiveSet := nFlag != nil && nFlag.Changed

	if interactiveSet && noInteractiveSet {
		return false, fmt.Errorf("cannot use --interactive and --no-interactive together")
	}

	if interactiveSet {
		if !style.IsTTY() || !style.IsStdinTTY() {
			return false, fmt.Errorf("--interactive requires a terminal")
		}
		return true, nil
	}

	if noInteractiveSet {
		return false, nil
	}

	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		return false, nil
	}
	if yamlFlag, _ := cmd.Flags().GetBool("yaml"); yamlFlag {
		return false, nil
	}

	// Both streams must be terminals: forms read from stdin, so a piped or
	// closed stdin would make huh die on EOF instead of the flag error.
	if inputsMissing && style.IsTTY() && style.IsStdinTTY() {
		return true, nil
	}

	return false, nil
}
