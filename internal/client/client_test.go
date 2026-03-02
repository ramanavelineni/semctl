package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestServer creates a mock Semaphore server for auth flow testing.
func newTestServer(t *testing.T, loginStatus int, tokenStatus int, tokenBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			w.WriteHeader(loginStatus)
		case "/api/user/tokens":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tokenStatus)
			_, _ = w.Write([]byte(tokenBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestLoginAndCreateToken_Success(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, http.StatusCreated, `{"id":"test-token-123"}`)
	defer ts.Close()

	token, err := LoginAndCreateToken(ts.URL+"/api", "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token-123" {
		t.Errorf("got %q, want %q", token, "test-token-123")
	}
}

func TestLoginAndCreateToken_LoginFails(t *testing.T) {
	ts := newTestServer(t, http.StatusUnauthorized, http.StatusCreated, `{"id":"tok"}`)
	defer ts.Close()

	_, err := LoginAndCreateToken(ts.URL+"/api", "admin", "wrong")
	if err == nil {
		t.Fatal("expected error for failed login")
	}
}

func TestLoginAndCreateToken_TokenCreationFails(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, http.StatusInternalServerError, ``)
	defer ts.Close()

	_, err := LoginAndCreateToken(ts.URL+"/api", "admin", "password")
	if err == nil {
		t.Fatal("expected error for failed token creation")
	}
}

func TestLoginAndCreateToken_EmptyToken(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, http.StatusCreated, `{"id":""}`)
	defer ts.Close()

	_, err := LoginAndCreateToken(ts.URL+"/api", "admin", "password")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestLoginAndCreateToken_InvalidJSON(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, http.StatusCreated, `not json`)
	defer ts.Close()

	_, err := LoginAndCreateToken(ts.URL+"/api", "admin", "password")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoginAndCreateToken_NetworkError(t *testing.T) {
	_, err := LoginAndCreateToken("http://127.0.0.1:1/api", "admin", "password")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}

func TestLoginAndCreateToken_Login200OK(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, http.StatusCreated, `{"id":"tok-200"}`)
	defer ts.Close()

	token, err := LoginAndCreateToken(ts.URL+"/api", "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok-200" {
		t.Errorf("got %q, want %q", token, "tok-200")
	}
}

func TestLoginAndCreateToken_Token200OK(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, http.StatusOK, `{"id":"tok-ok"}`)
	defer ts.Close()

	token, err := LoginAndCreateToken(ts.URL+"/api", "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok-ok" {
		t.Errorf("got %q, want %q", token, "tok-ok")
	}
}

func TestTokenCachePathForContext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	got := TokenCachePathForContext("prod")
	want := filepath.Join(dir, "semctl", "tokens", "prod.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTokenCachePathForContext_DefaultPath(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")

	got := TokenCachePathForContext("default")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".cache", "semctl", "tokens", "default.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveTokenCacheForContext_WritesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	if err := SaveTokenCacheForContext("test", "my-secret-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := TokenCachePathForContext("test")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}

	var cache tokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("failed to unmarshal cache: %v", err)
	}
	if cache.Token != "my-secret-token" {
		t.Errorf("got %q, want %q", cache.Token, "my-secret-token")
	}
}

func TestSaveTokenCacheForContext_Permissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	if err := SaveTokenCacheForContext("perms", "tok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := TokenCachePathForContext("perms")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions: got %o, want 0600", perm)
	}
}

func TestSaveTokenCacheForContext_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	tokensDir := filepath.Join(dir, "semctl", "tokens")
	if _, err := os.Stat(tokensDir); !os.IsNotExist(err) {
		t.Fatal("tokens dir should not exist before save")
	}

	if err := SaveTokenCacheForContext("test", "tok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(tokensDir); err != nil {
		t.Fatalf("tokens dir should exist after save: %v", err)
	}
}

func TestSaveAndLoadTokenCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)

	if err := SaveTokenCacheForContext("roundtrip", "rt-token-456"); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Read the file back manually (loadCachedToken uses the current context which we can't easily set)
	path := TokenCachePathForContext("roundtrip")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var cache tokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if cache.Token != "rt-token-456" {
		t.Errorf("got %q, want %q", cache.Token, "rt-token-456")
	}
}
