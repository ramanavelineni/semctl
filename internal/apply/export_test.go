package apply

import (
	"encoding/json"
	"testing"

	"github.com/ramanavelineni/semctl/pkg/semapi/models"
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
	if result[1] != ResourceVariableGroup {
		t.Errorf("result[1] = %q, want %q", result[1], ResourceVariableGroup)
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
		{"env", ResourceVariableGroup},
		{"envs", ResourceVariableGroup},
		{"environments", ResourceVariableGroup},
		{"variable_groups", ResourceVariableGroup},
		{"vg", ResourceVariableGroup},
		{"repo", ResourceRepository},
		{"repos", ResourceRepository},
		{"repositories", ResourceRepository},
		{"inventory", ResourceInventory},
		{"inventories", ResourceInventory},
		{"inv", ResourceInventory},
		{"template", ResourceTemplate},
		{"templates", ResourceTemplate},
		{"tpl", ResourceTemplate},
		{"schedule", ResourceSchedule},
		{"schedules", ResourceSchedule},
		{"sched", ResourceSchedule},
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

func TestConvertSchedules(t *testing.T) {
	schedules := []*models.Schedule{
		{ID: 1, Name: "Nightly", CronFormat: "0 2 * * *", TemplateID: 40, TplName: "Deploy", Active: true},
		{ID: 2, Name: "Paused", CronFormat: "0 3 * * *", TemplateID: 41, Active: false},
	}

	entries := convertSchedules(schedules)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Template != "Deploy" {
		t.Errorf("template name ref: got %q, want Deploy", entries[0].Template)
	}
	if entries[0].TemplateID != 0 {
		t.Errorf("template_id should be omitted when name is known, got %d", entries[0].TemplateID)
	}
	if entries[0].Active != nil {
		t.Errorf("active should be omitted when true, got %v", *entries[0].Active)
	}

	if entries[1].Template != "" || entries[1].TemplateID != 41 {
		t.Errorf("expected template_id fallback when tpl_name is empty, got template=%q id=%d", entries[1].Template, entries[1].TemplateID)
	}
	if entries[1].Active == nil || *entries[1].Active {
		t.Error("active should be explicit false for inactive schedules")
	}
}

func TestMarshalYAMLRoundTrip(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test Project",
		Keys: []KeyEntry{
			{Name: "Key1", Type: "none"},
		},
		VariableGroups: []VariableGroupEntry{
			{Name: "Prod", Variables: map[string]string{"k": "v"}},
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
	if len(parsed.VariableGroups) != 1 || parsed.VariableGroups[0].Name != "Prod" {
		t.Errorf("VariableGroups roundtrip failed")
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
