package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
)

// tokenCache holds a cached API token bound to the server it was issued by.
// Server binding stops a redefined context (e.g. from a current-directory
// config file) from redirecting the token to a different host. Legacy cache
// files without the server field are treated as invalid.
type tokenCache struct {
	Token  string `json:"token"`
	Server string `json:"server"`
}

// ServerID renders the canonical server identity stored in the token cache.
func ServerID(scheme, host string, port int) string {
	return scheme + "://" + joinHostPort(host, port)
}

// resolvedServerID returns the ServerID of the currently resolved server.
func resolvedServerID() (string, error) {
	host, port, scheme, err := config.ResolveServer()
	if err != nil {
		return "", err
	}
	return ServerID(scheme, host, port), nil
}

var (
	// ErrNoCredentials means no token or username/password is configured.
	// Exposed as a sentinel so the CLI can map it to a distinct exit code.
	ErrNoCredentials = errors.New("no credentials available")

	// ErrAuthFailed means the server rejected the provided credentials.
	ErrAuthFailed = errors.New("the server rejected the credentials")

	// requestTimeout is the per-request HTTP timeout.
	requestTimeout = 30 * time.Second

	// insecureSkipVerify and caCertPath are session TLS overrides (from flags
	// or env); they are merged with the context config in buildTLSConfig.
	insecureSkipVerify bool
	caCertPath         string
)

// SetTimeout overrides the per-request HTTP timeout.
func SetTimeout(d time.Duration) {
	if d > 0 {
		requestTimeout = d
	}
}

// SetInsecureSkipVerify disables TLS certificate verification for the session.
func SetInsecureSkipVerify(insecure bool) {
	insecureSkipVerify = insecure
}

// SetCACert sets a custom CA certificate file for TLS verification.
func SetCACert(path string) {
	caCertPath = path
}

// tlsWarnOnce keeps the disabled-verification warning to one per invocation
// (clients are constructed more than once per command).
var tlsWarnOnce sync.Once

