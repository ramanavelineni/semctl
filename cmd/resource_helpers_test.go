package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseIDArg(t *testing.T) {
	if id, err := parseIDArg("42", "widget"); err != nil || id != 42 {
		t.Errorf("parseIDArg(42) = %d, %v", id, err)
	}
	_, err := parseIDArg("abc", "widget")
	if err == nil || !strings.Contains(err.Error(), "invalid widget ID") {
		t.Errorf("parseIDArg(abc) error = %v, want 'invalid widget ID'", err)
	}
}

func TestRunList_WrapsFetchError(t *testing.T) {
	err := runList("widgets", []string{"ID"},
		func() ([]string, error) { return nil, errors.New("boom") },
		func(s string) []string { return []string{s} })
	if err == nil || !strings.Contains(err.Error(), "failed to list widgets: boom") {
		t.Errorf("error = %v, want wrapped fetch error", err)
	}
}

func TestRunShow_WrapsFetchError(t *testing.T) {
	err := runShow("widget",
		func() (string, error) { return "", errors.New("boom") },
		func(s string) [][]string { return nil })
	if err == nil || !strings.Contains(err.Error(), "failed to get widget: boom") {
		t.Errorf("error = %v, want wrapped fetch error", err)
	}
}

func TestRunDelete_YesSkipsPromptAndWrapsError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("yes", true, "")

	called := false
	if err := runDelete(cmd, "widget", 7, func() error { called = true; return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("delete closure was not called with --yes")
	}

	err := runDelete(cmd, "widget", 7, func() error { return errors.New("boom") })
	if err == nil || !strings.Contains(err.Error(), "failed to delete widget: boom") {
		t.Errorf("error = %v, want wrapped delete error", err)
	}
}
