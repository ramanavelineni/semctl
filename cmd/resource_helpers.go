package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

// parseIDArg parses a positional resource ID argument.
func parseIDArg(arg, what string) (int64, error) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s ID: %w", what, err)
	}
	return id, nil
}

// nameID is one row of a resource listing, enough to resolve a name to an ID
// and to complete positional arguments (see cmd/resolve.go for the fetchers).
type nameID struct {
	ID   int64
	Name string
}

// resolveIDOrName turns a positional resource argument into an ID: numeric
// input is used as-is, anything else resolves as a case-insensitive name via
// list. what is singular.
func resolveIDOrName(cmd *cobra.Command, arg, what string, list func(*cobra.Command) ([]nameID, error)) (int64, error) {
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
		return id, nil
	}
	items, err := list(cmd)
	if err != nil {
		return 0, err
	}
	return matchNameID(items, arg, what)
}

// matchNameID finds the single item named name (case-insensitive). Duplicate
// names are possible server-side (schedules especially), so multiple matches
// are an error rather than a silent first-wins pick.
func matchNameID(items []nameID, name, what string) (int64, error) {
	var matches []nameID
	for _, it := range items {
		if strings.EqualFold(it.Name, name) {
			matches = append(matches, it)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0].ID, nil
	case 0:
		return 0, withExitCode(fmt.Errorf("%s %q not found", what, name), exitNotFound)
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = strconv.FormatInt(m.ID, 10)
		}
		return 0, fmt.Errorf("%s name %q is ambiguous (IDs %s) — use a numeric ID", what, name, strings.Join(ids, ", "))
	}
}

// runList executes the shared list-command flow: fetch, machine-readable
// branch, empty-collection notice, table. what is plural ("templates").
// The fetch closure owns authentication and the API call so each resource's
// typed params stay in its own file.
func runList[T any](what string, headers []string, fetch func() ([]T, error), row func(T) []string) error {
	items, err := fetch()
	if err != nil {
		// A 404 on a LIST endpoint is never "resource not found" (empty
		// collections return 200 []) — the server lacks the API entirely,
		// which usually means an older Semaphore version.
		if client.IsNotFound(err) {
			return fmt.Errorf("failed to list %s: %w (the server may not support this API — semctl targets Semaphore %s.x, check 'semctl info')", what, err, client.TargetSemaphoreVersion)
		}
		return fmt.Errorf("failed to list %s: %w", what, err)
	}

	if output.GetFormat() != output.FormatTable {
		return output.Print(items, nil, nil)
	}

	if len(items) == 0 {
		printEmptyList(what)
		return nil
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, row(item))
	}
	output.PrintTable(headers, rows)
	return nil
}

// runShow executes the shared show-command flow: fetch one resource, print it
// raw under --json/--yaml or as the Field/Value table from fields otherwise.
// what is singular ("template").
func runShow[T any](what string, fetch func() (T, error), fields func(T) [][]string) error {
	item, err := fetch()
	if err != nil {
		return fmt.Errorf("failed to get %s: %w", what, err)
	}

	if output.GetFormat() != output.FormatTable {
		return output.Print(item, nil, nil)
	}

	output.PrintTable([]string{"Field", "Value"}, fields(item))
	return nil
}

// runDelete executes the shared delete-command flow: confirm, delete, report.
// what is singular ("template").
func runDelete(cmd *cobra.Command, what string, id int64, del func() error) error {
	return runDeleteNamed(cmd, what, strconv.FormatInt(id, 10), del)
}

// runDeleteNamed is runDelete for resources with non-numeric identifiers
// (e.g. API tokens, whose ID is a string).
func runDeleteNamed(cmd *cobra.Command, what, ident string, del func() error) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete %s %s?", what, ident)); err != nil {
		return err
	}
	if err := del(); err != nil {
		return fmt.Errorf("failed to delete %s: %w", what, err)
	}
	style.Success(fmt.Sprintf("Deleted %s %s", what, ident))
	return nil
}
