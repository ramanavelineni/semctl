package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

// errCancelled is returned when the user declines a confirmation prompt.
// It surfaces as a non-zero exit so scripts and CI never mistake a
// cancelled action for a completed one.
var errCancelled = fmt.Errorf("cancelled")

// confirmAction asks the user to confirm a destructive or mutating action.
//   - --yes given → proceed
//   - stdin is not a terminal → error telling the caller to pass --yes
//   - otherwise prompt on stderr; anything but y/yes → errCancelled
func confirmAction(cmd *cobra.Command, prompt string) error {
	if yes, _ := cmd.Flags().GetBool("yes"); yes {
		return nil
	}

	if !style.IsStdinTTY() {
		return fmt.Errorf("confirmation required for %q: re-run with --yes in non-interactive mode", cmd.CommandPath())
	}

	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	// Read on a goroutine so Ctrl-C (which cancels the command context but
	// cannot unblock a stdin read) aborts the prompt immediately. The reader
	// goroutine leaks on cancellation; the process is about to exit anyway.
	lines := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		lines <- response
	}()

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	var response string
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr)
		return errCancelled
	case response = <-lines:
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return errCancelled
	}
	return nil
}
