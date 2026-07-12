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

	"github.com/spf13/viper"
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

// serverOverride holds a session-scoped host:port override (from --server).
var serverOverride string

// SetServerOverride overrides the server host:port for the current session.
func SetServerOverride(hostPort string) {
	serverOverride = hostPort
}

// contextNameRE restricts context names to filename- and Viper-safe
// characters: no path separators or ".." (token cache paths embed the name)
// and no dots (Viper would interpret them as nested config keys).
var contextNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ValidateContextName rejects context names that could escape the token-cache
// directory or address foreign Viper keys.
func ValidateContextName(name string) error {
	if !contextNameRE.MatchString(name) {
		return fmt.Errorf("invalid context name %q: use letters, digits, '-' or '_', starting with a letter or digit (max 64 chars)", name)
	}
	return nil
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
	if override == "" || v == nil {
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

var v *viper.Viper

// loadedFromCWD records whether the config file was picked up from the
// current working directory (as opposed to the home config dir or --config).
var loadedFromCWD bool

// LoadedFromCWD reports whether the loaded config file came from the current
// working directory.
func LoadedFromCWD() bool {
	return loadedFromCWD
}

// Load initializes Viper and loads the config file.
func Load(cfgFile string) error {
	loadedFromCWD = false
	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetEnvPrefix("SEMCTL")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// Search paths (do NOT search "." — Viper would match the binary itself
		// since SetConfigName("semctl") matches any file named "semctl")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".config", "semctl"))
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			v.AddConfigPath(filepath.Join(xdg, "semctl"))
		}

		// For local project config, use the explicit filename with extension
		if _, err := os.Stat("semctl.yaml"); err == nil {
			v.SetConfigFile("semctl.yaml")
			loadedFromCWD = true
		} else if _, err := os.Stat(".semctl.yaml"); err == nil {
			v.SetConfigFile(".semctl.yaml")
			loadedFromCWD = true
		}
	}

	v.SetDefault("current_context", "default")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // No config file is OK
		}
		return err
	}

	// A malicious or corrupt config could set current_context to a path
	// like "../../x" that escapes the token-cache directory downstream.
	if name := v.GetString("current_context"); name != "" {
		if err := ValidateContextName(name); err != nil {
			return fmt.Errorf("config %s: current_context: %w", v.ConfigFileUsed(), err)
		}
	}

	return nil
}

// ConfigFilePath returns the path of the loaded config file.
func ConfigFilePath() string {
	if v == nil {
		return ""
	}
	return v.ConfigFileUsed()
}

// GetCurrentContext returns the active context name.
func GetCurrentContext() string {
	if v == nil {
		return "default"
	}
	return v.GetString("current_context")
}

// SetCurrentContext sets the active context name.
func SetCurrentContext(name string) error {
	return saveConfigKey("current_context", name)
}

// ApplyContext overrides the active context for the current session.
func ApplyContext(name string) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	contexts := ListContexts()
	for _, c := range contexts {
		if c == name {
			v.Set("current_context", name)
			return nil
		}
	}
	return fmt.Errorf("context %q not found", name)
}

