package apply

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExportPlaceholder is written by `semctl export` in place of secret values,
// which the API never returns. Apply refuses configs that still contain it.
const ExportPlaceholder = "<set-me>"

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool {
	return &b
}

// ApplyConfig represents the full declarative configuration file.
type ApplyConfig struct {
	Project        string               `json:"project" yaml:"project"`
	ProjectState   string               `json:"project_state,omitempty" yaml:"project_state,omitempty"`
	Keys           []KeyEntry           `json:"keys,omitempty" yaml:"keys,omitempty"`
	VariableGroups []VariableGroupEntry `json:"variable_groups,omitempty" yaml:"variable_groups,omitempty"`
	Repositories   []RepoEntry          `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Inventories    []InventoryEntry     `json:"inventories,omitempty" yaml:"inventories,omitempty"`
	Templates      []TemplateEntry      `json:"templates,omitempty" yaml:"templates,omitempty"`
	Schedules      []ScheduleEntry      `json:"schedules,omitempty" yaml:"schedules,omitempty"`
}

// KeyEntry represents a key in the config file.
type KeyEntry struct {
	Name          string             `json:"name" yaml:"name"`
	Type          string             `json:"type,omitempty" yaml:"type,omitempty"`
	State         string             `json:"state,omitempty" yaml:"state,omitempty"`
	SSH           *SSHKeyData        `json:"ssh,omitempty" yaml:"ssh,omitempty"`
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

// VariableGroupEntry represents a variable group in the config file.
type VariableGroupEntry struct {
	Name                       string            `json:"group_name" yaml:"group_name"`
	State                      string            `json:"state,omitempty" yaml:"state,omitempty"`
	Variables                  map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`
	EnvironmentVariables       map[string]string `json:"environment_variables,omitempty" yaml:"environment_variables,omitempty"`
	Secrets                    map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	SecretEnvironmentVariables map[string]string `json:"secret_environment_variables,omitempty" yaml:"secret_environment_variables,omitempty"`
}

// EnvVarsToJSON serializes an environment variables map into a JSON string for the API's env field.
func EnvVarsToJSON(vars map[string]string) string {
	if len(vars) == 0 {
		return "{}"
	}
	data, _ := json.Marshal(vars)
	return string(data)
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
// Boolean fields are pointers so that "not specified" (keep the existing
// value) is distinguishable from an explicit false.
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
	Autorun                 *bool  `json:"autorun,omitempty" yaml:"autorun,omitempty"`
	SuppressSuccessAlerts   *bool  `json:"suppress_success_alerts,omitempty" yaml:"suppress_success_alerts,omitempty"`
	AllowOverrideArgsInTask *bool  `json:"allow_override_args_in_task,omitempty" yaml:"allow_override_args_in_task,omitempty"`
	Repository              string `json:"repository,omitempty" yaml:"repository,omitempty"`
	RepositoryID            int64  `json:"repository_id,omitempty" yaml:"repository_id,omitempty"`
	VariableGroup           string `json:"variable_group,omitempty" yaml:"variable_group,omitempty"`
	Environment             string `json:"environment,omitempty" yaml:"environment,omitempty"` // alias for variable_group
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
	State      string `json:"state,omitempty" yaml:"state,omitempty"`
	CronFormat string `json:"cron_format,omitempty" yaml:"cron_format,omitempty"`
	Template   string `json:"template,omitempty" yaml:"template,omitempty"`
	TemplateID int64  `json:"template_id,omitempty" yaml:"template_id,omitempty"`
	Active     *bool  `json:"active,omitempty" yaml:"active,omitempty"`
}

// envVarPattern matches ${VAR} references and their $${VAR} escape form.
// Bare $VAR (no braces) is intentionally NOT expanded — Ansible arguments and
// passwords legitimately contain dollar signs.
var envVarPattern = regexp.MustCompile(`\$?\$\{[A-Za-z_][A-Za-z0-9_]*\}`)

// expandEnv expands ${VAR} references in s. "$${VAR}" escapes to a literal
// "${VAR}". Returns the expanded string and the sorted, de-duplicated names
// of referenced variables that are not set in the environment.
func expandEnv(s string) (string, []string) {
	missingSet := map[string]bool{}
	out := envVarPattern.ReplaceAllStringFunc(s, func(m string) string {
		if strings.HasPrefix(m, "$$") {
			return m[1:] // $${VAR} → ${VAR}
		}
		name := m[2 : len(m)-1]
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		missingSet[name] = true
		return ""
	})

	var missing []string
	for name := range missingSet {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return out, missing
}

// expandConfigEnv expands ${VAR} references in every string field, slice
// element, and string-map key/value of cfg. Expansion runs AFTER parsing so
// an environment value can never alter the YAML/JSON document structure —
// a value containing "\nproject_state: absent" stays an inert string.
// Returns the sorted names of referenced variables not set in the environment.
func expandConfigEnv(cfg *ApplyConfig) []string {
	missingSet := map[string]bool{}
	expandValue(reflect.ValueOf(cfg).Elem(), missingSet)

	missing := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return missing
}

func expandValue(v reflect.Value, missing map[string]bool) {
	switch v.Kind() {
	case reflect.String:
		out, miss := expandEnv(v.String())
		for _, m := range miss {
			missing[m] = true
		}
		v.SetString(out)
	case reflect.Pointer:
		if !v.IsNil() {
			expandValue(v.Elem(), missing)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			expandValue(v.Field(i), missing)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			expandValue(v.Index(i), missing)
		}
	case reflect.Map:
		if v.IsNil() || v.Type().Key().Kind() != reflect.String || v.Type().Elem().Kind() != reflect.String {
			return
		}
		// Map entries are not addressable; rebuild the map.
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			key, missK := expandEnv(iter.Key().String())
			val, missV := expandEnv(iter.Value().String())
			for _, m := range append(missK, missV...) {
				missing[m] = true
			}
			out.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
		}
		v.Set(out)
	}
}