// buildTLSConfig merges session TLS overrides with the context config.
func buildTLSConfig() (*tls.Config, error) {
	insecure := insecureSkipVerify
	ca := caCertPath

	if cc, err := config.GetContextConfig(config.GetCurrentContext()); err == nil && cc != nil {
		insecure = insecure || cc.InsecureSkipVerify
		if ca == "" {
			ca = cc.CACert
		}
	}

	// Config-driven insecure_skip_verify would otherwise disable verification
	// silently forever — make it visible whatever the source.
	if insecure {
		tlsWarnOnce.Do(func() {
			style.Warning("TLS certificate verification is DISABLED (--insecure or server.insecure_skip_verify) — connections can be intercepted.")
		})
	}

	if !insecure && ca == "" {
		return nil, nil // default verification
	}

	tlsCfg := &tls.Config{}
	if insecure {
		tlsCfg.InsecureSkipVerify = true
	}
	if ca != "" {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in %s", ca)
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

// newHTTPClient builds an *http.Client with the session timeout and TLS config.
func newHTTPClient() (*http.Client, error) {
	tlsCfg, err := buildTLSConfig()
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	return &http.Client{
		Timeout:   requestTimeout,
		Transport: transport,
	}, nil
}

// WarnIfPlaintext warns when credentials are about to travel over plain HTTP
// to a host that is not obviously local.
func WarnIfPlaintext(scheme, host string) {
	if scheme != "http" {
		return
	}
	switch host {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return
	}
	style.Warning(fmt.Sprintf("Connecting to %s over plain HTTP: credentials and secrets are sent unencrypted. Use https if possible.", host))
}

// NewAuthenticatedClient initializes the SDK client with bearer token auth.
// Auth precedence: SEMCTL_API_TOKEN / configured api_token → cached token →
// username/password login (which caches a fresh token).
func NewAuthenticatedClient() (*apiclient.Semapi, error) {
	// 1. Direct API token from env or config
	if token := config.GetAPIToken(); token != "" {
		return newClientWithToken(token, false)
	}

	// 2. Cached token; allow transparent re-login on 401 if creds available
	if token, err := loadCachedToken(); err == nil && token != "" {
		return newClientWithToken(token, true)
	}

	// 3. Authenticate with username/password
	username := config.GetUsername()
	password := config.GetPassword()
	if username == "" || password == "" {
		return nil, fmt.Errorf("%w: run 'semctl login', set SEMCTL_API_TOKEN, or set SEMCTL_AUTH_USERNAME and SEMCTL_AUTH_PASSWORD", ErrNoCredentials)
	}

	// Stored credentials must not follow a --server/SEMCTL_SERVER redirect
	// away from the context's configured server.
	if config.ServerRedirected() {
		return nil, fmt.Errorf("refusing to send context %q credentials to a --server/SEMCTL_SERVER override pointing at a different server: run 'semctl login' against it explicitly, or use SEMCTL_API_TOKEN", config.GetCurrentContext())
	}

	serverURL, err := config.GetServerURL()
	if err != nil {
		return nil, err
	}

	// Environment credentials follow whatever server the config resolves to;
	// make that visible when the config came from the working directory.
	if config.LoadedFromCWD() && config.CredentialsFromEnv() {
		style.Warning(fmt.Sprintf("Logging in to %s (from %s) with environment credentials — verify this server is yours.", serverURL, config.ConfigFilePath()))
	}

	token, err := LoginAndCreateToken(serverURL, username, password)
	if err != nil {
		return nil, err
	}
	_ = SaveTokenCache(token)

	return newClientWithToken(token, false)
}

// newClientWithToken creates a go-swagger client authenticated with a bearer
// token. When allowReauth is true and username/password credentials are
// available, a 401 response triggers one transparent re-login and retry.
func newClientWithToken(token string, allowReauth bool) (*apiclient.Semapi, error) {
	host, port, scheme, err := config.ResolveServer()
	if err != nil {
		return nil, err
	}
	WarnIfPlaintext(scheme, host)

	httpClient, err := newHTTPClient()
	if err != nil {
		return nil, err
	}

	// No re-auth when a server override redirects away from the context's
	// configured server — a 401 from the redirect target must not trigger a
	// password login against it.
	if allowReauth && !config.ServerRedirected() && config.GetUsername() != "" && config.GetPassword() != "" {
		httpClient.Transport = &reauthTransport{base: httpClient.Transport}
	}

	transport := httptransport.NewWithClient(joinHostPort(host, port), "/api", []string{scheme}, httpClient)
	// go-openapi dumps full requests — Authorization header and secret
	// bodies included — whenever the generic DEBUG/SWAGGER_DEBUG env vars
	// are set. Require our own variable instead.
	transport.Debug = os.Getenv("SEMCTL_DEBUG") != ""
	transport.DefaultAuthentication = httptransport.BearerToken(token)
	// Every API error passes through TranslateAPIError on its way out.
	return apiclient.New(&translatingTransport{inner: transport}, strfmt.Default), nil
}

func joinHostPort(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

// reauthTransport retries a request once with a fresh token after a 401,
// covering expired or revoked cached tokens.
type reauthTransport struct {
	base http.RoundTripper

	mu    sync.Mutex
	tried bool
	token string
}

func (t *reauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	t.mu.Unlock()

	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	// Only bodyless requests or requests with GetBody can be safely replayed.
	if req.Body != nil && req.GetBody == nil {
		return resp, nil
	}

	t.mu.Lock()
	if t.tried {
		t.mu.Unlock()
		return resp, nil
	}
	t.tried = true
	t.mu.Unlock()

	serverURL, cfgErr := config.GetServerURL()
	if cfgErr != nil {
		return resp, nil
	}
	newToken, loginErr := LoginAndCreateToken(serverURL, config.GetUsername(), config.GetPassword())
	if loginErr != nil {
		return resp, nil // surface the original 401
	}
	_ = SaveTokenCache(newToken)
	style.Info("Cached API token was rejected; re-authenticated with username/password.")

	t.mu.Lock()
	t.token = newToken
	t.mu.Unlock()

	resp.Body.Close()
	retry := req.Clone(req.Context())
	if req.GetBody != nil {
		body, bodyErr := req.GetBody()
		if bodyErr != nil {
			return resp, nil
		}
		retry.Body = body
	}
	retry.Header.Set("Authorization", "Bearer "+newToken)
	return t.base.RoundTrip(retry)
}

// LoginAndCreateToken performs the Semaphore cookie-based login and creates an API token.
// Flow: POST /auth/login → session cookie → POST /user/tokens → API token ID
func LoginAndCreateToken(serverURL, username, password string) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create cookie jar: %w", err)
	}
	httpClient, err := newHTTPClient()
	if err != nil {
		return "", err
	}
	httpClient.Jar = jar

	// Step 1: Login to get session cookie
	loginBody, _ := json.Marshal(map[string]string{
		"auth":     username,
		"password": password,
	})

	loginResp, err := httpClient.Post(serverURL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		// Raw net/http path — translate here since it bypasses the
		// swagger transport hook.
		return "", fmt.Errorf("login request failed: %w", TranslateAPIError(err))
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusNoContent && loginResp.StatusCode != http.StatusOK {
		if loginResp.StatusCode == http.StatusUnauthorized || loginResp.StatusCode == http.StatusForbidden {
			return "", fmt.Errorf("%w (status %d)", ErrAuthFailed, loginResp.StatusCode)
		}
		return "", fmt.Errorf("login failed (status %d)", loginResp.StatusCode)
	}

	// Step 2: Create API token using the session cookie
	tokenResp, err := httpClient.Post(serverURL+"/user/tokens", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("token creation request failed: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusCreated && tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token creation failed (status %d)", tokenResp.StatusCode)
	}

	var apiToken struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&apiToken); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if apiToken.ID == "" {
		return "", fmt.Errorf("empty token returned from server")
	}

	return apiToken.ID, nil
}

// RevokeToken expires an API token server-side (DELETE /user/tokens/{id}).
func RevokeToken(serverURL, token string) error {
	httpClient, err := newHTTPClient()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, serverURL+"/user/tokens/"+token, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token revocation failed (status %d)", resp.StatusCode)
	}
	return nil
}

