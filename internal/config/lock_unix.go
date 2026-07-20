//go:build !windows

package config

import (
	"fmt"
	"os"
	"syscall"
)

// withConfigLock holds an exclusive advisory flock on <path>.lock while fn
// runs, serializing config read-modify-write cycles across processes.
func withConfigLock(path string, fn func() error) error {
	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("failed to lock config: %w", err)
	}
	defer func() { _ = f.Close() }()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock config: %w", err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	return fn()
}
