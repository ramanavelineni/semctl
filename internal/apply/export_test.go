package apply

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseResourceFilterEmpty(t *testing.T) {
	result, err := ParseResourceFilter("")
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestParseResourceFilterValid(t *testing.T) {
	result, err := ParseResourceFilter("keys,envs,repos")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 types, got %d", len(result))
	}
	if result[0] != ResourceKey {
		t.Errorf("result[0] = %q, want %q", result[0], ResourceKey)
	}
	if result[1] != ResourceEnvironment {
		t.Errorf("result[1] = %q, want %q", result[1], ResourceEnvironment)
	}
	if result[2] != ResourceRepository {
		t.Errorf("result[2] = %q, want %q", result[2], ResourceRepository)
	}
}

func TestParseResourceFilterAliases(t *testing.T) {
	tests := []struct {
		input string
		want  ResourceType
	}{
		{"key", ResourceKey},
		{"keys", ResourceKey},
		{"env", ResourceEnvironment},
		{"envs", ResourceEnvironment},
		{"environments", ResourceEnvironment},
		{"repo", ResourceRepository},
		{"repos", ResourceRepository},
		{"repositories", ResourceRepository},
		{"inventory", ResourceInventory},
		{"inventories", ResourceInventory},
		{"inv", ResourceInventory},
		{"template", ResourceTemplate},
		{"templates", ResourceTemplate},
		{"tpl", ResourceTemplate},
	}
	for _, tt := range tests {
		result, err := ParseResourceFilter(tt.input)
		if err != nil {
			t.Errorf("ParseResourceFilter(%q) error: %v", tt.input, err)
			continue
		}
		if len(result) != 1 || result[0] != tt.want {
			t.Errorf("ParseResourceFilter(%q) = %v, want [%q]", tt.input, result, tt.want)
		}
	}
}

func TestParseResourceFilterInvalid(t *testing.T) {
	_, err := ParseResourceFilter("bad")
	if err == nil {
		t.Error("expected error for invalid resource type")
	}
}

func TestParseResourceFilterDedup(t *testing.T) {
	result, err := ParseResourceFilter("keys,keys,key")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 type after dedup, got %d", len(result))
	}
}

func TestMarshalYAMLRoundTrip(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test Project",
		Keys: []KeyEntry{
			{Name: "Key1", Type: "none"},
		},
		Environments: []EnvEntry{
			{Name: "Prod", JSON: `{"k":"v"}`},
		},
	}

	data, err := MarshalYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var parsed ApplyConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Project != "Test Project" {
		t.Errorf("Project = %q, want %q", parsed.Project, "Test Project")
	}
	if len(parsed.Keys) != 1 || parsed.Keys[0].Name != "Key1" {
		t.Errorf("Keys roundtrip failed")
	}
	if len(parsed.Environments) != 1 || parsed.Environments[0].Name != "Prod" {
		t.Errorf("Environments roundtrip failed")
	}
}

func TestMarshalJSONRoundTrip(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test Project",
		Repositories: []RepoEntry{
			{Name: "Repo1", GitURL: "git@github.com:org/repo.git", SSHKey: "My Key"},
		},
	}

	data, err := MarshalJSON(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var parsed ApplyConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Project != "Test Project" {
		t.Errorf("Project = %q, want %q", parsed.Project, "Test Project")
	}
	if len(parsed.Repositories) != 1 || parsed.Repositories[0].SSHKey != "My Key" {
		t.Errorf("Repositories roundtrip failed")
	}
}
