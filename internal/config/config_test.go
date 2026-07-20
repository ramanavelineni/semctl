package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// writeTestConfig writes YAML content to a temp dir and returns the file path.
func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

func TestLoad_Empty(t *testing.T) {
	path := writeTestConfig(t, "")
	if err := Load(path); err != nil {
		t.Fatalf("Load empty config should not error: %v", err)
	}
}

func TestLoad_ExplicitFile(t *testing.T) {
	path := writeTestConfig(t, `current_context: prod`)
	if err := Load(path); err != nil {
		t.Fatalf("Load should not error: %v", err)
	}
	if got := GetCurrentContext(); got != "prod" {
		t.Errorf("got %q, want %q", got, "prod")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTestConfig(t, `{{{invalid`)
	err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestConfigFilePath(t *testing.T) {
	path := writeTestConfig(t, `current_context: default`)
	_ = Load(path)
	if got := ConfigFilePath(); got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestConfigFilePath_NilViper(t *testing.T) {
	cfg = nil
	configFileUsed = ""
	if got := ConfigFilePath(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestGetCurrentContext_Default(t *testing.T) {
	cfg = nil
	configFileUsed = ""
	if got := GetCurrentContext(); got != "default" {
		t.Errorf("got %q, want %q", got, "default")
	}
}

func TestGetCurrentContext_FromConfig(t *testing.T) {
	path := writeTestConfig(t, `current_context: staging`)
	_ = Load(path)
	if got := GetCurrentContext(); got != "staging" {
		t.Errorf("got %q, want %q", got, "staging")
	}
}

func TestSetCurrentContext_Persistence(t *testing.T) {
	path := writeTestConfig(t, `current_context: default`)
	_ = Load(path)

	if err := SetCurrentContext("production"); err != nil {
		t.Fatalf("SetCurrentContext error: %v", err)
	}
	if got := GetCurrentContext(); got != "production" {
		t.Errorf("got %q, want %q", got, "production")
	}

	// Verify it persisted to file
	_ = Load(path)
	if got := GetCurrentContext(); got != "production" {
		t.Errorf("after reload: got %q, want %q", got, "production")
	}
}

func TestApplyContext_Found(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
  staging:
    server:
      host: staging.example.com
`)
	_ = Load(path)

	if err := ApplyContext("staging"); err != nil {
		t.Fatalf("ApplyContext error: %v", err)
	}
	if got := GetCurrentContext(); got != "staging" {
		t.Errorf("got %q, want %q", got, "staging")
	}
}

func TestApplyContext_NotFound(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	_ = Load(path)

	err := ApplyContext("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent context")
	}
}

func TestListContexts_Empty(t *testing.T) {
	path := writeTestConfig(t, `current_context: default`)
	_ = Load(path)

	contexts := ListContexts()
	if len(contexts) != 0 {
		t.Errorf("expected 0 contexts, got %d", len(contexts))
	}
}

func TestListContexts_Sorted(t *testing.T) {
	path := writeTestConfig(t, `
contexts:
  staging:
    server:
      host: staging.example.com
  default:
    server:
      host: localhost
  production:
    server:
      host: prod.example.com
`)
	_ = Load(path)

	contexts := ListContexts()
	expected := []string{"default", "production", "staging"}
	if len(contexts) != len(expected) {
		t.Fatalf("got %d contexts, want %d", len(contexts), len(expected))
	}
	for i, name := range expected {
		if contexts[i] != name {
			t.Errorf("contexts[%d] = %q, want %q", i, contexts[i], name)
		}
	}
}

func TestListContexts_NilViper(t *testing.T) {
	cfg = nil
	configFileUsed = ""
	if got := ListContexts(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGetContextConfig_Found(t *testing.T) {
	path := writeTestConfig(t, `
contexts:
  default:
    server:
      host: 10.0.0.1
      port: 3000
      scheme: https
    auth:
      username: admin
      password: secret
      api_token: tok123
`)
	_ = Load(path)

	cc, err := GetContextConfig("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cc.ServerHost != "10.0.0.1" {
		t.Errorf("ServerHost: got %q, want %q", cc.ServerHost, "10.0.0.1")
	}
	if cc.ServerPort != 3000 {
		t.Errorf("ServerPort: got %d, want 3000", cc.ServerPort)
	}
	if cc.ServerScheme != "https" {
		t.Errorf("ServerScheme: got %q, want %q", cc.ServerScheme, "https")
	}
	if cc.AuthUsername != "admin" {
		t.Errorf("AuthUsername: got %q, want %q", cc.AuthUsername, "admin")
	}
	if cc.AuthPassword != "secret" {
		t.Errorf("AuthPassword: got %q, want %q", cc.AuthPassword, "secret")
	}
	if cc.AuthAPIToken != "tok123" {
		t.Errorf("AuthAPIToken: got %q, want %q", cc.AuthAPIToken, "tok123")
	}
}

func TestGetContextConfig_NotFound(t *testing.T) {
	path := writeTestConfig(t, `
contexts:
  default:
    server:
      host: localhost
`)
	_ = Load(path)

	_, err := GetContextConfig("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent context")
	}
}

func TestGetContextConfig_NoConfigLoaded(t *testing.T) {
	cfg = nil
	configFileUsed = ""
	_, err := GetContextConfig("default")
	if err == nil {
		t.Fatal("expected error when no config is loaded")
	}
}

func TestGetContextServerDisplay(t *testing.T) {
	path := writeTestConfig(t, `
contexts:
  default:
    server:
      host: 10.0.0.1
      port: 3000
      scheme: https
`)
	_ = Load(path)

	got := GetContextServerDisplay("default")
	want := "https://10.0.0.1:3000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetContextServerDisplay_DefaultScheme(t *testing.T) {
	path := writeTestConfig(t, `
contexts:
  default:
    server:
      host: 10.0.0.1
      port: 3000
`)
	_ = Load(path)

	got := GetContextServerDisplay("default")
	want := "http://10.0.0.1:3000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetServerURL(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: 10.0.0.1
      port: 8080
      scheme: https
`)
	_ = Load(path)

	got, err := GetServerURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://10.0.0.1:8080/api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetServerURL_DefaultPort(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: 10.0.0.1
`)
	_ = Load(path)

	got, err := GetServerURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "http://10.0.0.1:3000/api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseHostPort(t *testing.T) {
	cases := []struct {
		in      string
		host    string
		port    int
		wantErr bool
	}{
		{"10.0.0.1:3000", "10.0.0.1", 3000, false},
		{"myserver", "myserver", DefaultPort, false},
		{"example.com:8080", "example.com", 8080, false},
		{"  10.0.0.1:3000  ", "10.0.0.1", 3000, false},
		{"[::1]:3000", "::1", 3000, false},
		{"::1", "::1", DefaultPort, false},
		{"", "", 0, true},
		{"   ", "", 0, true},
		{"example.com:abc", "", 0, true},
		{"example.com:0", "", 0, true},
		{"example.com:70000", "", 0, true},
	}
	for _, c := range cases {
		host, port, err := ParseHostPort(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseHostPort(%q): expected error, got host=%q port=%d", c.in, host, port)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseHostPort(%q): unexpected error: %v", c.in, err)
			continue
		}
		if host != c.host || port != c.port {
			t.Errorf("ParseHostPort(%q): got (%q, %d), want (%q, %d)", c.in, host, port, c.host, c.port)
		}
	}
}

func TestEnvVarOverrides(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: 10.0.0.1
      port: 3000
    auth:
      username: cfguser
      password: cfgpass
      api_token: cfgtoken
`)
	_ = Load(path)

	t.Setenv("SEMCTL_AUTH_USERNAME", "envuser")
	t.Setenv("SEMCTL_AUTH_PASSWORD", "envpass")
	t.Setenv("SEMCTL_API_TOKEN", "envtoken")
	t.Setenv("SEMCTL_SERVER", "envhost:9999")
	t.Setenv("SEMCTL_SCHEME", "https")

	if got := GetUsername(); got != "envuser" {
		t.Errorf("GetUsername: got %q, want envuser", got)
	}
	if got := GetPassword(); got != "envpass" {
		t.Errorf("GetPassword: got %q, want envpass", got)
	}
	if got := GetAPIToken(); got != "envtoken" {
		t.Errorf("GetAPIToken: got %q, want envtoken", got)
	}

	host, port, scheme, err := ResolveServer()
	if err != nil {
		t.Fatalf("ResolveServer: unexpected error: %v", err)
	}
	if host != "envhost" || port != 9999 || scheme != "https" {
		t.Errorf("ResolveServer: got (%q, %d, %q), want (envhost, 9999, https)", host, port, scheme)
	}
}

func TestResolveServer_FlagBeatsEnv(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: 10.0.0.1
`)
	_ = Load(path)

	t.Setenv("SEMCTL_SERVER", "envhost:9999")
	SetServerOverride("flaghost:8888")
	defer SetServerOverride("")

	host, port, _, err := ResolveServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "flaghost" || port != 8888 {
		t.Errorf("got (%q, %d), want (flaghost, 8888)", host, port)
	}
}

func TestResolveServer_NoServer(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
`)
	_ = Load(path)
	SetServerOverride("")

	_, _, _, err := ResolveServer()
	if err == nil {
		t.Fatal("expected error when no server is configured")
	}
}

func TestGetUsername(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    auth:
      username: admin
`)
	_ = Load(path)

	if got := GetUsername(); got != "admin" {
		t.Errorf("got %q, want %q", got, "admin")
	}
}

func TestGetPassword(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    auth:
      password: secret
`)
	_ = Load(path)

	if got := GetPassword(); got != "secret" {
		t.Errorf("got %q, want %q", got, "secret")
	}
}

func TestGetAPIToken(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    auth:
      api_token: mytoken
`)
	_ = Load(path)

	if got := GetAPIToken(); got != "mytoken" {
		t.Errorf("got %q, want %q", got, "mytoken")
	}
}

func TestGetDefaultProjectID(t *testing.T) {
	path := writeTestConfig(t, `
defaults:
  project_id: 42
`)
	_ = Load(path)

	if got := GetDefaultProjectID(); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestGetDefaultProjectID_NilViper(t *testing.T) {
	cfg = nil
	configFileUsed = ""
	if got := GetDefaultProjectID(); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestGetOutputFormat(t *testing.T) {
	path := writeTestConfig(t, `
output:
  format: json
`)
	_ = Load(path)

	if got := GetOutputFormat(); got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestGetOutputFormat_NilViper(t *testing.T) {
	cfg = nil
	configFileUsed = ""
	if got := GetOutputFormat(); got != "table" {
		t.Errorf("got %q, want %q", got, "table")
	}
}

func TestSaveContext_New(t *testing.T) {
	path := writeTestConfig(t, `current_context: default`)
	_ = Load(path)

	serverData := map[string]interface{}{
		"host":   "10.0.0.1",
		"port":   3000,
		"scheme": "http",
	}
	authData := map[string]interface{}{
		"username": "admin",
		"password": "secret",
	}

	if err := SaveContext("prod", serverData, authData); err != nil {
		t.Fatalf("SaveContext error: %v", err)
	}

	// Reload and verify
	_ = Load(path)
	cc, err := GetContextConfig("prod")
	if err != nil {
		t.Fatalf("GetContextConfig error: %v", err)
	}
	if cc.ServerHost != "10.0.0.1" {
		t.Errorf("ServerHost: got %q, want %q", cc.ServerHost, "10.0.0.1")
	}
	if cc.AuthUsername != "admin" {
		t.Errorf("AuthUsername: got %q, want %q", cc.AuthUsername, "admin")
	}
}

func TestSaveContext_Update(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
      port: 3000
    auth:
      username: old
`)
	_ = Load(path)

	serverData := map[string]interface{}{
		"host":   "newhost",
		"port":   4000,
		"scheme": "https",
	}
	authData := map[string]interface{}{
		"username": "newuser",
		"password": "newpass",
	}

	if err := SaveContext("default", serverData, authData); err != nil {
		t.Fatalf("SaveContext error: %v", err)
	}

	_ = Load(path)
	cc, err := GetContextConfig("default")
	if err != nil {
		t.Fatalf("GetContextConfig error: %v", err)
	}
	if cc.ServerHost != "newhost" {
		t.Errorf("ServerHost: got %q, want %q", cc.ServerHost, "newhost")
	}
	if cc.AuthUsername != "newuser" {
		t.Errorf("AuthUsername: got %q, want %q", cc.AuthUsername, "newuser")
	}
}

