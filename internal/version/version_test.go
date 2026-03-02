package version

import (
	"testing"
)

func TestDefaultValues(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version: got %q, want %q", Version, "dev")
	}
	if Commit != "none" {
		t.Errorf("Commit: got %q, want %q", Commit, "none")
	}
	if Date != "unknown" {
		t.Errorf("Date: got %q, want %q", Date, "unknown")
	}
}
