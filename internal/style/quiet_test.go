package style

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestQuietSuppressesSuccessAndInfoOnly(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	SetQuiet(true)
	Success("hidden-success")
	Info("hidden-info")
	Warning("visible-warning")
	Error("visible-error")
	SetQuiet(false)

	_ = w.Close()
	os.Stderr = old
	out, _ := io.ReadAll(r)

	s := string(out)
	if strings.Contains(s, "hidden-success") || strings.Contains(s, "hidden-info") {
		t.Errorf("quiet mode leaked success/info: %q", s)
	}
	if !strings.Contains(s, "visible-warning") || !strings.Contains(s, "visible-error") {
		t.Errorf("quiet mode suppressed warning/error: %q", s)
	}
}
