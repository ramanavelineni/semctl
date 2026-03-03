package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ApplyConfig represents the full declarative configuration file.
type ApplyConfig struct {
	Project       string           `json:"project" yaml:"project"`
	Keys          []KeyEntry       `json:"keys,omitempty" yaml:"keys,omitempty"`
	Environments  []EnvEntry       `json:"environments,omitempty" yaml:"environments,omitempty"`
	Repositories  []RepoEntry      `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Inventories   []InventoryEntry `json:"inventories,omitempty" yaml:"inventories,omitempty"`
	Templates     []TemplateEntry  `json:"templates,omitempty" yaml:"templates,omitempty"`
	Schedules     []ScheduleEntry  `json:"schedules,omitempty" yaml:"schedules,omitempty"`
}

// KeyEntry represents a key in the config file.
type KeyEntry struct {
	Name       string       `json:"name" yaml:"name"`
	Type       string       `json:"type,omitempty" yaml:"type,omitempty"`
	State      string       `json:"state,omitempty" yaml:"state,omitempty"`
	SSH        *SSHKeyData  `json:"ssh,omitempty" yaml:"ssh,omitempty"`
	LoginPassword *LoginPasswordData `json:"login_password,omitempty" yaml:"login_password,omitempty"`
}

// SSHKeyData holds SSH key fields.
type SSHKeyData struct {
	Login      string `json:"login,omitempty" yaml:"login,omitempty"`
	PrivateKey string `json:"private_key,omitempty" yaml:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty" yaml:"passphrase,omitempty"`
}

// LoginPasswordData holds login/password key fields.
type LoginPasswordData struct {
	Login    string `json:"login,omitempty" yaml:"login,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// EnvEntry represents an environment in the config file.
type EnvEntry struct {
	Name     string `json:"name" yaml:"name"`
	State    string `json:"state,omitempty" yaml:"state,omitempty"`
	JSON     string `json:"json,omitempty" yaml:"json,omitempty"`
	Env      string `json:"env,omitempty" yaml:"env,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// RepoEntry represents a repository in the config file.
type RepoEntry struct {
	Name      string `json:"name" yaml:"name"`
	State     string `json:"state,omitempty" yaml:"state,omitempty"`
	GitURL    string `json:"git_url,omitempty" yaml:"git_url,omitempty"`
	GitBranch string `json:"git_branch,omitempty" yaml:"git_branch,omitempty"`
	SSHKey    string `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"`       // name ref
	SSHKeyID  int64  `json:"ssh_key_id,omitempty" yaml:"ssh_key_id,omitempty"` // explicit ID
}

// InventoryEntry represents an inventory in the config file.
type InventoryEntry struct {
	Name         string `json:"name" yaml:"name"`
	State        string `json:"state,omitempty" yaml:"state,omitempty"`
	Type         string `json:"type,omitempty" yaml:"type,omitempty"`
	Inventory    string `json:"inventory,omitempty" yaml:"inventory,omitempty"`
	SSHKey       string `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"`
	SSHKeyID     int64  `json:"ssh_key_id,omitempty" yaml:"ssh_key_id,omitempty"`
	BecomeKey    string `json:"become_key,omitempty" yaml:"become_key,omitempty"`
	BecomeKeyID  int64  `json:"become_key_id,omitempty" yaml:"become_key_id,omitempty"`
	Repository   string `json:"repository,omitempty" yaml:"repository,omitempty"`
	RepositoryID int64  `json:"repository_id,omitempty" yaml:"repository_id,omitempty"`
}

// TemplateEntry represents a template in the config file.
type TemplateEntry struct {
	Name                    string `json:"name" yaml:"name"`
	State                   string `json:"state,omitempty" yaml:"state,omitempty"`
	Type                    string `json:"type,omitempty" yaml:"type,omitempty"`
	App                     string `json:"app,omitempty" yaml:"app,omitempty"`
	Playbook                string `json:"playbook,omitempty" yaml:"playbook,omitempty"`
	Description             string `json:"description,omitempty" yaml:"description,omitempty"`
	GitBranch               string `json:"git_branch,omitempty" yaml:"git_branch,omitempty"`
	Arguments               string `json:"arguments,omitempty" yaml:"arguments,omitempty"`
	StartVersion            string `json:"start_version,omitempty" yaml:"start_version,omitempty"`
	Autorun                 bool   `json:"autorun,omitempty" yaml:"autorun,omitempty"`
	SuppressSuccessAlerts   bool   `json:"suppress_success_alerts,omitempty" yaml:"suppress_success_alerts,omitempty"`
	AllowOverrideArgsInTask bool   `json:"allow_override_args_in_task,omitempty" yaml:"allow_override_args_in_task,omitempty"`
	Repository              string `json:"repository,omitempty" yaml:"repository,omitempty"`
	RepositoryID            int64  `json:"repository_id,omitempty" yaml:"repository_id,omitempty"`
	Environment             string `json:"environment,omitempty" yaml:"environment,omitempty"`
	EnvironmentID           int64  `json:"environment_id,omitempty" yaml:"environment_id,omitempty"`
	Inventory               string `json:"inventory,omitempty" yaml:"inventory,omitempty"`
	InventoryID             int64  `json:"inventory_id,omitempty" yaml:"inventory_id,omitempty"`
	BuildTemplate           string `json:"build_template,omitempty" yaml:"build_template,omitempty"`
	BuildTemplateID         int64  `json:"build_template_id,omitempty" yaml:"build_template_id,omitempty"`
	ViewID                  int64  `json:"view_id,omitempty" yaml:"view_id,omitempty"`
}

// ScheduleEntry represents a schedule in the config file.
type ScheduleEntry struct {
	Name       string `json:"name" yaml:"name"`
	CronFormat string `json:"cron_format,omitempty" yaml:"cron_format,omitempty"`
	Template   string `json:"template,omitempty" yaml:"template,omitempty"`
	TemplateID int64  `json:"template_id,omitempty" yaml:"template_id,omitempty"`
	Active     *bool  `json:"active,omitempty" yaml:"active,omitempty"`
}

// ParseFile reads and parses a config file (YAML or JSON) with env var expansion.
func ParseFile(path string) (*ApplyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg ApplyConfig

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file extension %q (use .yaml, .yml, or .json)", ext)
	}

	return &cfg, nil
}

