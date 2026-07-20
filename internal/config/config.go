package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// DefaultPort is the port assumed when a server string has no port.
const DefaultPort = 3000

// ContextConfig holds the configuration for a single server context.
type ContextConfig struct {
	ServerHost         string
	ServerPort         int
	ServerScheme       string
	InsecureSkipVerify bool
	CACert             string
	AuthUsername       string
	AuthPassword       string
	AuthAPIToken       string
}

// fileConfig mirrors the on-disk YAML for reads. Writes go through a generic
// map round-trip instead (see updateConfigFile) so keys this struct doesn't
// know about survive a save.
type fileConfig struct {
	CurrentContext string                   `yaml:"current_context"`
	Contexts       map[string]*contextEntry `yaml:"contexts"`
	Defaults       struct {
		ProjectID int `yaml:"project_id"`
	} `yaml:"defaults"`
	Output struct {
		Format string `yaml:"format"`
	} `yaml:"output"`
}

type contextEntry struct {
	Server struct {
		Host               string `yaml:"host"`
		Port               int    `yaml:"port"`
		Scheme             string `yaml:"scheme"`
		InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
		CACert             string `yaml:"ca_cert"`
	} `yaml:"server"`
	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		APIToken string `yaml:"api_token"`
	} `yaml:"auth"`
}

// Package state. cfg is nil until Load runs; sessionContext is a --context
// override that must not be persisted.
var (
	cfg            *fileConfig
	configFileUsed string
	loadedFromCWD  bool
	sessionContext string
	serverOverride string
)

// SetServerOverride overrides the server host:port for the current session.
func SetServerOverride(hostPort string) {
	serverOverride = hostPort
}

// contextNameRE restricts context names to filename-safe characters: no path
// separators or ".." (token cache paths embed the name) and no dots.
var contextNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ValidateContextName rejects context names that could escape the token-cache
// directory. Names are case-insensitive: they are lowercased at every config
// boundary so "Prod" and "prod" are the same context.
func ValidateContextName(name string) error {
	if !contextNameRE.MatchString(name) {
		return fmt.Errorf("invalid context name %q: use letters, digits, '-' or '_', starting with a letter or digit (max 64 chars)", name)
	}
	return nil
}

// NormalizeContextName is the single place context names are folded (context
// names are case-insensitive). Applied on read AND write, and by the token
// cache, so a config hand-edited with mixed case still resolves.
func NormalizeContextName(name string) string {
	return strings.ToLower(name)
}

// ServerRedirected reports whether a session server override (--server or
// SEMCTL_SERVER) is in effect and points somewhere other than the current
// context's configured server. Stored credentials must not follow such a
// redirect — the override names a server the saved login never consented to.
func ServerRedirected() bool {
	override := serverOverride
	if override == "" {
		override = os.Getenv("SEMCTL_SERVER")
	}
	if override == "" || cfg == nil {
		return false
	}
	cc, err := GetContextConfig(GetCurrentContext())
	if err != nil || cc == nil || cc.ServerHost == "" {
		return false // configless usage: the override IS the server
	}
	oHost, oPort, err := ParseHostPort(override)
	if err != nil {
		return false // ResolveServer will surface the parse error
	}
	ctxPort := cc.ServerPort
	if ctxPort == 0 {
		ctxPort = DefaultPort
	}
	return !strings.EqualFold(oHost, cc.ServerHost) || oPort != ctxPort
}

// CredentialsFromEnv reports whether the username/password credentials come
// from environment variables rather than the loaded config file.
func CredentialsFromEnv() bool {
	return os.Getenv("SEMCTL_AUTH_USERNAME") != "" || os.Getenv("SEMCTL_AUTH_PASSWORD") != ""
}

// LoadedFromCWD reports whether the loaded config file came from the current
// working directory.
func LoadedFromCWD() bool {
	return loadedFromCWD
}