// ParseFile reads and parses a config file (YAML or JSON), then expands
// ${VAR} environment variable references in the parsed string values.
// Referencing an unset variable is an error.
func ParseFile(path string) (*ApplyConfig, error) {
	cfg, missing, err := parseFileLenient(path)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("undefined environment variable(s): %s (use $${VAR} for a literal ${VAR})", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// ParseFileOffline parses a config file like ParseFile but tolerates unset
// environment variables (they expand to empty), returning their names so the
// caller can warn. Used by offline validation, where secrets are typically
// not present in the environment.
func ParseFileOffline(path string) (*ApplyConfig, []string, error) {
	return parseFileLenient(path)
}

func parseFileLenient(path string) (*ApplyConfig, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg ApplyConfig

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		// Strict field checking: a typo like "enviroment_variables:" used
		// to be dropped silently, meaning secrets never got applied.
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, fmt.Errorf("parsing YAML: %w", err)
		}
	case ".json":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cfg); err != nil {
			return nil, nil, fmt.Errorf("parsing JSON: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported file extension %q (use .yaml, .yml, or .json)", ext)
	}

	// Expand env references only after the document structure is fixed.
	missing := expandConfigEnv(&cfg)

	return &cfg, missing, nil
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

	if c.ProjectState != "" && c.ProjectState != "absent" {
		return fmt.Errorf("project_state must be \"absent\" or empty")
	}

	// Check for duplicate names within each resource type
	if err := checkDuplicateNames("keys", keyNames(c.Keys)); err != nil {
		return err
	}
	if err := checkDuplicateNames("variable_groups", varGroupNames(c.VariableGroups)); err != nil {
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

	for i, vg := range c.VariableGroups {
		if vg.Name == "" {
			return fmt.Errorf("variable_groups[%d]: group_name is required", i)
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

	for i := range c.Templates {
		if c.Templates[i].Name == "" {
			return fmt.Errorf("templates[%d]: name is required", i)
		}
		// Support "environment" as alias for "variable_group" in template refs
		if c.Templates[i].Environment != "" && c.Templates[i].VariableGroup != "" {
			return fmt.Errorf("templates[%d] %q: cannot specify both \"environment\" and \"variable_group\"", i, c.Templates[i].Name)
		}
		if c.Templates[i].Environment != "" {
			c.Templates[i].VariableGroup = c.Templates[i].Environment
			c.Templates[i].Environment = ""
		}
	}

	for i, s := range c.Schedules {
		if s.Name == "" {
			return fmt.Errorf("schedules[%d]: name is required", i)
		}
		if s.State == "absent" {
			continue
		}
		if s.CronFormat == "" {
			return fmt.Errorf("schedules[%d] %q: cron_format is required", i, s.Name)
		}
		if s.Template == "" && s.TemplateID == 0 {
			return fmt.Errorf("schedules[%d] %q: template or template_id is required", i, s.Name)
		}
	}

	return c.checkPlaceholders()
}

// checkPlaceholders rejects configs that still contain the <set-me>
// placeholder written by `semctl export`. Applying one would overwrite real
// keys and secrets with the literal placeholder text.
func (c *ApplyConfig) checkPlaceholders() error {
	fail := func(where string) error {
		return fmt.Errorf("%s still contains the %q placeholder from 'semctl export': set the real value (or an ${ENV_VAR} reference) before applying", where, ExportPlaceholder)
	}

	for i, k := range c.Keys {
		if k.SSH != nil && (k.SSH.PrivateKey == ExportPlaceholder || k.SSH.Passphrase == ExportPlaceholder || k.SSH.Login == ExportPlaceholder) {
			return fail(fmt.Sprintf("keys[%d] %q", i, k.Name))
		}
		if k.LoginPassword != nil && (k.LoginPassword.Password == ExportPlaceholder || k.LoginPassword.Login == ExportPlaceholder) {
			return fail(fmt.Sprintf("keys[%d] %q", i, k.Name))
		}
	}

	for i, vg := range c.VariableGroups {
		for name, val := range vg.Secrets {
			if val == ExportPlaceholder {
				return fail(fmt.Sprintf("variable_groups[%d] %q secret %q", i, vg.Name, name))
			}
		}
		for name, val := range vg.SecretEnvironmentVariables {
			if val == ExportPlaceholder {
				return fail(fmt.Sprintf("variable_groups[%d] %q secret env var %q", i, vg.Name, name))
			}
		}
	}

	return nil
}

// VarsToJSON serializes a variables map into a JSON string for the API's json field.
func VarsToJSON(vars map[string]string) string {
	if len(vars) == 0 {
		return "{}"
	}
	data, _ := json.Marshal(vars)
	return string(data)
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

func varGroupNames(entries []VariableGroupEntry) []string {
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
