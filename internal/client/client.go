package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ramanavelineni/semctl/internal/config"
	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
)

// tokenCache holds a cached API token.
type tokenCache struct {
	Token string `json:"token"`
}

// NewAuthenticatedClient initializes the SDK client with bearer token auth.
func NewAuthenticatedClient() (*apiclient.Semapi, error) {
	// 1. Try direct API token from config
	if token := config.GetAPIToken(); token != "" {
		return newClientWithToken(token), nil
	}

	// 2. Try cached token
	token, err := loadCachedToken()
	if err == nil && token != "" {
		return newClientWithToken(token), nil
	}

	// 3. Authenticate with username/password
	username := config.GetUsername()
	password := config.GetPassword()
	if username == "" || password == "" {
		return nil, fmt.Errorf("auth.username and auth.password must be configured (via config file or SEMCTL_AUTH_USERNAME / SEMCTL_AUTH_PASSWORD env vars)")
	}

	token, err = LoginAndCreateToken(config.GetServerURL(), username, password)
	if err != nil {
		return nil, err
	}

	// Cache the token
	_ = SaveTokenCache(token)

	return newClientWithToken(token), nil
}

// newClientWithToken creates a go-swagger client authenticated with a bearer token.
func newClientWithToken(token string) *apiclient.Semapi {
	cc, _ := config.GetContextConfig(config.GetCurrentContext())
	host := "localhost:3000"
	scheme := "http"
	if cc != nil {
		host = fmt.Sprintf("%s:%d", cc.ServerHost, cc.ServerPort)
		if cc.ServerScheme != "" {
			scheme = cc.ServerScheme
		}
	}

	transport := httptransport.New(host, "/api", []string{scheme})
	transport.DefaultAuthentication = httptransport.BearerToken(token)
	return apiclient.New(transport, strfmt.Default)
}

// LoginAndCreateToken performs the Semaphore cookie-based login and creates an API token.
// Flow: POST /auth/login → session cookie → POST /user/tokens → API token ID
func LoginAndCreateToken(serverURL, username, password string) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create cookie jar: %w", err)
	}
	httpClient := &http.Client{Jar: jar}

	// Step 1: Login to get session cookie
	loginBody, _ := json.Marshal(map[string]string{
		"auth":     username,
		"password": password,
	})

	loginResp, err := httpClient.Post(serverURL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusNoContent && loginResp.StatusCode != http.StatusOK {
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

// getCacheDir returns the cache directory path for semctl.
func getCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "semctl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "semctl")
}

// TokenCachePath returns the path to the cached token file for the current context.
func TokenCachePath() string {
	return TokenCachePathForContext(config.GetCurrentContext())
}

// TokenCachePathForContext returns the path to the cached token file for a specific context.
func TokenCachePathForContext(name string) string {
	return filepath.Join(getCacheDir(), "tokens", name+".json")
}

// loadCachedToken reads and validates the cached token.
func loadCachedToken() (string, error) {
	cachePath := TokenCachePath()
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

// SaveTokenCache writes the token to the cache file for the current context.
func SaveTokenCache(token string) error {
	return SaveTokenCacheForContext(config.GetCurrentContext(), token)
}

// SaveTokenCacheForContext writes the token to the cache file for a specific context.
func SaveTokenCacheForContext(name, token string) error {
	cachePath := TokenCachePathForContext(name)
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	cache := tokenCache{Token: token}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}
