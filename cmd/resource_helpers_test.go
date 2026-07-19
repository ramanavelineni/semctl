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

func TestResolveIDOrName(t *testing.T) {
	list := func(*cobra.Command) ([]nameID, error) {
		return []nameID{{ID: 1, Name: "Deploy"}, {ID: 2, Name: "backup"}, {ID: 3, Name: "Backup"}}, nil
	}

	// Numeric input never hits the API.
	id, err := resolveIDOrName(nil, "42", "widget", func(*cobra.Command) ([]nameID, error) {
		t.Fatal("list called for numeric arg")
		return nil, nil
	})
	if err != nil || id != 42 {
		t.Errorf("numeric: got %d, %v", id, err)
	}

	// Case-insensitive unique name.
	if id, err := resolveIDOrName(nil, "deploy", "widget", list); err != nil || id != 1 {
		t.Errorf("name: got %d, %v", id, err)
	}

	// Unknown name: not-found error carrying exit code 4.
	_, err = resolveIDOrName(nil, "missing", "widget", list)
	if err == nil || !strings.Contains(err.Error(), `widget "missing" not found`) {
		t.Errorf("not found: got %v", err)
	}
	if code := exitCodeFor(err); code != exitNotFound {
		t.Errorf("not-found exit code = %d, want %d", code, exitNotFound)
	}

	// Duplicate names (case-insensitively) are ambiguous, listing the IDs.
	_, err = resolveIDOrName(nil, "BACKUP", "widget", list)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), "2, 3") {
		t.Errorf("ambiguous: got %v", err)
	}

	// List errors pass through untouched (auth sentinels must survive).
	sentinel := errors.New("boom")
	_, err = resolveIDOrName(nil, "x", "widget", func(*cobra.Command) ([]nameID, error) { return nil, sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("list error: got %v, want wrapped sentinel", err)
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

func TestRunList_NotFoundGetsVersionHint(t *testing.T) {
	err := runList("runners", []string{"ID"},
		func() ([]string, error) { return nil, &codedErr{404} },
		func(s string) []string { return []string{s} })
	if err == nil || !strings.Contains(err.Error(), "may not support this API") {
		t.Errorf("error = %v, want version hint for list 404", err)
	}

	// Other statuses keep the plain wrap.
	err = runList("runners", []string{"ID"},
		func() ([]string, error) { return nil, &codedErr{500} },
		func(s string) []string { return []string{s} })
	if err == nil || strings.Contains(err.Error(), "may not support") {
		t.Errorf("error = %v, want plain wrap for 500", err)
	}
}
