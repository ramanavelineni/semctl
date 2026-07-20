package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// A config file auto-discovered in the working directory can redirect semctl
// — and any env credentials — to a server the user never chose (ROADMAP
// §1.1). Loading one therefore requires opt-in trust, direnv-style: the user
// consents once (interactive prompt or 'semctl trust') and the file's path +
// content hash are recorded; a content change revokes trust until re-granted.
// Explicit --config paths never pass through this gate — naming the file is
// consent.

// TrustPrompt asks the user whether to trust an auto-discovered
// working-directory config. changed=true means the file was trusted before
// but its content differs now. Set by cmd at startup; nil (e.g. in tests or
// non-interactive runs) means never trust without 'semctl trust'.
var TrustPrompt func(path string, changed bool) bool

// skippedUntrusted is the CWD config Load refused to auto-load this run, ""
// if none. The CLI turns it into a warning with the trust instructions.
var skippedUntrusted string

// SkippedUntrustedConfig returns the working-directory config file that was
// ignored because it is not trusted, or "".
func SkippedUntrustedConfig() string {
	return skippedUntrusted
}

// FindCWDConfig returns the auto-discoverable config file in the current
// working directory ("" if none). Same candidates, same order, as Load.
func FindCWDConfig() string {
	for _, name := range []string{"semctl.yaml", ".semctl.yaml"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

// trustStore mirrors trusted.yaml: absolute config path → sha256 of the
// content that was trusted.
type trustStore struct {
	Trusted map[string]string `yaml:"trusted"`
}

func trustStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate home directory for the trust store: %w", err)
	}
	return filepath.Join(home, ".config", "semctl", "trusted.yaml"), nil
}

func loadTrustStore() (*trustStore, string, error) {
	path, err := trustStorePath()
	if err != nil {
		return nil, "", err
	}
	ts := &trustStore{Trusted: map[string]string{}}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ts, path, nil
	}
	if err != nil {
		return nil, "", err
	}
	if err := yaml.Unmarshal(raw, ts); err != nil {
		return nil, "", fmt.Errorf("trust store %s: %w", path, err)
	}
	if ts.Trusted == nil {
		ts.Trusted = map[string]string{}
	}
	return ts, path, nil
}

// saveTrustStore writes atomically (temp + rename, 0600) like the config
// file, so a crash can't leave a half-written store that would then be
// refused on every load.
func saveTrustStore(path string, ts *trustStore) error {
	out, err := yaml.Marshal(ts)
	if err != nil {
		return fmt.Errorf("failed to marshal trust store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".trusted-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to write trust store: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write trust store: %w", err)
	}
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write trust store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write trust store: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to write trust store: %w", err)
	}
	return nil
}

func hashConfigContent(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// trustedCWDConfigContent decides whether the auto-discovered working-
// directory config may be loaded: already trusted with unchanged content, or
// the user accepts the trust prompt now (recorded for next time). On success
// it returns the exact bytes that were hashed, so Load parses what was
// vetted rather than re-reading a file that could have changed in between.
func trustedCWDConfigContent(path string) ([]byte, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, false
	}
	hash := hashConfigContent(raw)

	previous := ""
	if ts, _, err := loadTrustStore(); err == nil {
		previous = ts.Trusted[abs]
	}
	if previous == hash {
		return raw, true
	}

	if TrustPrompt == nil || !TrustPrompt(abs, previous != "") {
		return nil, false
	}
	// Consent was given interactively; a failure to persist it only means
	// the prompt reappears next run. Record the vetted bytes, not the file's
	// possibly-changed current content.
	if ts, storePath, err := loadTrustStore(); err == nil {
		ts.Trusted[abs] = hash
		_ = saveTrustStore(storePath, ts)
	}
	return raw, true
}

// TrustConfig records path's current content as trusted.
func TrustConfig(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	ts, storePath, err := loadTrustStore()
	if err != nil {
		return err
	}
	ts.Trusted[abs] = hashConfigContent(raw)
	return saveTrustStore(storePath, ts)
}

// RevokeTrust removes path from the trust store. The bool reports whether an
// entry existed.
func RevokeTrust(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	ts, storePath, err := loadTrustStore()
	if err != nil {
		return false, err
	}
	if _, ok := ts.Trusted[abs]; !ok {
		return false, nil
	}
	delete(ts.Trusted, abs)
	return true, saveTrustStore(storePath, ts)
}

// TrustedConfigs returns the trusted config paths, sorted.
func TrustedConfigs() ([]string, error) {
	ts, _, err := loadTrustStore()
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(ts.Trusted))
	for p := range ts.Trusted {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}
