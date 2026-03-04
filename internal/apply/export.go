package apply

import (
	"encoding/json"
	"fmt"
	"strings"

	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"gopkg.in/yaml.v3"
)

// Exporter dumps current Semaphore project state to ApplyConfig format.
type Exporter struct {
	client    *apiclient.Semapi
	projectID int64
	filter    []ResourceType
}

// NewExporter creates a new exporter.
func NewExporter(client *apiclient.Semapi, projectID int64, filter []ResourceType) *Exporter {
	return &Exporter{
		client:    client,
		projectID: projectID,
		filter:    filter,
	}
}

// ParseResourceFilter parses a comma-separated resource filter string.
func ParseResourceFilter(s string) ([]ResourceType, error) {
	if s == "" {
		return nil, nil
	}

	seen := make(map[ResourceType]bool)
	var result []ResourceType

	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		var rt ResourceType
		switch part {
		case "keys", "key":
			rt = ResourceKey
		case "variable_groups", "variable-groups", "vg", "environments", "envs", "env":
			rt = ResourceVariableGroup
		case "repositories", "repos", "repo":
			rt = ResourceRepository
		case "inventories", "inventory", "inv":
			rt = ResourceInventory
		case "templates", "template", "tpl":
			rt = ResourceTemplate
		default:
			return nil, fmt.Errorf("unknown resource type %q (valid: keys, variable_groups, repositories, inventories, templates)", part)
		}
		if !seen[rt] {
			seen[rt] = true
			result = append(result, rt)
		}
	}

	return result, nil
}

// Export fetches all resources and returns an ApplyConfig.
func (e *Exporter) Export(projectName string) (*ApplyConfig, error) {
	cfg := &ApplyConfig{
		Project: projectName,
	}

	// Always fetch keys and repos for cross-ref resolution
	keyIDToName := make(map[int64]string)
	repoIDToName := make(map[int64]string)
	envIDToName := make(map[int64]string)
	invIDToName := make(map[int64]string)
	tplIDToName := make(map[int64]string)

	// Fetch keys
	keys, err := e.fetchKeys()
	if err != nil {
		return nil, fmt.Errorf("fetching keys: %w", err)
	}
	for _, k := range keys {
		keyIDToName[k.ID] = k.Name
	}
	if e.includeType(ResourceKey) {
		cfg.Keys = convertKeys(keys)
	}

	// Fetch environments
	envs, err := e.fetchEnvironments()
	if err != nil {
		return nil, fmt.Errorf("fetching environments: %w", err)
	}
	for _, env := range envs {
		envIDToName[env.ID] = env.Name
	}
	if e.includeType(ResourceVariableGroup) {
		cfg.VariableGroups = convertVariableGroups(envs)
	}

	// Fetch repositories
	repos, err := e.fetchRepositories()
	if err != nil {
		return nil, fmt.Errorf("fetching repositories: %w", err)
	}
	for _, r := range repos {
		repoIDToName[r.ID] = r.Name
	}
	if e.includeType(ResourceRepository) {
		cfg.Repositories = convertRepositories(repos, keyIDToName)
	}

	// Fetch inventories
	invs, err := e.fetchInventories()
	if err != nil {
		return nil, fmt.Errorf("fetching inventories: %w", err)
	}
	for _, inv := range invs {
		invIDToName[inv.ID] = inv.Name
	}
	if e.includeType(ResourceInventory) {
		cfg.Inventories = convertInventories(invs, keyIDToName, repoIDToName)
	}

	// Fetch templates
	if e.includeType(ResourceTemplate) {
		templates, err := e.fetchTemplates()
		if err != nil {
			return nil, fmt.Errorf("fetching templates: %w", err)
		}
		for _, t := range templates {
			tplIDToName[t.ID] = t.Name
		}
		cfg.Templates = convertTemplates(templates, repoIDToName, envIDToName, invIDToName, tplIDToName)
	}

	// Schedules not exported (no list API)

	return cfg, nil
}

func (e *Exporter) includeType(rt ResourceType) bool {
	if len(e.filter) == 0 {
		return true
	}
	for _, f := range e.filter {
		if f == rt {
			return true
		}
	}
	return false
}