func TestDeleteContext_Found(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
  staging:
    server:
      host: staging.example.com
`)
	_ = Load(path)

	if err := DeleteContext("staging"); err != nil {
		t.Fatalf("DeleteContext error: %v", err)
	}

	_ = Load(path)
	contexts := ListContexts()
	for _, c := range contexts {
		if c == "staging" {
			t.Error("staging should have been deleted")
		}
	}
}

func TestDeleteContext_NotFound(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	_ = Load(path)

	err := DeleteContext("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent context")
	}
}

func TestDeleteContext_SwitchesCurrent(t *testing.T) {
	path := writeTestConfig(t, `
current_context: staging
contexts:
  default:
    server:
      host: localhost
  staging:
    server:
      host: staging.example.com
`)
	_ = Load(path)

	if err := DeleteContext("staging"); err != nil {
		t.Fatalf("DeleteContext error: %v", err)
	}

	_ = Load(path)
	cur := GetCurrentContext()
	if cur == "staging" {
		t.Error("current context should have switched away from deleted context")
	}
}

func TestRenameContext_Success(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	_ = Load(path)

	if err := RenameContext("default", "production"); err != nil {
		t.Fatalf("RenameContext error: %v", err)
	}

	_ = Load(path)
	contexts := ListContexts()
	found := false
	for _, c := range contexts {
		if c == "production" {
			found = true
		}
		if c == "default" {
			t.Error("old name 'default' should not exist")
		}
	}
	if !found {
		t.Error("new name 'production' not found")
	}
}

func TestRenameContext_NotFound(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	_ = Load(path)

	err := RenameContext("nonexistent", "newname")
	if err == nil {
		t.Fatal("expected error for nonexistent context")
	}
}

func TestRenameContext_AlreadyExists(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
  staging:
    server:
      host: staging.example.com
`)
	_ = Load(path)

	err := RenameContext("default", "staging")
	if err == nil {
		t.Fatal("expected error when target name already exists")
	}
}

