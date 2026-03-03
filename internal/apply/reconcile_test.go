package apply

import (
	"testing"

	"github.com/ramanavelineni/semctl/pkg/semapi/models"
)

func TestKeyNeedsUpdate(t *testing.T) {
	tests := []struct {
		name     string
		entry    KeyEntry
		existing *models.AccessKey
		want     bool
	}{
		{
			name:     "same type no secrets",
			entry:    KeyEntry{Name: "k", Type: "none"},
			existing: &models.AccessKey{Name: "k", Type: "none"},
			want:     false,
		},
		{
			name:     "type changed",
			entry:    KeyEntry{Name: "k", Type: "ssh"},
			existing: &models.AccessKey{Name: "k", Type: "none"},
			want:     true,
		},
		{
			name:  "ssh with private key",
			entry: KeyEntry{Name: "k", Type: "ssh", SSH: &SSHKeyData{PrivateKey: "key"}},
			existing: &models.AccessKey{Name: "k", Type: "ssh"},
			want:     true,
		},
		{
			name:  "ssh without private key",
			entry: KeyEntry{Name: "k", Type: "ssh", SSH: &SSHKeyData{Login: "user"}},
			existing: &models.AccessKey{Name: "k", Type: "ssh"},
			want:     false,
		},
		{
			name:  "login_password with password",
			entry: KeyEntry{Name: "k", Type: "login_password", LoginPassword: &LoginPasswordData{Password: "pw"}},
			existing: &models.AccessKey{Name: "k", Type: "login_password"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keyNeedsUpdate(tt.entry, tt.existing); got != tt.want {
				t.Errorf("keyNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvNeedsUpdate(t *testing.T) {
	tests := []struct {
		name     string
		entry    EnvEntry
		existing *models.Environment
		want     bool
	}{
		{
			name:     "no changes",
			entry:    EnvEntry{Name: "e"},
			existing: &models.Environment{Name: "e"},
			want:     false,
		},
		{
			name:     "json changed",
			entry:    EnvEntry{Name: "e", JSON: `{"new":"val"}`},
			existing: &models.Environment{Name: "e", JSON: `{"old":"val"}`},
			want:     true,
		},
		{
			name:     "json same",
			entry:    EnvEntry{Name: "e", JSON: `{"k":"v"}`},
			existing: &models.Environment{Name: "e", JSON: `{"k":"v"}`},
			want:     false,
		},
		{
			name:     "password set",
			entry:    EnvEntry{Name: "e", Password: "secret"},
			existing: &models.Environment{Name: "e"},
			want:     true,
		},
		{
			name:     "env changed",
			entry:    EnvEntry{Name: "e", Env: "new"},
			existing: &models.Environment{Name: "e", Env: "old"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envNeedsUpdate(tt.entry, tt.existing); got != tt.want {
				t.Errorf("envNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoNeedsUpdate(t *testing.T) {
	r := &Reconciler{
		KeyIDByName: map[string]int64{"ssh key": 5},
	}

	tests := []struct {
		name     string
		entry    RepoEntry
		existing *models.Repository
		want     bool
	}{
		{
			name:     "no changes",
			entry:    RepoEntry{Name: "r", GitURL: "url"},
			existing: &models.Repository{Name: "r", GitURL: "url"},
			want:     false,
		},
		{
			name:     "url changed",
			entry:    RepoEntry{Name: "r", GitURL: "new-url"},
			existing: &models.Repository{Name: "r", GitURL: "old-url"},
			want:     true,
		},
		{
			name:     "branch changed",
			entry:    RepoEntry{Name: "r", GitBranch: "develop"},
			existing: &models.Repository{Name: "r", GitBranch: "main"},
			want:     true,
		},
		{
			name:     "ssh key by name changed",
			entry:    RepoEntry{Name: "r", SSHKey: "SSH Key"},
			existing: &models.Repository{Name: "r", SSHKeyID: 3},
			want:     true,
		},
		{
			name:     "ssh key by name same",
			entry:    RepoEntry{Name: "r", SSHKey: "SSH Key"},
			existing: &models.Repository{Name: "r", SSHKeyID: 5},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.repoNeedsUpdate(tt.entry, tt.existing); got != tt.want {
				t.Errorf("repoNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateNeedsUpdate(t *testing.T) {
	r := &Reconciler{
		RepoIDByName:      map[string]int64{"main repo": 10},
		EnvIDByName:       map[string]int64{"prod": 20},
		InventoryIDByName: map[string]int64{"hosts": 30},
		TemplateIDByName:  map[string]int64{"build": 40},
	}

	tests := []struct {
		name     string
		entry    TemplateEntry
		existing *models.Template
		want     bool
	}{
		{
			name:     "no changes",
			entry:    TemplateEntry{Name: "t"},
			existing: &models.Template{Name: "t"},
			want:     false,
		},
		{
			name:     "playbook changed",
			entry:    TemplateEntry{Name: "t", Playbook: "new.yml"},
			existing: &models.Template{Name: "t", Playbook: "old.yml"},
			want:     true,
		},
		{
			name:     "autorun changed",
			entry:    TemplateEntry{Name: "t", Autorun: true},
			existing: &models.Template{Name: "t", Autorun: false},
			want:     true,
		},
		{
			name:     "repo ref by name same",
			entry:    TemplateEntry{Name: "t", Repository: "Main Repo"},
			existing: &models.Template{Name: "t", RepositoryID: 10},
			want:     false,
		},
		{
			name:     "repo ref by name changed",
			entry:    TemplateEntry{Name: "t", Repository: "Main Repo"},
			existing: &models.Template{Name: "t", RepositoryID: 99},
			want:     true,
		},
		{
			name:     "env ref by name same",
			entry:    TemplateEntry{Name: "t", Environment: "Prod"},
			existing: &models.Template{Name: "t", EnvironmentID: 20},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.templateNeedsUpdate(tt.entry, tt.existing); got != tt.want {
				t.Errorf("templateNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindByNameHelpers(t *testing.T) {
	keys := []*models.AccessKey{
		{ID: 1, Name: "Key A"},
		{ID: 2, Name: "Key B"},
	}

	if k := findKeyByName(keys, "key a"); k == nil || k.ID != 1 {
		t.Error("findKeyByName case-insensitive failed")
	}
	if k := findKeyByName(keys, "Key C"); k != nil {
		t.Error("findKeyByName should return nil for missing")
	}

	envs := []*models.Environment{
		{ID: 10, Name: "Prod"},
	}
	if e := findEnvByName(envs, "PROD"); e == nil || e.ID != 10 {
		t.Error("findEnvByName case-insensitive failed")
	}

	repos := []*models.Repository{
		{ID: 20, Name: "Main Repo"},
	}
	if r := findRepoByName(repos, "main repo"); r == nil || r.ID != 20 {
		t.Error("findRepoByName case-insensitive failed")
	}

	invs := []*models.Inventory{
		{ID: 30, Name: "Hosts"},
	}
	if inv := findInventoryByName(invs, "hosts"); inv == nil || inv.ID != 30 {
		t.Error("findInventoryByName case-insensitive failed")
	}

	templates := []*models.Template{
		{ID: 40, Name: "Deploy"},
	}
	if tpl := findTemplateByName(templates, "DEPLOY"); tpl == nil || tpl.ID != 40 {
		t.Error("findTemplateByName case-insensitive failed")
	}
}

func TestResolveIDHelpers(t *testing.T) {
	r := &Reconciler{
		KeyIDByName:       map[string]int64{"mykey": 5},
		EnvIDByName:       map[string]int64{"myenv": 10},
		RepoIDByName:      map[string]int64{"myrepo": 15},
		InventoryIDByName: map[string]int64{"myinv": 20},
		TemplateIDByName:  map[string]int64{"mytpl": 25},
	}

	// Explicit ID takes precedence
	if id := r.resolveKeyID("mykey", 99); id != 99 {
		t.Errorf("resolveKeyID with explicit ID = %d, want 99", id)
	}

	// Name lookup
	if id := r.resolveKeyID("mykey", 0); id != 5 {
		t.Errorf("resolveKeyID with name = %d, want 5", id)
	}

	// No match
	if id := r.resolveKeyID("unknown", 0); id != 0 {
		t.Errorf("resolveKeyID unknown = %d, want 0", id)
	}

	// Same pattern for others
	if id := r.resolveEnvID("myenv", 0); id != 10 {
		t.Errorf("resolveEnvID = %d, want 10", id)
	}
	if id := r.resolveRepoID("myrepo", 0); id != 15 {
		t.Errorf("resolveRepoID = %d, want 15", id)
	}
	if id := r.resolveInventoryID("myinv", 0); id != 20 {
		t.Errorf("resolveInventoryID = %d, want 20", id)
	}
	if id := r.resolveTemplateID("mytpl", 0); id != 25 {
		t.Errorf("resolveTemplateID = %d, want 25", id)
	}
}