// isSupportedExt returns true if the file extension is a supported config format.
func isSupportedExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

// CollectFiles resolves a list of file/directory paths into individual config file paths.
// Directories are scanned (non-recursively) for .yaml, .yml, and .json files.
func CollectFiles(paths []string) ([]string, error) {
	var result []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot access %q: %w", p, err)
		}
		if info.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return nil, fmt.Errorf("reading directory %q: %w", p, err)
			}
			found := false
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if isSupportedExt(e.Name()) {
					result = append(result, filepath.Join(p, e.Name()))
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("no .yaml, .yml, or .json files found in directory %q", p)
			}
		} else {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no config files specified")
	}
	return result, nil
}


// Validate checks the config for errors.
func (c *ApplyConfig) Validate() error {
	if c.Project == "" {
		return fmt.Errorf("project name is required")
	}

	// Check for duplicate names within each resource type
	if err := checkDuplicateNames("keys", keyNames(c.Keys)); err != nil {
		return err
	}
	if err := checkDuplicateNames("environments", envNames(c.Environments)); err != nil {
		return err
	}
	if err := checkDuplicateNames("repositories", repoNames(c.Repositories)); err != nil {
		return err
	}
	if err := checkDuplicateNames("inventories", inventoryNames(c.Inventories)); err != nil {
		return err
	}
	if err := checkDuplicateNames("templates", templateNames(c.Templates)); err != nil {
		return err
	}
	if err := checkDuplicateNames("schedules", scheduleNames(c.Schedules)); err != nil {
		return err
	}

	// Validate individual entries
	for i, k := range c.Keys {
		if k.Name == "" {
			return fmt.Errorf("keys[%d]: name is required", i)
		}
		if k.State == "absent" {
			continue
		}
		if k.Type == "" {
			return fmt.Errorf("keys[%d] %q: type is required (none, ssh, login_password)", i, k.Name)
		}
		switch k.Type {
		case "none", "ssh", "login_password":
		default:
			return fmt.Errorf("keys[%d] %q: invalid type %q (must be none, ssh, or login_password)", i, k.Name, k.Type)
		}
	}

	for i, e := range c.Environments {
		if e.Name == "" {
			return fmt.Errorf("environments[%d]: name is required", i)
		}
	}

	for i, r := range c.Repositories {
		if r.Name == "" {
			return fmt.Errorf("repositories[%d]: name is required", i)
		}
		if r.State == "absent" {
			continue
		}
		if r.GitURL == "" {
			return fmt.Errorf("repositories[%d] %q: git_url is required", i, r.Name)
		}
	}

	for i, inv := range c.Inventories {
		if inv.Name == "" {
			return fmt.Errorf("inventories[%d]: name is required", i)
		}
		if inv.State == "absent" {
			continue
		}
		if inv.Type == "" {
			return fmt.Errorf("inventories[%d] %q: type is required", i, inv.Name)
		}
		switch inv.Type {
		case "static", "static-yaml", "file", "terraform-workspace":
		default:
			return fmt.Errorf("inventories[%d] %q: invalid type %q", i, inv.Name, inv.Type)
		}
	}

	for i, t := range c.Templates {
		if t.Name == "" {
			return fmt.Errorf("templates[%d]: name is required", i)
		}
	}

	for i, s := range c.Schedules {
		if s.Name == "" {
			return fmt.Errorf("schedules[%d]: name is required", i)
		}
		if s.CronFormat == "" {
			return fmt.Errorf("schedules[%d] %q: cron_format is required", i, s.Name)
		}
		if s.Template == "" && s.TemplateID == 0 {
			return fmt.Errorf("schedules[%d] %q: template or template_id is required", i, s.Name)
		}
	}

	return nil
}

func checkDuplicateNames(resource string, names []string) error {
	seen := make(map[string]bool)
	for _, n := range names {
		lower := strings.ToLower(n)
		if seen[lower] {
			return fmt.Errorf("%s: duplicate name %q", resource, n)
		}
		seen[lower] = true
	}
	return nil
}

func keyNames(entries []KeyEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

func envNames(entries []EnvEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

func repoNames(entries []RepoEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

func inventoryNames(entries []InventoryEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

func templateNames(entries []TemplateEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

func scheduleNames(entries []ScheduleEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}
