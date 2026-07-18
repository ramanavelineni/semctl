package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-openapi/runtime"
)

func TestMajorMinor(t *testing.T) {
	cases := map[string]string{
		"v2.18.20^0-5d87656-1783280162": "2.18", // real /info format
		"v2.18.20":                      "2.18",
		"2.16.51":                       "2.16",
		"v3.0.0-beta1":                  "3.0",
		"development":                   "",
		"":                              "",
	}
	for in, want := range cases {
		if got := majorMinor(in); got != want {
			t.Errorf("majorMinor(%q) = %q, want %q", in, got, want)
		}
	}
}

type statusErr struct{ code int }

func (e *statusErr) Error() string { return fmt.Sprintf("status %d", e.code) }
func (e *statusErr) Code() int     { return e.code }

func TestHTTPStatus(t *testing.T) {
	if got := HTTPStatus(fmt.Errorf("wrap: %w", &statusErr{404})); got != 404 {
		t.Errorf("coded 404: got %d", got)
	}
	if got := HTTPStatus(fmt.Errorf("wrap: %w", runtime.NewAPIError("op", nil, 401))); got != 401 {
		t.Errorf("runtime 401: got %d", got)
	}
	if got := HTTPStatus(errors.New("plain")); got != 0 {
		t.Errorf("plain error: got %d, want 0", got)
	}
	if !IsNotFound(&statusErr{404}) || IsNotFound(&statusErr{500}) {
		t.Error("IsNotFound misclassified")
	}
}