func (e *Exporter) fetchKeys() ([]*models.AccessKey, error) {
	params := key_store.NewGetProjectProjectIDKeysParams()
	params.ProjectID = e.projectID
	resp, err := e.client.KeyStore.GetProjectProjectIDKeys(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}

func (e *Exporter) fetchEnvironments() ([]*models.Environment, error) {
	params := variable_group.NewGetProjectProjectIDEnvironmentParams()
	params.ProjectID = e.projectID
	resp, err := e.client.VariableGroup.GetProjectProjectIDEnvironment(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}

func (e *Exporter) fetchRepositories() ([]*models.Repository, error) {
	params := repository.NewGetProjectProjectIDRepositoriesParams()
	params.ProjectID = e.projectID
	resp, err := e.client.Repository.GetProjectProjectIDRepositories(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}

func (e *Exporter) fetchInventories() ([]*models.Inventory, error) {
	params := inventory.NewGetProjectProjectIDInventoryParams()
	params.ProjectID = e.projectID
	resp, err := e.client.Inventory.GetProjectProjectIDInventory(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}

func (e *Exporter) fetchTemplates() ([]*models.Template, error) {
	params := template.NewGetProjectProjectIDTemplatesParams()
	params.ProjectID = e.projectID
	resp, err := e.client.Template.GetProjectProjectIDTemplates(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}

// Conversion helpers

func convertKeys(keys []*models.AccessKey) []KeyEntry {
	var result []KeyEntry
	for _, k := range keys {
		entry := KeyEntry{
			Name: k.Name,
			Type: k.Type,
		}
		switch k.Type {
		case "ssh":
			entry.SSH = &SSHKeyData{
				PrivateKey: "<set-me>",
			}
		case "login_password":
			entry.LoginPassword = &LoginPasswordData{
				Password: "<set-me>",
			}
		}
		result = append(result, entry)
	}
	return result
}

func convertVariableGroups(envs []*models.Environment) []VariableGroupEntry {
	var result []VariableGroupEntry
	for _, env := range envs {
		entry := VariableGroupEntry{
			Name: env.Name,
		}
		if env.JSON != "" && env.JSON != "{}" {
			var vars map[string]string
			if err := json.Unmarshal([]byte(env.JSON), &vars); err == nil {
				entry.Variables = vars
			}
		}
		if env.Env != "" && env.Env != "{}" {
			var envVars map[string]string
			if err := json.Unmarshal([]byte(env.Env), &envVars); err == nil {
				entry.EnvironmentVariables = envVars
			}
		}
		for _, s := range env.Secrets {
			switch s.Type {
			case "env":
				if entry.SecretEnvironmentVariables == nil {
					entry.SecretEnvironmentVariables = make(map[string]string)
				}
				entry.SecretEnvironmentVariables[s.Name] = "<set-me>"
			default:
				if entry.Secrets == nil {
					entry.Secrets = make(map[string]string)
				}
				entry.Secrets[s.Name] = "<set-me>"
			}
		}
		result = append(result, entry)
	}
	return result
}

func convertRepositories(repos []*models.Repository, keyIDToName map[int64]string) []RepoEntry {
	var result []RepoEntry
	for _, r := range repos {
		entry := RepoEntry{
			Name:      r.Name,
			GitURL:    r.GitURL,
			GitBranch: r.GitBranch,
		}
		if name, ok := keyIDToName[r.SSHKeyID]; ok && r.SSHKeyID != 0 {
			entry.SSHKey = name
		} else if r.SSHKeyID != 0 {
			entry.SSHKeyID = r.SSHKeyID
		}
		result = append(result, entry)
	}
	return result
}

func convertInventories(invs []*models.Inventory, keyIDToName, repoIDToName map[int64]string) []InventoryEntry {
	var result []InventoryEntry
	for _, inv := range invs {
		entry := InventoryEntry{
			Name:      inv.Name,
			Type:      inv.Type,
			Inventory: inv.Inventory,
		}
		if name, ok := keyIDToName[inv.SSHKeyID]; ok && inv.SSHKeyID != 0 {
			entry.SSHKey = name
		} else if inv.SSHKeyID != 0 {
			entry.SSHKeyID = inv.SSHKeyID
		}
		if name, ok := keyIDToName[inv.BecomeKeyID]; ok && inv.BecomeKeyID != 0 {
			entry.BecomeKey = name
		} else if inv.BecomeKeyID != 0 {
			entry.BecomeKeyID = inv.BecomeKeyID
		}
		if name, ok := repoIDToName[inv.RepositoryID]; ok && inv.RepositoryID != 0 {
			entry.Repository = name
		} else if inv.RepositoryID != 0 {
			entry.RepositoryID = inv.RepositoryID
		}
		result = append(result, entry)
	}
	return result
}

func convertTemplates(templates []*models.Template, repoIDToName, envIDToName, invIDToName, tplIDToName map[int64]string) []TemplateEntry {
	var result []TemplateEntry
	for _, t := range templates {
		entry := TemplateEntry{
			Name:                    t.Name,
			Type:                    t.Type,
			App:                     t.App,
			Playbook:                t.Playbook,
			Description:             t.Description,
			GitBranch:               t.GitBranch,
			Arguments:               t.Arguments,
			StartVersion:            t.StartVersion,
			Autorun:                 t.Autorun,
			SuppressSuccessAlerts:   t.SuppressSuccessAlerts,
			AllowOverrideArgsInTask: t.AllowOverrideArgsInTask,
			ViewID:                  t.ViewID,
		}
		if name, ok := repoIDToName[t.RepositoryID]; ok && t.RepositoryID != 0 {
			entry.Repository = name
		} else if t.RepositoryID != 0 {
			entry.RepositoryID = t.RepositoryID
		}
		if name, ok := envIDToName[t.EnvironmentID]; ok && t.EnvironmentID != 0 {
			entry.VariableGroup = name
		} else if t.EnvironmentID != 0 {
			entry.EnvironmentID = t.EnvironmentID
		}
		if name, ok := invIDToName[t.InventoryID]; ok && t.InventoryID != 0 {
			entry.Inventory = name
		} else if t.InventoryID != 0 {
			entry.InventoryID = t.InventoryID
		}
		if name, ok := tplIDToName[t.BuildTemplateID]; ok && t.BuildTemplateID != 0 {
			entry.BuildTemplate = name
		} else if t.BuildTemplateID != 0 {
			entry.BuildTemplateID = t.BuildTemplateID
		}
		result = append(result, entry)
	}
	return result
}

// MarshalYAML serializes an ApplyConfig to YAML bytes.
func MarshalYAML(cfg *ApplyConfig) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// MarshalJSON serializes an ApplyConfig to pretty-printed JSON bytes.
func MarshalJSON(cfg *ApplyConfig) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}
