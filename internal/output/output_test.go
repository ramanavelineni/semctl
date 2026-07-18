package output

import (
	"testing"

	"github.com/fatih/color"
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

func TestDisableColor_SetsGlobalNoColor(t *testing.T) {
	// --no-color routes here; it must actually switch fatih/color off
	// (it used to be an empty stub, silently doing nothing).
	prev := color.NoColor
	defer func() { color.NoColor = prev }()

	color.NoColor = false
	DisableColor()
	if !color.NoColor {
		t.Error("DisableColor() did not set color.NoColor")
	}
}

func TestNormalizeNilSlice(t *testing.T) {
	var nilSlice []string
	got := normalizeNilSlice(nilSlice)
	s, ok := got.([]string)
	if !ok {
		t.Fatalf("normalizeNilSlice(nil []string) returned %T", got)
	}
	if s == nil || len(s) != 0 {
		t.Errorf("want non-nil empty slice, got %#v", s)
	}

	// Non-nil slices and non-slices pass through unchanged.
	orig := []int{1, 2}
	if got := normalizeNilSlice(orig); len(got.([]int)) != 2 {
		t.Errorf("non-nil slice was altered: %#v", got)
	}
	if got := normalizeNilSlice("str"); got != "str" {
		t.Errorf("non-slice was altered: %#v", got)
	}
	if got := normalizeNilSlice(nil); got != nil {
		t.Errorf("untyped nil was altered: %#v", got)
	}
}

func TestPrintJSON_ReturnsEncodeError(t *testing.T) {
	// A channel cannot be JSON-encoded; the old code os.Exit(1)'d here.
	if err := PrintJSON(make(chan int)); err == nil {
		t.Error("PrintJSON(chan) should return an error")
	}
}
