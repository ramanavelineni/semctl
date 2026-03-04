package apply

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `
project: "My Project"
keys:
  - name: "SSH Key"
    type: ssh
    ssh:
      login: deploy
      private_key: "secret-key"
variable_groups:
  - group_name: "Production"
    variables:
      db: "10.0.0.1"
repositories:
  - name: "Main Repo"
    git_url: "git@github.com:org/repo.git"
    ssh_key: "SSH Key"
inventories:
  - name: "Prod Hosts"
    type: static
    inventory: "[all]\n10.0.0.1"
templates:
  - name: "Deploy"
    playbook: deploy.yml
    repository: "Main Repo"
schedules:
  - name: "Nightly"
    cron_format: "0 2 * * *"
    template: "Deploy"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Project != "My Project" {
		t.Errorf("Project = %q, want %q", cfg.Project, "My Project")
	}
	if len(cfg.Keys) != 1 {
		t.Fatalf("len(Keys) = %d, want 1", len(cfg.Keys))
	}
	if cfg.Keys[0].Type != "ssh" {
		t.Errorf("Keys[0].Type = %q, want %q", cfg.Keys[0].Type, "ssh")
	}
	if cfg.Keys[0].SSH.PrivateKey != "secret-key" {
		t.Errorf("Keys[0].SSH.PrivateKey = %q, want %q", cfg.Keys[0].SSH.PrivateKey, "secret-key")
	}
	if len(cfg.VariableGroups) != 1 {
		t.Fatalf("len(VariableGroups) = %d, want 1", len(cfg.VariableGroups))
	}
	if len(cfg.Repositories) != 1 {
		t.Fatalf("len(Repositories) = %d, want 1", len(cfg.Repositories))
	}
	if cfg.Repositories[0].SSHKey != "SSH Key" {
		t.Errorf("Repositories[0].SSHKey = %q, want %q", cfg.Repositories[0].SSHKey, "SSH Key")
	}
	if len(cfg.Templates) != 1 {
		t.Fatalf("len(Templates) = %d, want 1", len(cfg.Templates))
	}
	if len(cfg.Schedules) != 1 {
		t.Fatalf("len(Schedules) = %d, want 1", len(cfg.Schedules))
	}
}

func TestParseFileJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	content := `{
  "project": "JSON Project",
  "keys": [
    {"name": "Key1", "type": "none"}
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Project != "JSON Project" {
		t.Errorf("Project = %q, want %q", cfg.Project, "JSON Project")
	}
	if len(cfg.Keys) != 1 {
		t.Fatalf("len(Keys) = %d, want 1", len(cfg.Keys))
	}
}

func TestParseFileEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `
project: "Test"
keys:
  - name: "SSH Key"
    type: ssh
    ssh:
      private_key: "${TEST_SEMCTL_KEY}"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_SEMCTL_KEY", "my-expanded-key")

	cfg, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Keys[0].SSH.PrivateKey != "my-expanded-key" {
		t.Errorf("SSH.PrivateKey = %q, want %q", cfg.Keys[0].SSH.PrivateKey, "my-expanded-key")
	}
}

func TestParseFileBadExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for unsupported extension")
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidateProjectRequired(t *testing.T) {
	cfg := &ApplyConfig{}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty project")
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test",
		Keys: []KeyEntry{
			{Name: "Key1", Type: "none"},
			{Name: "key1", Type: "ssh"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for duplicate key names")
	}
}

func TestValidateKeyTypeRequired(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test",
		Keys:    []KeyEntry{{Name: "Key1"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing key type")
	}
}

func TestValidateKeyTypeInvalid(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test",
		Keys:    []KeyEntry{{Name: "Key1", Type: "bad"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid key type")
	}
}

func TestValidateAbsentOnlyNeedsName(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test",
		Keys:    []KeyEntry{{Name: "Key1", State: "absent"}},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("absent key should only need name, got: %v", err)
	}
}

func TestValidateRepoGitURLRequired(t *testing.T) {
	cfg := &ApplyConfig{
		Project:      "Test",
		Repositories: []RepoEntry{{Name: "Repo1"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing git_url")
	}
}

func TestValidateInventoryTypeRequired(t *testing.T) {
	cfg := &ApplyConfig{
		Project:     "Test",
		Inventories: []InventoryEntry{{Name: "Inv1"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing inventory type")
	}
}

func TestValidateInventoryTypeInvalid(t *testing.T) {
	cfg := &ApplyConfig{
		Project:     "Test",
		Inventories: []InventoryEntry{{Name: "Inv1", Type: "bad"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid inventory type")
	}
}

func TestValidateScheduleRequiresCron(t *testing.T) {
	cfg := &ApplyConfig{
		Project:   "Test",
		Schedules: []ScheduleEntry{{Name: "Sched1", Template: "tpl"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing cron_format")
	}
}

func TestValidateScheduleRequiresTemplate(t *testing.T) {
	cfg := &ApplyConfig{
		Project:   "Test",
		Schedules: []ScheduleEntry{{Name: "Sched1", CronFormat: "* * * * *"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing template")
	}
}

func TestValidateValid(t *testing.T) {
	cfg := &ApplyConfig{
		Project: "Test",
		Keys:    []KeyEntry{{Name: "Key1", Type: "none"}},
		VariableGroups: []VariableGroupEntry{{Name: "vg1"}},
		Repositories: []RepoEntry{{Name: "Repo1", GitURL: "git@github.com:org/repo.git"}},
		Inventories:  []InventoryEntry{{Name: "Inv1", Type: "static"}},
		Templates:    []TemplateEntry{{Name: "Tpl1"}},
		Schedules:    []ScheduleEntry{{Name: "Sched1", CronFormat: "* * * * *", Template: "Tpl1"}},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestCollectFilesWithFiles(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.yaml")
	p2 := filepath.Join(dir, "b.json")
	os.WriteFile(p1, []byte("project: A"), 0644)
	os.WriteFile(p2, []byte(`{"project":"B"}`), 0644)

	files, err := CollectFiles([]string{p1, p2})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestCollectFilesWithDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("project: A"), 0644)
	os.WriteFile(filepath.Join(dir, "b.yml"), []byte("project: B"), 0644)
	os.WriteFile(filepath.Join(dir, "c.json"), []byte(`{"project":"C"}`), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0644)

	files, err := CollectFiles([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files from directory, got %d", len(files))
	}
}

func TestCollectFilesEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("no configs"), 0644)

	_, err := CollectFiles([]string{dir})
	if err == nil {
		t.Error("expected error for directory with no config files")
	}
}

func TestCollectFilesMissing(t *testing.T) {
	_, err := CollectFiles([]string{"/nonexistent/path"})
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestCollectFilesEmpty(t *testing.T) {
	_, err := CollectFiles([]string{})
	if err == nil {
		t.Error("expected error for empty paths")
	}
}

func TestCollectFilesMixedFileAndDir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "configs")
	os.MkdirAll(subdir, 0755)

	single := filepath.Join(dir, "main.yaml")
	os.WriteFile(single, []byte("project: Main"), 0644)
	os.WriteFile(filepath.Join(subdir, "extra.yaml"), []byte("project: Extra"), 0644)

	files, err := CollectFiles([]string{single, subdir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}
