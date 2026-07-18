package client

import (
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/go-openapi/runtime"

	"github.com/ramanavelineni/semctl/internal/style"
	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/operations"
)

// TargetSemaphoreVersion is the major.minor Semaphore version the generated
// API client (pkg/semapi) was built against. Bump when regenerating the
// client from a new spec (scripts/generate-api.sh).
const TargetSemaphoreVersion = "2.18"

// HTTPStatus extracts the HTTP status code from a go-swagger client error:
// spec-declared error responses implement Code(); undeclared statuses surface
// as *runtime.APIError. Returns 0 when err carries no status.
func HTTPStatus(err error) int {
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	var apiErr *runtime.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return 0
}

// IsNotFound reports whether err is an HTTP 404 from the API.
func IsNotFound(err error) bool {
	return HTTPStatus(err) == 404
}

var (
	versionOnce   sync.Once
	serverVersion string
)

// majorMinorRE captures the leading major.minor from strings like
// "v2.18.20^0-5d87656-1783280162".
var majorMinorRE = regexp.MustCompile(`^v?(\d+\.\d+)`)

func majorMinor(version string) string {
	if m := majorMinorRE.FindStringSubmatch(version); m != nil {
		return m[1]
	}
	return ""
}

// WarnIfVersionMismatch fetches the server version (once per process) and
// warns when its major.minor differs from the version the client targets.
// Best-effort: fetch failures are silent — commands must work against
// servers whose /info endpoint is unavailable.
func WarnIfVersionMismatch(api *apiclient.Semapi) {
	versionOnce.Do(func() {
		resp, err := api.Operations.GetInfo(operations.NewGetInfoParams(), nil)
		if err != nil {
			return
		}
		serverVersion = resp.GetPayload().Version
	})

	if serverVersion == "" {
		return
	}
	if got := majorMinor(serverVersion); got != "" && got != TargetSemaphoreVersion {
		style.Warning(fmt.Sprintf("Server runs Semaphore %s but semctl targets %s.x — some commands and fields may not work as expected.", serverVersion, TargetSemaphoreVersion))
	}
}
