package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-openapi/runtime"

	"github.com/ramanavelineni/semctl/internal/client"
)

// codedErr mimics a go-swagger spec-declared error response (has Code()).
type codedErr struct{ code int }

func (c *codedErr) Error() string { return fmt.Sprintf("status %d", c.code) }
func (c *codedErr) Code() int     { return c.code }

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, exitOK},
		{"generic", errors.New("boom"), exitError},
		{"cancelled wrapped", fmt.Errorf("wrapped: %w", errCancelled), exitCancelled},
		{"explicit code", withExitCode(errors.New("drift"), exitDrift), exitDrift},
		{"explicit task failed", withExitCode(errors.New("task"), exitTaskFailed), exitTaskFailed},
		{"no credentials", fmt.Errorf("auth: %w", client.ErrNoCredentials), exitAuth},
		{"rejected login", fmt.Errorf("auth: %w", client.ErrAuthFailed), exitAuth},
		{"typed 401", fmt.Errorf("call: %w", &codedErr{401}), exitAuth},
		{"typed 403", fmt.Errorf("call: %w", &codedErr{403}), exitAuth},
		{"typed 404", fmt.Errorf("call: %w", &codedErr{404}), exitNotFound},
		{"typed 500", fmt.Errorf("call: %w", &codedErr{500}), exitError},
		{"runtime 401", fmt.Errorf("call: %w", runtime.NewAPIError("op", nil, 401)), exitAuth},
		{"runtime 404", fmt.Errorf("call: %w", runtime.NewAPIError("op", nil, 404)), exitNotFound},
	}
	for _, tc := range cases {
		if got := exitCodeFor(tc.err); got != tc.want {
			t.Errorf("%s: exitCodeFor() = %d, want %d", tc.name, got, tc.want)
		}
	}
}