func TestRenameContext_UpdatesCurrent(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
`)
	_ = Load(path)

	if err := RenameContext("default", "production"); err != nil {
		t.Fatalf("RenameContext error: %v", err)
	}

	_ = Load(path)
	if got := GetCurrentContext(); got != "production" {
		t.Errorf("got %q, want %q", got, "production")
	}
}

func TestRemoveAuthConfig(t *testing.T) {
	path := writeTestConfig(t, `
current_context: default
contexts:
  default:
    server:
      host: localhost
    auth:
      username: admin
      password: secret
`)
	_ = Load(path)

	if err := RemoveAuthConfig(); err != nil {
		t.Fatalf("RemoveAuthConfig error: %v", err)
	}

	_ = Load(path)
	cc, err := GetContextConfig("default")
	if err != nil {
		t.Fatalf("GetContextConfig error: %v", err)
	}
	if cc.AuthUsername != "" {
		t.Errorf("expected empty username, got %q", cc.AuthUsername)
	}
	if cc.AuthPassword != "" {
		t.Errorf("expected empty password, got %q", cc.AuthPassword)
	}
}

func TestValidateContextName(t *testing.T) {
	valid := []string{"default", "prod", "my-context", "ctx_2", "A1"}
	for _, name := range valid {
		if err := ValidateContextName(name); err != nil {
			t.Errorf("ValidateContextName(%q) = %v, want nil", name, err)
		}
	}
	invalid := []string{"", "../../evil", "a/b", `a\b`, "a.b", ".hidden", "-lead", "ctx name", strings.Repeat("x", 65)}
	for _, name := range invalid {
		if err := ValidateContextName(name); err == nil {
			t.Errorf("ValidateContextName(%q) = nil, want error", name)
		}
	}
}

func TestLoad_RejectsTraversalCurrentContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("current_context: ../../evil\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err == nil {
		t.Fatal("expected error for path-traversal current_context")
	}
}

func TestServerRedirected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := `current_context: prod
contexts:
  prod:
    server:
      host: sem.example
      port: 3000
`
	if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	defer SetServerOverride("")

	SetServerOverride("")
	if ServerRedirected() {
		t.Error("no override should not redirect")
	}
	SetServerOverride("sem.example:3000")
	if ServerRedirected() {
		t.Error("override matching the context server should not redirect")
	}
	SetServerOverride("attacker.example:3000")
	if !ServerRedirected() {
		t.Error("override pointing elsewhere should redirect")
	}
	SetServerOverride("sem.example:9999")
	if !ServerRedirected() {
		t.Error("same host, different port should redirect")
	}
}

func TestContextNames_CaseInsensitive(t *testing.T) {
	path := writeTestConfig(t, "current_context: prod\ncontexts:\n  Prod:\n    server: {host: a}\n")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ListContexts(), ","); got != "prod" {
		t.Errorf("ListContexts = %q, want lowercased %q", got, "prod")
	}
	cc, err := GetContextConfig("PROD")
	if err != nil || cc.ServerHost != "a" {
		t.Errorf("GetContextConfig(PROD) = %+v, %v", cc, err)
	}
	if got := GetCurrentContext(); got != "prod" {
		t.Errorf("GetCurrentContext = %q, want %q", got, "prod")
	}
}

func TestLoad_CaseCollisionRejected(t *testing.T) {
	path := writeTestConfig(t, "contexts:\n  Prod:\n    server: {host: a}\n  prod:\n    server: {host: b}\n")
	err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Errorf("expected case-collision error, got %v", err)
	}
}

func TestSaveContext_NormalizesAndReplacesCase(t *testing.T) {
	path := writeTestConfig(t, "contexts:\n  Prod:\n    server: {host: old}\n")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if err := SaveContext("PROD", map[string]interface{}{"host": "new"}, map[string]interface{}{}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "Prod:") || strings.Contains(string(raw), "PROD:") {
		t.Errorf("mixed-case context key survived save:\n%s", raw)
	}
	cc, err := GetContextConfig("prod")
	if err != nil || cc.ServerHost != "new" {
		t.Errorf("after save: %+v, %v", cc, err)
	}
}

func TestUpdate_RefusesToClobberCorruptConfig(t *testing.T) {
	path := writeTestConfig(t, "current_context: default\n")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{{{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	err := SaveContext("x", map[string]interface{}{"host": "h"}, map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("expected refusal on corrupt config, got %v", err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "{{{invalid" {
		t.Error("corrupt config was clobbered")
	}
}

func TestUpdate_PreservesUnknownKeys(t *testing.T) {
	path := writeTestConfig(t, "custom_key: keepme\ncurrent_context: default\n")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if err := SetCurrentContext("default"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "custom_key: keepme") {
		t.Errorf("unknown key dropped on save:\n%s", raw)
	}
}

func TestConcurrentSaves_NoLostUpdates(t *testing.T) {
	path := writeTestConfig(t, "current_context: default\n")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := SaveContext(fmt.Sprintf("ctx%d", n),
				map[string]interface{}{"host": fmt.Sprintf("h%d", n)},
				map[string]interface{}{}); err != nil {
				t.Errorf("save %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()
	if err := Load(path); err != nil {
		t.Fatalf("config corrupted by concurrent saves: %v", err)
	}
	if got := len(ListContexts()); got != 20 {
		t.Errorf("%d contexts survived, want 20 (lost updates)", got)
	}
}

func TestLoad_DuplicateKeyRejected(t *testing.T) {
	path := writeTestConfig(t, "contexts:\n  prod:\n    server: {host: a}\n  prod:\n    server: {host: b}\n")
	if err := Load(path); err == nil {
		t.Error("expected error for duplicate mapping key")
	}
}
