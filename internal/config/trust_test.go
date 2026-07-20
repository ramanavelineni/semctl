package config

import (
	"os"
	"path/filepath"
	"testing"
)

// trustTestSetup gives the test an isolated HOME (trust store location) and
// working directory containing a CWD config, restoring globals afterwards.
func trustTestSetup(t *testing.T, cwdConfig string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
		TrustPrompt = nil
		cfg = nil
		configFileUsed = ""
		loadedFromCWD = false
		skippedUntrusted = ""
	})

	if cwdConfig != "" {
		if err := os.WriteFile(filepath.Join(dir, "semctl.yaml"), []byte(cwdConfig), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const trustTestConfig = "current_context: cwdctx\ncontexts:\n  cwdctx:\n    server:\n      host: 10.9.9.9\n"

func TestLoad_UntrustedCWDConfigSkipped(t *testing.T) {
	trustTestSetup(t, trustTestConfig)
	TrustPrompt = nil

	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if LoadedFromCWD() {
		t.Error("untrusted CWD config must not be loaded")
	}
	if SkippedUntrustedConfig() != "semctl.yaml" {
		t.Errorf("SkippedUntrustedConfig() = %q, want semctl.yaml", SkippedUntrustedConfig())
	}
	if GetCurrentContext() == "cwdctx" {
		t.Error("context from untrusted config took effect")
	}
}

func TestLoad_TrustPromptAcceptLoadsAndPersists(t *testing.T) {
	trustTestSetup(t, trustTestConfig)
	prompts := 0
	TrustPrompt = func(path string, changed bool) bool {
		prompts++
		if changed {
			t.Error("first prompt must not report a changed file")
		}
		if !filepath.IsAbs(path) {
			t.Errorf("prompt path %q not absolute", path)
		}
		return true
	}

	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !LoadedFromCWD() || GetCurrentContext() != "cwdctx" {
		t.Fatal("accepted CWD config was not loaded")
	}

	// Second load: trust was persisted, no new prompt.
	TrustPrompt = func(string, bool) bool { t.Error("re-prompted despite persisted trust"); return false }
	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !LoadedFromCWD() {
		t.Error("persisted trust not honored on reload")
	}
	if prompts != 1 {
		t.Errorf("prompts = %d, want 1", prompts)
	}
}

func TestLoad_TrustPromptDeclineSkips(t *testing.T) {
	trustTestSetup(t, trustTestConfig)
	TrustPrompt = func(string, bool) bool { return false }

	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if LoadedFromCWD() {
		t.Error("declined CWD config must not be loaded")
	}

	// A decline is not remembered: the next run asks again.
	asked := false
	TrustPrompt = func(string, bool) bool { asked = true; return false }
	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !asked {
		t.Error("decline must not be persisted as distrust-forever")
	}
}

func TestLoad_ChangedContentRevokesTrust(t *testing.T) {
	dir := trustTestSetup(t, trustTestConfig)
	if err := TrustConfig("semctl.yaml"); err != nil {
		t.Fatalf("TrustConfig: %v", err)
	}

	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !LoadedFromCWD() {
		t.Fatal("trusted config not loaded")
	}

	// Edit the file: trust must lapse and the prompt must say "changed".
	if err := os.WriteFile(filepath.Join(dir, "semctl.yaml"), []byte(trustTestConfig+"# edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sawChanged := false
	TrustPrompt = func(_ string, changed bool) bool { sawChanged = changed; return false }
	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if LoadedFromCWD() {
		t.Error("edited config loaded on stale trust")
	}
	if !sawChanged {
		t.Error("prompt should report the file changed since it was trusted")
	}
}

func TestLoad_ExplicitConfigBypassesTrust(t *testing.T) {
	trustTestSetup(t, trustTestConfig)
	TrustPrompt = func(string, bool) bool { t.Error("explicit --config must not prompt"); return false }

	if err := Load("semctl.yaml"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if GetCurrentContext() != "cwdctx" {
		t.Error("explicitly named config was not loaded")
	}
	if SkippedUntrustedConfig() != "" {
		t.Error("explicit config must not be reported as skipped")
	}
}

func TestRevokeTrust(t *testing.T) {
	trustTestSetup(t, trustTestConfig)
	if err := TrustConfig("semctl.yaml"); err != nil {
		t.Fatalf("TrustConfig: %v", err)
	}

	paths, err := TrustedConfigs()
	if err != nil || len(paths) != 1 {
		t.Fatalf("TrustedConfigs = %v, %v; want one entry", paths, err)
	}

	removed, err := RevokeTrust("semctl.yaml")
	if err != nil || !removed {
		t.Fatalf("RevokeTrust = %v, %v; want removed", removed, err)
	}
	removed, err = RevokeTrust("semctl.yaml")
	if err != nil || removed {
		t.Fatalf("second RevokeTrust = %v, %v; want no-op", removed, err)
	}

	if err := Load(""); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if LoadedFromCWD() {
		t.Error("revoked config must not load")
	}
}

func TestTrustStore_Permissions(t *testing.T) {
	trustTestSetup(t, trustTestConfig)
	if err := TrustConfig("semctl.yaml"); err != nil {
		t.Fatalf("TrustConfig: %v", err)
	}
	storePath, err := trustStorePath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("trust store perms = %o, want 0600", perm)
	}
}
