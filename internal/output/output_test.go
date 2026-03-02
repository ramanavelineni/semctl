package output

import (
	"testing"
)

func TestSetFormat_GetFormat_RoundTrip(t *testing.T) {
	tests := []Format{FormatTable, FormatJSON, FormatYAML}
	for _, f := range tests {
		SetFormat(f)
		if got := GetFormat(); got != f {
			t.Errorf("SetFormat(%q) then GetFormat() = %q", f, got)
		}
	}
	// Reset
	SetFormat(FormatTable)
}

func TestFormatConstants(t *testing.T) {
	if FormatTable != "table" {
		t.Errorf("FormatTable: got %q, want %q", FormatTable, "table")
	}
	if FormatJSON != "json" {
		t.Errorf("FormatJSON: got %q, want %q", FormatJSON, "json")
	}
	if FormatYAML != "yaml" {
		t.Errorf("FormatYAML: got %q, want %q", FormatYAML, "yaml")
	}
}

func TestDefaultFormat(t *testing.T) {
	// After package init, default should be table
	SetFormat(FormatTable)
	if got := GetFormat(); got != FormatTable {
		t.Errorf("default format: got %q, want %q", got, FormatTable)
	}
}

func TestDisableColor_DoesNotPanic(t *testing.T) {
	// DisableColor is a no-op stub; just ensure it doesn't panic
	DisableColor()
}
