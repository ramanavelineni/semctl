package client

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// swaggerErr mimics a go-swagger typed error response.
type swaggerErr struct {
	code int
	text string
}

func (e *swaggerErr) Error() string { return e.text }
func (e *swaggerErr) Code() int     { return e.code }

func TestTranslateAPIError(t *testing.T) {
	t.Setenv("SEMCTL_SERVER", "sem.example:3000")

	cases := []struct {
		name string
		err  error
		want []string
	}{
		{"404", &swaggerErr{404, "[GET /x] opName (status 404): {}"},
			[]string{"not found", "status 404", "server http://sem.example:3000"}},
		{"401 hint", &swaggerErr{401, "op (status 401): {}"},
			[]string{"authentication rejected", "semctl login"}},
		{"403 hint", &swaggerErr{403, "op (status 403): {}"},
			[]string{"permission denied", "admin"}},
		{"400 with body", &swaggerErr{400, `op (status 400): {"error":"name is required"}`},
			[]string{"rejected the request", `server says: {"error":"name is required"}`}},
		{"500", &swaggerErr{500, "op (status 500): {}"},
			[]string{"server error", "status 500"}},
	}
	for _, tc := range cases {
		got := TranslateAPIError(tc.err)
		for _, want := range tc.want {
			if !strings.Contains(got.Error(), want) {
				t.Errorf("%s: %q missing %q", tc.name, got.Error(), want)
			}
		}
		// The raw operation dump must be gone from the display text.
		if strings.Contains(got.Error(), "[GET /x]") {
			t.Errorf("%s: raw dump leaked into message: %q", tc.name, got.Error())
		}
		// Exit-code mapping must still see the status.
		if HTTPStatus(got) != tc.err.(*swaggerErr).code {
			t.Errorf("%s: HTTPStatus lost: got %d", tc.name, HTTPStatus(got))
		}
		// The original stays reachable for debugging.
		var orig *swaggerErr
		if !errors.As(got, &orig) {
			t.Errorf("%s: original error not in chain", tc.name)
		}
	}
}

func TestTranslateAPIError_Unreachable(t *testing.T) {
	t.Setenv("SEMCTL_SERVER", "sem.example:3000")
	err := TranslateAPIError(&url.Error{Op: "Get", URL: "http://sem.example:3000/api/projects", Err: errors.New("connection refused")})
	if !strings.Contains(err.Error(), "cannot reach http://sem.example:3000") ||
		!strings.Contains(err.Error(), "connection refused") {
		t.Errorf("unreachable message: %q", err.Error())
	}
}

func TestTranslateAPIError_Passthrough(t *testing.T) {
	if TranslateAPIError(nil) != nil {
		t.Error("nil must stay nil")
	}
	plain := fmt.Errorf("some local problem")
	if got := TranslateAPIError(plain); got != plain {
		t.Errorf("non-API error must pass through, got %v", got)
	}
}

func TestTranslateAPIError_DebugAppendsOriginal(t *testing.T) {
	t.Setenv("SEMCTL_SERVER", "sem.example:3000")
	t.Setenv("SEMCTL_DEBUG", "1")
	got := TranslateAPIError(&swaggerErr{404, "[GET /x] opName (status 404): {}"})
	if !strings.Contains(got.Error(), "[GET /x] opName") {
		t.Errorf("SEMCTL_DEBUG should append the original dump: %q", got.Error())
	}
}