// Load reads the config file. Search order: explicit path, ./semctl.yaml,
// ./.semctl.yaml, ~/.config/semctl/config.{yaml,yml}, then
// $XDG_CONFIG_HOME/semctl/config.{yaml,yml}. A missing file is fine; a
// malformed one is an error.
func Load(cfgFile string) error {
	cfg = &fileConfig{}
	configFileUsed = ""
	loadedFromCWD = false
	sessionContext = ""

	path := cfgFile
	if path == "" {
		if _, err := os.Stat("semctl.yaml"); err == nil {
			path = "semctl.yaml"
			loadedFromCWD = true
		} else if _, err := os.Stat(".semctl.yaml"); err == nil {
			path = ".semctl.yaml"
			loadedFromCWD = true
		} else {
			for _, candidate := range searchPaths() {
				if _, err := os.Stat(candidate); err == nil {
					path = candidate
					break
				}
			}
		}
		if path == "" {
			return nil // no config file is OK
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		// An explicit --config target is remembered even when it doesn't
		// exist yet: login ignores this error and then writes the file there.
		if cfgFile != "" {
			configFileUsed = cfgFile
		}
		return err
	}
	// yaml.v3 also rejects duplicate mapping keys here — previously Viper
	// merged them unpredictably (the "two prod: blocks" failure mode).
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	configFileUsed = path

	// Fold context keys to lowercase; a config carrying both "Prod" and
	// "prod" is ambiguous rather than silently last-one-wins.
	if len(cfg.Contexts) > 0 {
		folded := make(map[string]*contextEntry, len(cfg.Contexts))
		for name, entry := range cfg.Contexts {
			lower := NormalizeContextName(name)
			if _, dup := folded[lower]; dup {
				return fmt.Errorf("config %s: contexts %q defined more than once (context names are case-insensitive)", path, lower)
			}
			folded[lower] = entry
		}
		cfg.Contexts = folded
	}

	// A malicious or corrupt config could set current_context to a path
	// like "../../x" that escapes the token-cache directory downstream.
	if cfg.CurrentContext != "" {
		if err := ValidateContextName(cfg.CurrentContext); err != nil {
			return fmt.Errorf("config %s: current_context: %w", path, err)
		}
		cfg.CurrentContext = NormalizeContextName(cfg.CurrentContext)
	}

	return nil
}

// searchPaths lists the home-directory config candidates in precedence order.
func searchPaths() []string {
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "semctl", "config.yaml"),
			filepath.Join(home, ".config", "semctl", "config.yml"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths,
			filepath.Join(xdg, "semctl", "config.yaml"),
			filepath.Join(xdg, "semctl", "config.yml"))
	}
	return paths
}

// ConfigFilePath returns the path of the loaded config file.
func ConfigFilePath() string {
	return configFileUsed
}

// GetCurrentContext returns the active context name.
func GetCurrentContext() string {
	if sessionContext != "" {
		return sessionContext
	}
	if cfg == nil || cfg.CurrentContext == "" {
		return "default"
	}
	return cfg.CurrentContext
}

// SetCurrentContext sets the active context name in the config file.
func SetCurrentContext(name string) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	return updateConfigFile(func(data map[string]interface{}) error {
		data["current_context"] = NormalizeContextName(name)
		return nil
	})
}

// ApplyContext overrides the active context for the current session.
func ApplyContext(name string) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	name = NormalizeContextName(name)
	if cfg != nil {
		if _, ok := cfg.Contexts[name]; ok {
			sessionContext = name
			return nil
		}
	}
	return fmt.Errorf("context %q not found", name)
}

// ListContexts returns all context names sorted alphabetically.
func ListContexts() []string {
	if cfg == nil {
		return nil
	}
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetContextConfig returns the configuration for a named context.
func GetContextConfig(name string) (*ContextConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	entry, ok := cfg.Contexts[NormalizeContextName(name)]
	if !ok || entry == nil {
		return nil, fmt.Errorf("context %q not found", name)
	}
	return &ContextConfig{
		ServerHost:         entry.Server.Host,
		ServerPort:         entry.Server.Port,
		ServerScheme:       entry.Server.Scheme,
		InsecureSkipVerify: entry.Server.InsecureSkipVerify,
		CACert:             entry.Server.CACert,
		AuthUsername:       entry.Auth.Username,
		AuthPassword:       entry.Auth.Password,
		AuthAPIToken:       entry.Auth.APIToken,
	}, nil
}

// GetContextServerDisplay returns a display string like "http://host:port" for the context.
func GetContextServerDisplay(name string) string {
	cc, err := GetContextConfig(name)
	if err != nil {
		return ""
	}
	scheme := cc.ServerScheme
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(cc.ServerHost, strconv.Itoa(cc.ServerPort)))
}

