package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ContextConfig holds the configuration for a single server context.
type ContextConfig struct {
	ServerHost   string
	ServerPort   int
	ServerScheme string
	AuthUsername  string
	AuthPassword string
	AuthAPIToken string
}

var v *viper.Viper

// Load initializes Viper and loads the config file.
func Load(cfgFile string) error {
	v = viper.New()
	v.SetConfigName("semctl")
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
		} else if _, err := os.Stat(".semctl.yaml"); err == nil {
			v.SetConfigFile(".semctl.yaml")
		}
	}

	v.SetDefault("current_context", "default")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // No config file is OK
		}
		return err
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
		ServerHost:   v.GetString(prefix + ".server.host"),
		ServerPort:   v.GetInt(prefix + ".server.port"),
		ServerScheme: v.GetString(prefix + ".server.scheme"),
		AuthUsername:  v.GetString(prefix + ".auth.username"),
		AuthPassword: v.GetString(prefix + ".auth.password"),
		AuthAPIToken: v.GetString(prefix + ".auth.api_token"),
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
	return fmt.Sprintf("%s://%s:%d", scheme, cc.ServerHost, cc.ServerPort)
}

// GetServerURL returns the full API URL for the current context.
func GetServerURL() string {
	ctx := GetCurrentContext()
	cc, err := GetContextConfig(ctx)
	if err != nil {
		return ""
	}
	scheme := cc.ServerScheme
	if scheme == "" {
		scheme = "http"
	}
	port := cc.ServerPort
	if port == 0 {
		port = 3000
	}
	return fmt.Sprintf("%s://%s:%d/api", scheme, cc.ServerHost, port)
}

// GetUsername returns the auth username for the current context.
func GetUsername() string {
	ctx := GetCurrentContext()
	return v.GetString(fmt.Sprintf("contexts.%s.auth.username", ctx))
}

// GetPassword returns the auth password for the current context.
func GetPassword() string {
	ctx := GetCurrentContext()
	return v.GetString(fmt.Sprintf("contexts.%s.auth.password", ctx))
}

// GetAPIToken returns the auth API token for the current context.
func GetAPIToken() string {
	ctx := GetCurrentContext()
	return v.GetString(fmt.Sprintf("contexts.%s.auth.api_token", ctx))
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
