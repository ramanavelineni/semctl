//go:build windows

package config

// withConfigLock is a no-op on Windows: flock semantics aren't portable, and
// the atomic temp+rename in writeConfigFile still prevents torn files.
func withConfigLock(_ string, fn func() error) error {
	return fn()
}