// ParseHostPort splits a "host[:port]" string, defaulting the port to
// DefaultPort when omitted. Invalid ports are an error (not silently ignored).
// Handles IPv6 literals like "[::1]:3000" and bare "::1".
func ParseHostPort(s string) (string, int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, fmt.Errorf("server is required")
	}

	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		var addrErr *net.AddrError
		if errors.As(err, &addrErr) && strings.Contains(addrErr.Err, "missing port") {
			return strings.Trim(s, "[]"), DefaultPort, nil
		}
		// A bare IPv6 literal like "::1" trips "too many colons".
		if ip := net.ParseIP(s); ip != nil {
			return s, DefaultPort, nil
		}
		return "", 0, fmt.Errorf("invalid server %q: %w", s, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port %q in server %q", portStr, s)
	}
	return host, port, nil
}

// ResolveServer returns the effective server host, port, and scheme for the
// current context. Precedence: --server flag > SEMCTL_SERVER env var > context
// config. Scheme precedence: SEMCTL_SCHEME env var > context config > "http".
func ResolveServer() (host string, port int, scheme string, err error) {
	var cc *ContextConfig
	if cfg != nil {
		cc, _ = GetContextConfig(GetCurrentContext())
	}

	switch {
	case serverOverride != "":
		host, port, err = ParseHostPort(serverOverride)
		if err != nil {
			return "", 0, "", fmt.Errorf("--server: %w", err)
		}
	case os.Getenv("SEMCTL_SERVER") != "":
		host, port, err = ParseHostPort(os.Getenv("SEMCTL_SERVER"))
		if err != nil {
			return "", 0, "", fmt.Errorf("SEMCTL_SERVER: %w", err)
		}
	case cc != nil && cc.ServerHost != "":
		host = cc.ServerHost
		port = cc.ServerPort
		if port == 0 {
			port = DefaultPort
		}
	default:
		return "", 0, "", fmt.Errorf("no server configured for context %q: run 'semctl login', set SEMCTL_SERVER, or use --server", GetCurrentContext())
	}

	scheme = os.Getenv("SEMCTL_SCHEME")
	if scheme == "" && cc != nil {
		scheme = cc.ServerScheme
	}
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return "", 0, "", fmt.Errorf("invalid scheme %q (must be http or https)", scheme)
	}

	return host, port, scheme, nil
}

// GetServerURL returns the full API URL for the current context.
func GetServerURL() (string, error) {
	host, port, scheme, err := ResolveServer()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s/api", scheme, net.JoinHostPort(host, strconv.Itoa(port))), nil
}

// currentContextConfig returns the active context's config, or nil.
func currentContextConfig() *ContextConfig {
	if cfg == nil {
		return nil
	}
	cc, _ := GetContextConfig(GetCurrentContext())
	return cc
}

// GetUsername returns the auth username: SEMCTL_AUTH_USERNAME env var, or the
// current context's config value.
func GetUsername() string {
	if u := os.Getenv("SEMCTL_AUTH_USERNAME"); u != "" {
		return u
	}
	if cc := currentContextConfig(); cc != nil {
		return cc.AuthUsername
	}
	return ""
}

// GetPassword returns the auth password: SEMCTL_AUTH_PASSWORD env var, or the
// current context's config value.
func GetPassword() string {
	if p := os.Getenv("SEMCTL_AUTH_PASSWORD"); p != "" {
		return p
	}
	if cc := currentContextConfig(); cc != nil {
		return cc.AuthPassword
	}
	return ""
}

// GetAPIToken returns the auth API token: SEMCTL_API_TOKEN env var, or the
// current context's config value.
func GetAPIToken() string {
	if t := os.Getenv("SEMCTL_API_TOKEN"); t != "" {
		return t
	}
	if cc := currentContextConfig(); cc != nil {
		return cc.AuthAPIToken
	}
	return ""
}

// GetDefaultProjectID returns the default project ID from config.
func GetDefaultProjectID() int {
	if cfg == nil {
		return 0
	}
	return cfg.Defaults.ProjectID
}

// GetOutputFormat returns the configured output format.
func GetOutputFormat() string {
	if cfg == nil {
		return "table"
	}
	return cfg.Output.Format
}

// SaveContext saves or updates a context in the config file.
func SaveContext(name string, serverData, authData map[string]interface{}) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	return updateConfigFile(func(data map[string]interface{}) error {
		contexts, _ := data["contexts"].(map[string]interface{})
		if contexts == nil {
			contexts = make(map[string]interface{})
		}
		key := NormalizeContextName(name)
		// Replace any differently-cased spelling of the same context.
		if existing, ok := findContextKey(contexts, name); ok && existing != key {
			delete(contexts, existing)
		}
		contexts[key] = map[string]interface{}{
			"server": serverData,
			"auth":   authData,
		}
		data["contexts"] = contexts
		return nil
	})
}

