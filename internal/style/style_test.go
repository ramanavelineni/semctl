package style

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// captureStderr redirects os.Stderr to capture output written by style functions.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestSuccess_WritesToStderr(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	output := captureStderr(t, func() {
		Success("it worked")
	})
	if !strings.Contains(output, "it worked") {
		t.Errorf("expected stderr to contain %q, got %q", "it worked", output)
	}
}

func TestError_WritesToStderr(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	output := captureStderr(t, func() {
		Error("something broke")
	})
	if !strings.Contains(output, "something broke") {
		t.Errorf("expected stderr to contain %q, got %q", "something broke", output)
	}
}

func TestWarning_WritesToStderr(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	output := captureStderr(t, func() {
		Warning("be careful")
	})
	if !strings.Contains(output, "be careful") {
		t.Errorf("expected stderr to contain %q, got %q", "be careful", output)
	}
}

func TestInfo_WritesToStderr(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	output := captureStderr(t, func() {
		Info("fyi")
	})
	if !strings.Contains(output, "fyi") {
		t.Errorf("expected stderr to contain %q, got %q", "fyi", output)
	}
}

func TestIsTTY_FalseInTests(t *testing.T) {
	if IsTTY() {
		t.Error("IsTTY should return false in test environment")
	}
}

func TestDisableColor_SetsNoColor(t *testing.T) {
	oldVal := color.NoColor
	defer func() { color.NoColor = oldVal }()

	DisableColor()
	if !color.NoColor {
		t.Error("DisableColor should set color.NoColor to true")
	}
}

func TestSetEmojiEnabled(t *testing.T) {
	// Save and restore
	old := emojiEnabled
	defer func() { emojiEnabled = old }()

	SetEmojiEnabled(false)
	color.NoColor = true
	defer func() { color.NoColor = false }()

	output := captureStderr(t, func() {
		Success("no emoji")
	})
	if strings.Contains(output, EmojiSuccess) {
		t.Error("emoji should not appear when disabled")
	}

	SetEmojiEnabled(true)
	output = captureStderr(t, func() {
		Success("with emoji")
	})
	if !strings.Contains(output, EmojiSuccess) {
		t.Error("emoji should appear when enabled")
	}
}
