package style

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	Green   = color.New(color.FgGreen).SprintFunc()
	Red     = color.New(color.FgRed).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	Bold    = color.New(color.Bold).SprintFunc()
	BoldRed = color.New(color.Bold, color.FgRed).SprintFunc()

	EmojiSuccess = "✅"
	EmojiError   = "❌"
	EmojiWarning = "⚠️"
	EmojiInfo    = "ℹ️"
	EmojiProject = "📁"
	EmojiKey     = "🔑"
	EmojiTask    = "🚀"
	EmojiServer  = "🖥️"

	emojiEnabled = true
)

// SetEmojiEnabled enables or disables emoji output.
func SetEmojiEnabled(enabled bool) {
	emojiEnabled = enabled
}

func emoji(e string) string {
	if emojiEnabled {
		return e + " "
	}
	return ""
}

// Success prints a success message to stderr.
func Success(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s\n", emoji(EmojiSuccess), Green(msg))
}

// Error prints an error message to stderr.
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s\n", emoji(EmojiError), Red(msg))
}

// Warning prints a warning message to stderr.
func Warning(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s\n", emoji(EmojiWarning), Yellow(msg))
}

// Info prints an info message to stderr.
func Info(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s\n", emoji(EmojiInfo), Cyan(msg))
}

// DisableColor disables color output globally.
func DisableColor() {
	color.NoColor = true
}

// IsTTY returns whether stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