// getCacheDir returns the cache directory path for semctl. Failing to
// resolve the home directory is an error — falling back to a relative path
// would drop the token file into whatever directory semctl runs from.
func getCacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "semctl"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine cache directory: %w", err)
	}
	return filepath.Join(home, ".cache", "semctl"), nil
}

// TokenCachePath returns the path to the cached token file for the current context.
func TokenCachePath() (string, error) {
	return TokenCachePathForContext(config.GetCurrentContext())
}

// TokenCachePathForContext returns the path to the cached token file for a
// specific context. The name is normalized so every caller — login writing,
// logout/delete removing, rename moving — lands on the same file regardless
// of how the user cased the context name.
func TokenCachePathForContext(name string) (string, error) {
	dir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tokens", config.NormalizeContextName(name)+".json"), nil
}

// LoadCachedToken reads and validates the cached token for the current context.
func LoadCachedToken() (string, error) {
	return loadCachedToken()
}

// loadCachedToken reads the cached token and verifies it was issued by the
// server the session currently resolves to. A mismatch (context redefined,
// server override, or legacy cache without server binding) is a cache miss.
func loadCachedToken() (string, error) {
	cachePath, err := TokenCachePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", err
	}

	var cache tokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return "", err
	}

	if cache.Token == "" {
		return "", fmt.Errorf("empty token in cache")
	}
	if cache.Server == "" {
		return "", fmt.Errorf("cached token has no server binding (pre-upgrade cache): run 'semctl login' to refresh it")
	}

	current, err := resolvedServerID()
	if err != nil {
		return "", err
	}
	if cache.Server != current {
		return "", fmt.Errorf("cached token was issued by %s, not %s", cache.Server, current)
	}

	return cache.Token, nil
}

// LoadCachedTokenForContext reads the cached token for a specific context.
// No server check: callers (logout revocation) send it only to the context's
// own configured server.
func LoadCachedTokenForContext(name string) (string, error) {
	if err := config.ValidateContextName(name); err != nil {
		return "", err
	}
	cachePath, err := TokenCachePathForContext(name)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", err
	}
	var cache tokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return "", err
	}
	if cache.Token == "" {
		return "", fmt.Errorf("empty token in cache")
	}
	return cache.Token, nil
}

// SaveTokenCache writes the token, bound to the currently resolved server,
// to the cache file for the current context.
func SaveTokenCache(token string) error {
	server, err := resolvedServerID()
	if err != nil {
		return err
	}
	return SaveTokenCacheForContext(config.GetCurrentContext(), server, token)
}

// SaveTokenCacheForContext writes the token to the cache file for a specific
// context, recording the server it was issued by (use ServerID to build it).
func SaveTokenCacheForContext(name, server, token string) error {
	if err := config.ValidateContextName(name); err != nil {
		return err
	}
	cachePath, err := TokenCachePathForContext(name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	cache := tokenCache{Token: token, Server: server}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}
