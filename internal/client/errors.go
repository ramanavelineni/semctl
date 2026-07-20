package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/go-openapi/runtime"
)

// translatedAPIError replaces a go-swagger error's operation-dump text
// ("[GET /project/{project_id}/…] …NotFound (status 404): {}") with a human
// message, while keeping the original error in the chain and exposing the
// status code so exit-code mapping (client.HTTPStatus) still works.
type translatedAPIError struct {
	msg  string
	code int
	err  error
}

func (e *translatedAPIError) Error() string { return e.msg }
func (e *translatedAPIError) Code() int     { return e.code }
func (e *translatedAPIError) Unwrap() error { return e.err }

// serverSaysRE pulls a non-empty JSON body out of a go-swagger error string,
// e.g. `... (status 400): {"error":"name is required"}`.
var serverSaysRE = regexp.MustCompile(`\(status \d+\):? (\{.+\})\s*$`)

// TranslateAPIError converts go-swagger client errors into human messages.
// Non-API errors (and nil) pass through unchanged except for unreachable-
// server errors, which gain the resolved server identity.
func TranslateAPIError(err error) error {
	if err == nil {
		return nil
	}

	// Ctrl-C during a request: "cannot reach <server>" would be misleading.
	if errors.Is(err, context.Canceled) {
		return &translatedAPIError{msg: "interrupted", err: err}
	}

	server := ""
	if id, sErr := resolvedServerID(); sErr == nil {
		server = id
	}

	// Transport-level failures: the server was never reached.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		target := server
		if target == "" {
			target = urlErr.URL
		}
		return &translatedAPIError{
			msg: fmt.Sprintf("cannot reach %s: %v", target, urlErr.Err),
			err: err,
		}
	}

	status := HTTPStatus(err)
	if status == 0 {
		return err
	}

	var msg string
	switch {
	case status == 400:
		msg = "the server rejected the request as invalid"
	case status == 401:
		msg = "authentication rejected — token expired or revoked? Run 'semctl login' or check SEMCTL_API_TOKEN"
	case status == 403:
		msg = "permission denied — this action may require admin rights"
	case status == 404:
		msg = "not found"
	case status >= 500:
		msg = "server error"
	default:
		msg = "request failed"
	}

	// Surface the server's own explanation when the body carries one.
	if m := serverSaysRE.FindStringSubmatch(err.Error()); m != nil && m[1] != "{}" {
		msg += " — server says: " + m[1]
	}

	detail := fmt.Sprintf("status %d", status)
	if server != "" {
		detail += ", server " + server
	}
	msg = fmt.Sprintf("%s (%s)", msg, detail)

	// The raw operation dump remains available for debugging.
	if os.Getenv("SEMCTL_DEBUG") != "" {
		msg += " [" + strings.TrimSpace(err.Error()) + "]"
	}

	return &translatedAPIError{msg: msg, code: status, err: err}
}

// translatingTransport wraps the go-swagger transport so every API call
// returns translated errors — one hook instead of ~40 call-site edits, and
// it covers apply's reconciler/executor for free.
type translatingTransport struct {
	inner runtime.ContextualTransport
}

func (t *translatingTransport) Submit(op *runtime.ClientOperation) (interface{}, error) {
	res, err := t.inner.Submit(op)
	if err != nil {
		return res, TranslateAPIError(err)
	}
	return res, nil
}

func (t *translatingTransport) SubmitContext(ctx context.Context, op *runtime.ClientOperation) (interface{}, error) {
	// The generated client passes context.Background() when the call site
	// set no context of its own, so Ctrl-C must be grafted on here to reach
	// in-flight requests. Done() == nil means ctx can never be cancelled;
	// deriving from it (rather than replacing it) keeps any values. Safe to
	// cancel on return: the response body is fully consumed inside Submit.
	if rootCtx != nil && ctx.Done() == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		defer context.AfterFunc(rootCtx, cancel)()
	}
	res, err := t.inner.SubmitContext(ctx, op)
	if err != nil {
		return res, TranslateAPIError(err)
	}
	return res, nil
}
