package cmd

import (
	"context"
	"errors"

	"github.com/ramanavelineni/semctl/internal/client"
)

// Exit codes let scripts branch on the failure class. Documented in the root
// command help and the README. 2 follows the terraform plan -detailed-exitcode
// convention (changes pending), so it is reserved for apply.
const (
	exitOK          = 0
	exitError       = 1 // generic failure
	exitDrift       = 2 // apply --detailed-exitcode: plan has changes
	exitAuth        = 3 // authentication failure (401/403, rejected login, no credentials)
	exitNotFound    = 4 // resource not found (404)
	exitCancelled   = 5 // user declined a confirmation or aborted a form
	exitTaskFailed  = 6 // task run --wait: task finished with error/stopped status
	exitWaitTimeout = 7 // task run --wait-timeout expired while the task still ran
)

// exitCodeError attaches a specific process exit code to an error.
type exitCodeError struct {
	err  error
	code int
}

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }

// withExitCode wraps err so Execute exits with code instead of the generic 1.
func withExitCode(err error, code int) error {
	return &exitCodeError{err: err, code: code}
}

// exitCodeFor maps a command error to the process exit code.
func exitCodeFor(err error) int {
	if err == nil {
		return exitOK
	}
	var ec *exitCodeError
	if errors.As(err, &ec) {
		return ec.code
	}
	if errors.Is(err, errCancelled) || errors.Is(err, context.Canceled) {
		return exitCancelled
	}
	if errors.Is(err, client.ErrNoCredentials) || errors.Is(err, client.ErrAuthFailed) {
		return exitAuth
	}

	switch client.HTTPStatus(err) {
	case 401, 403:
		return exitAuth
	case 404:
		return exitNotFound
	}

	return exitError
}