// DeleteContext removes a context from the config file.
func DeleteContext(name string) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	return updateConfigFile(func(data map[string]interface{}) error {
		contexts, _ := data["contexts"].(map[string]interface{})
		key, ok := findContextKey(contexts, name)
		if !ok {
			return fmt.Errorf("context %q not found", name)
		}
		delete(contexts, key)
		data["contexts"] = contexts

		// If we deleted the current context, switch to another or clear
		if cur, ok := data["current_context"].(string); ok && NormalizeContextName(cur) == NormalizeContextName(name) {
			if len(contexts) > 0 {
				for k := range contexts {
					data["current_context"] = NormalizeContextName(k)
					break
				}
			} else {
				data["current_context"] = "default"
			}
		}
		return nil
	})
}

// RenameContext renames a context in the config file.
func RenameContext(oldName, newName string) error {
	if err := ValidateContextName(oldName); err != nil {
		return err
	}
	if err := ValidateContextName(newName); err != nil {
		return err
	}
	return updateConfigFile(func(data map[string]interface{}) error {
		contexts, _ := data["contexts"].(map[string]interface{})
		oldKey, ok := findContextKey(contexts, oldName)
		if !ok {
			return fmt.Errorf("context %q not found", oldName)
		}
		newKey := NormalizeContextName(newName)
		if existing, ok := findContextKey(contexts, newName); ok && existing != oldKey {
			return fmt.Errorf("context %q already exists", newName)
		}
		entry := contexts[oldKey]
		delete(contexts, oldKey)
		contexts[newKey] = entry
		data["contexts"] = contexts

		if cur, ok := data["current_context"].(string); ok && NormalizeContextName(cur) == NormalizeContextName(oldName) {
			data["current_context"] = newKey
		}
		return nil
	})
}

// RemoveAuthConfig removes auth credentials from the current context.
func RemoveAuthConfig() error {
	return updateConfigFile(func(data map[string]interface{}) error {
		contexts, _ := data["contexts"].(map[string]interface{})
		key, ok := findContextKey(contexts, GetCurrentContext())
		if !ok {
			return nil
		}
		ctxData, ok := contexts[key].(map[string]interface{})
		if !ok {
			return nil
		}
		delete(ctxData, "auth")
		contexts[key] = ctxData
		data["contexts"] = contexts
		return nil
	})
}

// findContextKey locates a context in the raw config map by case-insensitive
// name, returning the key as actually spelled in the file.
func findContextKey(contexts map[string]interface{}, name string) (string, bool) {
	want := NormalizeContextName(name)
	for k := range contexts {
		if NormalizeContextName(k) == want {
			return k, true
		}
	}
	return "", false
}

// resolveConfigPath determines where to write the config file.
func resolveConfigPath() string {
	if configFileUsed != "" {
		return configFileUsed
	}

	// Default location
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "semctl", "config.yaml")
	}
	return "semctl.yaml"
}

// configMu serializes in-process writers; the flock in withConfigLock
// serializes writers across processes.
var configMu sync.Mutex

// updateConfigFile runs a read-modify-write cycle on the config file under an
// advisory lock, so concurrent logins can't interleave and corrupt it. The
// raw YAML is round-tripped as a generic map: keys the fileConfig struct
// doesn't know about survive. An unreadable existing file aborts the write
// instead of silently starting from empty.
func updateConfigFile(mutate func(data map[string]interface{}) error) error {
	configMu.Lock()
	defer configMu.Unlock()
	cfgPath := resolveConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return withConfigLock(cfgPath, func() error {
		data := make(map[string]interface{})
		if raw, err := os.ReadFile(cfgPath); err == nil {
			if err := yaml.Unmarshal(raw, &data); err != nil {
				return fmt.Errorf("refusing to overwrite unparseable config %s: %w", cfgPath, err)
			}
			if data == nil {
				data = make(map[string]interface{})
			}
		}
		if err := mutate(data); err != nil {
			return err
		}
		return writeConfigFile(cfgPath, data)
	})
}

// writeConfigFile atomically replaces the config file (temp file + rename) so
// a crash or a concurrent reader can never observe a half-written config.
func writeConfigFile(path string, data map[string]interface{}) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename

	// 0600 like the token cache: the config can carry a password.
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write config: %w", err)
	}
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Refresh the in-memory view.
	prevCWD := loadedFromCWD
	prevSession := sessionContext
	if err := Load(path); err != nil {
		return err
	}
	loadedFromCWD = prevCWD
	sessionContext = prevSession
	return nil
}