// ListContexts returns all context names sorted alphabetically.
func ListContexts() []string {
	if v == nil {
		return nil
	}
	ctxMap := v.GetStringMap("contexts")
	names := make([]string, 0, len(ctxMap))
	for name := range ctxMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetContextConfig returns the configuration for a named context.
func GetContextConfig(name string) (*ContextConfig, error) {
	if v == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	prefix := fmt.Sprintf("contexts.%s", name)
	if !v.IsSet(prefix) {
		return nil, fmt.Errorf("context %q not found", name)
	}

	return &ContextConfig{
		ServerHost:         v.GetString(prefix + ".server.host"),
		ServerPort:         v.GetInt(prefix + ".server.port"),
		ServerScheme:       v.GetString(prefix + ".server.scheme"),
		InsecureSkipVerify: v.GetBool(prefix + ".server.insecure_skip_verify"),
		CACert:             v.GetString(prefix + ".server.ca_cert"),
		AuthUsername:       v.GetString(prefix + ".auth.username"),
		AuthPassword:       v.GetString(prefix + ".auth.password"),
		AuthAPIToken:       v.GetString(prefix + ".auth.api_token"),
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
	if v != nil {
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

// GetUsername returns the auth username: SEMCTL_AUTH_USERNAME env var, or the
// current context's config value.
func GetUsername() string {
	if u := os.Getenv("SEMCTL_AUTH_USERNAME"); u != "" {
		return u
	}
	if v == nil {
		return ""
	}
	return v.GetString(fmt.Sprintf("contexts.%s.auth.username", GetCurrentContext()))
}

// GetPassword returns the auth password: SEMCTL_AUTH_PASSWORD env var, or the
// current context's config value.
func GetPassword() string {
	if p := os.Getenv("SEMCTL_AUTH_PASSWORD"); p != "" {
		return p
	}
	if v == nil {
		return ""
	}
	return v.GetString(fmt.Sprintf("contexts.%s.auth.password", GetCurrentContext()))
}

// GetAPIToken returns the auth API token: SEMCTL_API_TOKEN env var, or the
// current context's config value.
func GetAPIToken() string {
	if t := os.Getenv("SEMCTL_API_TOKEN"); t != "" {
		return t
	}
	if v == nil {
		return ""
	}
	return v.GetString(fmt.Sprintf("contexts.%s.auth.api_token", GetCurrentContext()))
}

// GetDefaultProjectID returns the default project ID from config.
func GetDefaultProjectID() int {
	if v == nil {
		return 0
	}
	return v.GetInt("defaults.project_id")
}

// GetOutputFormat returns the configured output format.
func GetOutputFormat() string {
	if v == nil {
		return "table"
	}
	return v.GetString("output.format")
}

// SaveContext saves or updates a context in the config file.
func SaveContext(name string, serverData, authData map[string]interface{}) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	cfgPath := resolveConfigPath()

	data := make(map[string]interface{})

	// Read existing config
	if raw, err := os.ReadFile(cfgPath); err == nil {
		_ = yaml.Unmarshal(raw, &data)
	}

	// Ensure contexts map exists
	contexts, ok := data["contexts"].(map[string]interface{})
	if !ok {
		contexts = make(map[string]interface{})
	}

	// Build context entry
	ctxEntry := map[string]interface{}{
		"server": serverData,
		"auth":   authData,
	}
	contexts[name] = ctxEntry
	data["contexts"] = contexts

	return writeConfigFile(cfgPath, data)
}

// DeleteContext removes a context from the config file.
func DeleteContext(name string) error {
	if err := ValidateContextName(name); err != nil {
		return err
	}
	cfgPath := resolveConfigPath()

	data := make(map[string]interface{})
	if raw, err := os.ReadFile(cfgPath); err == nil {
		_ = yaml.Unmarshal(raw, &data)
	}

	contexts, ok := data["contexts"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("context %q not found", name)
	}

	if _, exists := contexts[name]; !exists {
		return fmt.Errorf("context %q not found", name)
	}

	delete(contexts, name)
	data["contexts"] = contexts

	// If we deleted the current context, switch to another or clear
	if cur, ok := data["current_context"].(string); ok && cur == name {
		if len(contexts) > 0 {
			for k := range contexts {
				data["current_context"] = k
				break
			}
		} else {
			data["current_context"] = "default"
		}
	}

	return writeConfigFile(cfgPath, data)
}

// RenameContext renames a context in the config file.
func RenameContext(oldName, newName string) error {
	if err := ValidateContextName(oldName); err != nil {
		return err
	}
	if err := ValidateContextName(newName); err != nil {
		return err
	}
	cfgPath := resolveConfigPath()

	data := make(map[string]interface{})
	if raw, err := os.ReadFile(cfgPath); err == nil {
		_ = yaml.Unmarshal(raw, &data)
	}

	contexts, ok := data["contexts"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("context %q not found", oldName)
	}

	entry, exists := contexts[oldName]
	if !exists {
		return fmt.Errorf("context %q not found", oldName)
	}

	if _, exists := contexts[newName]; exists {
		return fmt.Errorf("context %q already exists", newName)
	}

	contexts[newName] = entry
	delete(contexts, oldName)
	data["contexts"] = contexts

	// Update current_context if it was the renamed one
	if cur, ok := data["current_context"].(string); ok && cur == oldName {
		data["current_context"] = newName
	}

	return writeConfigFile(cfgPath, data)
}

// RemoveAuthConfig removes auth credentials from the current context.
func RemoveAuthConfig() error {
	cfgPath := resolveConfigPath()

	data := make(map[string]interface{})
	if raw, err := os.ReadFile(cfgPath); err == nil {
		_ = yaml.Unmarshal(raw, &data)
	}

	ctx := GetCurrentContext()
	contexts, ok := data["contexts"].(map[string]interface{})
	if !ok {
		return nil
	}

	ctxData, ok := contexts[ctx].(map[string]interface{})
	if !ok {
		return nil
	}

	delete(ctxData, "auth")
	contexts[ctx] = ctxData
	data["contexts"] = contexts

	return writeConfigFile(cfgPath, data)
}

// resolveConfigPath determines where to write the config file.
func resolveConfigPath() string {
	if v != nil && v.ConfigFileUsed() != "" {
		return v.ConfigFileUsed()
	}

	// Default location
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "semctl", "config.yaml")
	}
	return "semctl.yaml"
}

// saveConfigKey updates a single key in the config file.
func saveConfigKey(key string, value interface{}) error {
	cfgPath := resolveConfigPath()

	data := make(map[string]interface{})
	if raw, err := os.ReadFile(cfgPath); err == nil {
		_ = yaml.Unmarshal(raw, &data)
	}

	data[key] = value

	return writeConfigFile(cfgPath, data)
}

// writeConfigFile writes the config data to the file, creating directories as needed.
func writeConfigFile(path string, data map[string]interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Reload viper
	if v != nil {
		v.SetConfigFile(path)
		_ = v.ReadInConfig()
	}

	return nil
}
