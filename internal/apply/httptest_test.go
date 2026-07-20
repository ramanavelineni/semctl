package apply

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
)

// fakeSemaphore serves the Semaphore endpoints apply uses from in-memory
// state. The go-swagger client talks real HTTP to it, so these tests exercise
// the actual wire payloads the reconciler and executor produce — the layer
// that previously relied entirely on manual container smoke tests.
type fakeSemaphore struct {
	srv *httptest.Server

	projects  []*models.Project
	keys      []*models.AccessKey
	envs      []*models.Environment
	repos     []*models.Repository
	invs      []*models.Inventory
	templates []*models.Template
	schedules []*models.Schedule

	nextID   int64
	captured []capturedRequest
	failOn   map[string]int // "METHOD /path" → status code to fail with
}

type capturedRequest struct {
	key  string // "METHOD /path"
	body []byte
}

func newFakeSemaphore(t *testing.T) *fakeSemaphore {
	t.Helper()
	f := &fakeSemaphore{nextID: 100, failOn: map[string]int{}}
	mux := http.NewServeMux()

	handle := func(pattern string, fn http.HandlerFunc) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			key := r.Method + " " + r.URL.Path
			f.captured = append(f.captured, capturedRequest{key: key, body: body})
			if code, ok := f.failOn[key]; ok {
				w.WriteHeader(code)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			fn(w, r)
		})
	}
	writeJSON := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	noContent := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }

	// Projects
	handle("GET /api/projects", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.projects)
	})
	handle("POST /api/projects", func(w http.ResponseWriter, r *http.Request) {
		var req models.ProjectRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		p := &models.Project{ID: f.id(), Name: req.Name}
		f.projects = append(f.projects, p)
		writeJSON(w, http.StatusCreated, p)
	})
	// The generated client uses a trailing slash for project deletion.
	handle("DELETE /api/project/{pid}/{$}", noContent)

	// Keys
	handle("GET /api/project/{pid}/keys", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.keys)
	})
	handle("POST /api/project/{pid}/keys", func(w http.ResponseWriter, r *http.Request) {
		var req models.AccessKeyRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		k := &models.AccessKey{ID: f.id(), Name: req.Name, Type: req.Type}
		f.keys = append(f.keys, k)
		writeJSON(w, http.StatusCreated, k)
	})
	handle("PUT /api/project/{pid}/keys/{id}", noContent)
	handle("DELETE /api/project/{pid}/keys/{id}", noContent)

	// Variable groups ("environment"). The list response omits secrets, like
	// Semaphore 2.18 does; only GET-by-ID includes them.
	handle("GET /api/project/{pid}/environment", func(w http.ResponseWriter, _ *http.Request) {
		stripped := make([]*models.Environment, 0, len(f.envs))
		for _, e := range f.envs {
			cp := *e
			cp.Secrets = nil
			stripped = append(stripped, &cp)
		}
		writeJSON(w, http.StatusOK, stripped)
	})
	handle("GET /api/project/{pid}/environment/{id}", func(w http.ResponseWriter, r *http.Request) {
		for _, e := range f.envs {
			if pathID(r, "id") == e.ID {
				writeJSON(w, http.StatusOK, e)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})
	handle("POST /api/project/{pid}/environment", func(w http.ResponseWriter, r *http.Request) {
		var req models.EnvironmentRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		e := &models.Environment{ID: f.id(), Name: req.Name, JSON: req.JSON, Env: req.Env}
		f.envs = append(f.envs, e)
		writeJSON(w, http.StatusCreated, e)
	})
	handle("PUT /api/project/{pid}/environment/{id}", noContent)
	handle("DELETE /api/project/{pid}/environment/{id}", noContent)

	// Repositories
	handle("GET /api/project/{pid}/repositories", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.repos)
	})
	handle("POST /api/project/{pid}/repositories", func(w http.ResponseWriter, r *http.Request) {
		var req models.RepositoryRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		rp := &models.Repository{ID: f.id(), Name: req.Name, GitURL: req.GitURL, GitBranch: req.GitBranch, SSHKeyID: req.SSHKeyID}
		f.repos = append(f.repos, rp)
		writeJSON(w, http.StatusCreated, rp)
	})
	handle("PUT /api/project/{pid}/repositories/{id}", noContent)
	handle("DELETE /api/project/{pid}/repositories/{id}", noContent)

	// Inventories
	handle("GET /api/project/{pid}/inventory", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.invs)
	})
	handle("POST /api/project/{pid}/inventory", func(w http.ResponseWriter, r *http.Request) {
		var req models.InventoryRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		inv := &models.Inventory{ID: f.id(), Name: req.Name, Type: req.Type}
		f.invs = append(f.invs, inv)
		writeJSON(w, http.StatusCreated, inv)
	})
	handle("PUT /api/project/{pid}/inventory/{id}", noContent)
	handle("DELETE /api/project/{pid}/inventory/{id}", noContent)

	// Templates
	handle("GET /api/project/{pid}/templates", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.templates)
	})
	handle("POST /api/project/{pid}/templates", func(w http.ResponseWriter, r *http.Request) {
		var req models.TemplateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		tp := &models.Template{ID: f.id(), Name: req.Name}
		f.templates = append(f.templates, tp)
		writeJSON(w, http.StatusCreated, tp)
	})
	handle("PUT /api/project/{pid}/templates/{id}", noContent)
	handle("DELETE /api/project/{pid}/templates/{id}", noContent)

	// Schedules
	handle("GET /api/project/{pid}/schedules", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.schedules)
	})
	handle("POST /api/project/{pid}/schedules", func(w http.ResponseWriter, r *http.Request) {
		var req models.ScheduleRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		s := &models.Schedule{ID: f.id(), Name: req.Name, CronFormat: req.CronFormat}
		f.schedules = append(f.schedules, s)
		writeJSON(w, http.StatusCreated, s)
	})
	handle("PUT /api/project/{pid}/schedules/{id}", noContent)
	handle("DELETE /api/project/{pid}/schedules/{id}", noContent)

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func pathID(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id
}

func (f *fakeSemaphore) id() int64 {
	f.nextID++
	return f.nextID
}

// client returns a go-swagger client pointed at the fake server.
func (f *fakeSemaphore) client() *apiclient.Semapi {
	u, _ := url.Parse(f.srv.URL)
	tr := httptransport.New(u.Host, "/api", []string{"http"})
	return apiclient.New(tr, strfmt.Default)
}

// lastBody decodes the most recent captured request matching key into out.
func (f *fakeSemaphore) lastBody(t *testing.T, key string, out any) {
	t.Helper()
	for i := len(f.captured) - 1; i >= 0; i-- {
		if f.captured[i].key == key {
			if err := json.Unmarshal(f.captured[i].body, out); err != nil {
				t.Fatalf("decoding %s body: %v", key, err)
			}
			return
		}
	}
	t.Fatalf("no captured request for %s (captured: %v)", key, f.capturedKeys())
}

func (f *fakeSemaphore) sawRequest(key string) bool {
	for _, c := range f.captured {
		if c.key == key {
			return true
		}
	}
	return false
}

func (f *fakeSemaphore) capturedKeys() []string {
	keys := make([]string, 0, len(f.captured))
	for _, c := range f.captured {
		keys = append(keys, c.key)
	}
	return keys
}

func findAction(t *testing.T, plan *Plan, rt ResourceType, label string) ResourceAction {
	t.Helper()
	for _, a := range plan.Actions {
		if a.Type == rt && a.Label == label {
			return a
		}
	}
	t.Fatalf("no plan action for %s %q", rt, label)
	return ResourceAction{}
}

func TestBuildPlan_MixedActions(t *testing.T) {
	f := newFakeSemaphore(t)
	f.projects = []*models.Project{{ID: 1, Name: "proj"}}
	f.repos = []*models.Repository{{ID: 5, Name: "r1", GitURL: "https://git/x.git", GitBranch: "main", SSHKeyID: 9}}
	f.templates = []*models.Template{{ID: 7, Name: "Deploy", Playbook: "a.yml", App: "ansible"}}
	f.schedules = []*models.Schedule{{ID: 3, Name: "Nightly", CronFormat: "0 2 * * *"}}

	cfg := &ApplyConfig{
		Project:      "proj",
		Repositories: []RepoEntry{{Name: "r1", GitURL: "https://git/x.git"}},
		Templates:    []TemplateEntry{{Name: "Deploy", Playbook: "b.yml"}},
		Inventories:  []InventoryEntry{{Name: "inv-new", Type: "static"}},
		Schedules:    []ScheduleEntry{{Name: "Nightly", State: "absent"}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	cases := []struct {
		rt    ResourceType
		label string
		want  Action
	}{
		{ResourceRepository, "r1", ActionSkip},
		{ResourceTemplate, "Deploy", ActionUpdate},
		{ResourceInventory, "inv-new", ActionCreate},
		{ResourceSchedule, "Nightly", ActionDelete},
	}
	for _, tc := range cases {
		if got := findAction(t, plan, tc.rt, tc.label).Action; got != tc.want {
			t.Errorf("%s %q: action = %s, want %s", tc.rt, tc.label, got, tc.want)
		}
	}
}

func TestExecute_CreateAllResolvesNameRefs(t *testing.T) {
	f := newFakeSemaphore(t)
	autorun := false

	cfg := &ApplyConfig{
		Project: "newproj",
		Keys:    []KeyEntry{{Name: "k1", Type: "ssh", SSH: &SSHKeyData{Login: "u", PrivateKey: "material"}}},
		VariableGroups: []VariableGroupEntry{{
			Name:                 "vg1",
			Variables:            map[string]string{"a": "1"},
			EnvironmentVariables: map[string]string{"E": "2"},
			Secrets:              map[string]string{"s1": "shh"},
		}},
		Repositories: []RepoEntry{{Name: "r1", GitURL: "https://git/x.git", GitBranch: "main", SSHKey: "k1"}},
		Inventories:  []InventoryEntry{{Name: "inv1", Type: "static", Inventory: "localhost", SSHKey: "k1"}},
		Templates: []TemplateEntry{{
			Name: "tpl1", App: "ansible", Playbook: "p.yml",
			Repository: "r1", Inventory: "inv1", VariableGroup: "vg1",
			Autorun: &autorun,
		}},
		Schedules: []ScheduleEntry{{Name: "sch1", Template: "tpl1", CronFormat: "0 3 * * *"}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := findAction(t, plan, ResourceProject, "newproj").Action; got != ActionCreate {
		t.Fatalf("project action = %s, want create", got)
	}

	exec := NewExecutor(f.client(), cfg, recon)
	if errs := exec.Execute(context.Background(), plan); errs != 0 {
		t.Fatalf("Execute errors = %d, want 0", errs)
	}

	// Project 101, then key 102, vg 103, repo 104, inv 105, tpl 106, sched 107.
	pid := "101"

	var keyReq models.AccessKeyRequest
	f.lastBody(t, "POST /api/project/"+pid+"/keys", &keyReq)
	if keyReq.SSH == nil || keyReq.SSH.PrivateKey != "material" {
		t.Errorf("key request SSH = %+v, want private key material", keyReq.SSH)
	}

	var envReq models.EnvironmentRequest
	f.lastBody(t, "POST /api/project/"+pid+"/environment", &envReq)
	if envReq.JSON != `{"a":"1"}` || envReq.Env != `{"E":"2"}` {
		t.Errorf("env request JSON=%q Env=%q", envReq.JSON, envReq.Env)
	}
	if len(envReq.Secrets) != 1 || envReq.Secrets[0].Operation != "create" || envReq.Secrets[0].Type != "var" {
		t.Errorf("env request secrets = %+v, want one create/var", envReq.Secrets)
	}

	var repoReq models.RepositoryRequest
	f.lastBody(t, "POST /api/project/"+pid+"/repositories", &repoReq)
	if repoReq.SSHKeyID != 102 {
		t.Errorf("repo ssh_key_id = %d, want 102 (created key)", repoReq.SSHKeyID)
	}

	var tplReq models.TemplateRequest
	f.lastBody(t, "POST /api/project/"+pid+"/templates", &tplReq)
	if tplReq.RepositoryID != 104 || tplReq.InventoryID != 105 || tplReq.EnvironmentID != 103 {
		t.Errorf("template refs = repo %d, inv %d, env %d; want 104/105/103",
			tplReq.RepositoryID, tplReq.InventoryID, tplReq.EnvironmentID)
	}

	var schedReq models.ScheduleRequest
	f.lastBody(t, "POST /api/project/"+pid+"/schedules", &schedReq)
	if schedReq.TemplateID != 106 {
		t.Errorf("schedule template_id = %d, want 106 (created template)", schedReq.TemplateID)
	}
}

func TestExecute_TemplateUpdatePreservesUnmanagedFields(t *testing.T) {
	f := newFakeSemaphore(t)
	f.projects = []*models.Project{{ID: 1, Name: "proj"}}
	f.templates = []*models.Template{{
		ID: 7, Name: "Deploy", App: "ansible", Playbook: "a.yml",
		SurveyVars:     []*models.TemplateSurveyVar{{Name: "v1", Title: "Var 1"}},
		Vaults:         []*models.TemplateVault{{ID: 4, Name: "default", Type: "password"}},
		EnvironmentIds: []int64{9},
		TaskParams:     &models.TaskPrams{Message: "keep-me"},
	}}

	cfg := &ApplyConfig{
		Project:   "proj",
		Templates: []TemplateEntry{{Name: "Deploy", Description: "new desc"}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := findAction(t, plan, ResourceTemplate, "Deploy").Action; got != ActionUpdate {
		t.Fatalf("template action = %s, want update", got)
	}

	exec := NewExecutor(f.client(), cfg, recon)
	if errs := exec.Execute(context.Background(), plan); errs != 0 {
		t.Fatalf("Execute errors = %d, want 0", errs)
	}

	var req models.TemplateRequest
	f.lastBody(t, "PUT /api/project/1/templates/7", &req)

	// The historical bug class: an update wiping fields apply doesn't manage.
	if len(req.SurveyVars) != 1 || req.SurveyVars[0].Name != "v1" {
		t.Errorf("survey_vars = %+v, want the existing var preserved", req.SurveyVars)
	}
	if len(req.Vaults) != 1 || req.Vaults[0].ID != 4 {
		t.Errorf("vaults = %+v, want the existing vault preserved", req.Vaults)
	}
	if len(req.EnvironmentIds) != 1 || req.EnvironmentIds[0] != 9 {
		t.Errorf("environment_ids = %v, want [9] preserved", req.EnvironmentIds)
	}
	if req.TaskParams == nil || req.TaskParams.Message != "keep-me" {
		t.Errorf("task_params = %+v, want preserved", req.TaskParams)
	}
	// Merge semantics: unspecified fields keep server values, specified change.
	if req.Playbook != "a.yml" {
		t.Errorf("playbook = %q, want merged existing a.yml", req.Playbook)
	}
	if req.Description != "new desc" {
		t.Errorf("description = %q, want new desc", req.Description)
	}
}

func TestExecute_SecretUpdateMatchesExistingByName(t *testing.T) {
	f := newFakeSemaphore(t)
	f.projects = []*models.Project{{ID: 1, Name: "proj"}}
	f.envs = []*models.Environment{{
		ID: 3, Name: "vg", ProjectID: 1, JSON: "{}", Env: "{}",
		Secrets: []*models.EnvironmentSecret{{ID: 11, Name: "tok", Type: "var"}},
	}}

	cfg := &ApplyConfig{
		Project: "proj",
		VariableGroups: []VariableGroupEntry{{
			Name:    "vg",
			Secrets: map[string]string{"tok": "rotated", "fresh": "new"},
		}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if got := findAction(t, plan, ResourceVariableGroup, "vg").Action; got != ActionUpdate {
		t.Fatalf("variable group action = %s, want update (secrets always re-applied)", got)
	}

	exec := NewExecutor(f.client(), cfg, recon)
	if errs := exec.Execute(context.Background(), plan); errs != 0 {
		t.Fatalf("Execute errors = %d, want 0", errs)
	}

	var req models.EnvironmentRequest
	f.lastBody(t, "PUT /api/project/1/environment/3", &req)
	if len(req.Secrets) != 2 {
		t.Fatalf("secrets = %+v, want 2 entries", req.Secrets)
	}
	byName := map[string]*models.EnvironmentSecretRequest{}
	for _, s := range req.Secrets {
		byName[s.Name] = s
	}
	if s := byName["tok"]; s == nil || s.Operation != "update" || s.ID != 11 {
		t.Errorf("existing secret tok = %+v, want update with ID 11", s)
	}
	if s := byName["fresh"]; s == nil || s.Operation != "create" || s.ID != 0 {
		t.Errorf("new secret fresh = %+v, want create without ID", s)
	}
}

func TestExecute_PartialFailureContinues(t *testing.T) {
	f := newFakeSemaphore(t)
	// Project will be created with ID 101; fail its key creation.
	f.failOn["POST /api/project/101/keys"] = http.StatusInternalServerError

	cfg := &ApplyConfig{
		Project:        "newproj",
		Keys:           []KeyEntry{{Name: "k1", Type: "ssh", SSH: &SSHKeyData{PrivateKey: "m"}}},
		VariableGroups: []VariableGroupEntry{{Name: "vg1", Variables: map[string]string{"a": "1"}}},
		Repositories:   []RepoEntry{{Name: "r1", GitURL: "https://git/x.git", SSHKey: "k1"}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	exec := NewExecutor(f.client(), cfg, recon)
	// Two errors: the failed key create, and the dependent repo whose name
	// ref can no longer resolve (mustResolve guard — no more silent ID 0).
	if errs := exec.Execute(context.Background(), plan); errs != 2 {
		t.Fatalf("Execute errors = %d, want 2 (key + dependent repo)", errs)
	}

	// Independent resources still execute...
	if !f.sawRequest("POST /api/project/101/environment") {
		t.Error("variable group was not attempted after key failure")
	}
	// ...but the dependent repo must NOT be created with a dangling ref.
	if f.sawRequest("POST /api/project/101/repositories") {
		t.Error("repo was created despite its key ref failing to resolve")
	}
}

func TestExecute_FailFastStopsAtFirstError(t *testing.T) {
	f := newFakeSemaphore(t)
	f.failOn["POST /api/project/101/keys"] = http.StatusInternalServerError

	cfg := &ApplyConfig{
		Project:        "newproj",
		Keys:           []KeyEntry{{Name: "k1", Type: "ssh", SSH: &SSHKeyData{PrivateKey: "m"}}},
		VariableGroups: []VariableGroupEntry{{Name: "vg1", Variables: map[string]string{"a": "1"}}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	exec := NewExecutor(f.client(), cfg, recon)
	exec.SetFailFast(true)
	if errs := exec.Execute(context.Background(), plan); errs != 1 {
		t.Fatalf("Execute errors = %d, want 1", errs)
	}
	if f.sawRequest("POST /api/project/101/environment") {
		t.Error("--fail-fast must stop before the variable group")
	}
}

func TestExecute_CancelledContextStopsBeforeFirstAction(t *testing.T) {
	f := newFakeSemaphore(t)

	cfg := &ApplyConfig{
		Project: "newproj",
		Keys:    []KeyEntry{{Name: "k1", Type: "ssh", SSH: &SSHKeyData{PrivateKey: "m"}}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exec := NewExecutor(f.client(), cfg, recon)
	if errs := exec.Execute(ctx, plan); errs != 0 {
		t.Fatalf("Execute errors = %d, want 0 (an interrupt is not an apply error)", errs)
	}
	for _, req := range f.captured {
		switch {
		case strings.HasPrefix(req.key, "POST "),
			strings.HasPrefix(req.key, "PUT "),
			strings.HasPrefix(req.key, "DELETE "):
			t.Fatalf("mutating request %q sent despite cancelled context", req.key)
		}
	}
}

func TestBuildPlan_UnresolvedNameRefFailsPlan(t *testing.T) {
	f := newFakeSemaphore(t)

	cfg := &ApplyConfig{
		Project:      "newproj",
		Keys:         []KeyEntry{{Name: "real-key", Type: "ssh", SSH: &SSHKeyData{PrivateKey: "m"}}},
		Repositories: []RepoEntry{{Name: "r1", GitURL: "https://git/x.git", SSHKey: "real-kye"}}, // typo
	}

	recon := NewReconciler(f.client(), cfg)
	_, err := recon.BuildPlan()
	if err == nil {
		t.Fatal("BuildPlan should fail on a typo'd name ref")
	}
	for _, want := range []string{`key "real-kye"`, "repository r1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}

func TestBuildPlan_UpdateDescriptionListsChangedFields(t *testing.T) {
	f := newFakeSemaphore(t)
	f.projects = []*models.Project{{ID: 1, Name: "proj"}}
	f.templates = []*models.Template{{ID: 7, Name: "Deploy", App: "ansible", Playbook: "a.yml"}}

	cfg := &ApplyConfig{
		Project:   "proj",
		Templates: []TemplateEntry{{Name: "Deploy", Playbook: "b.yml", Description: "new"}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	a := findAction(t, plan, ResourceTemplate, "Deploy")
	if a.Action != ActionUpdate {
		t.Fatalf("action = %s, want update", a.Action)
	}
	for _, want := range []string{"playbook", "description"} {
		if !strings.Contains(a.Description, want) {
			t.Errorf("description %q missing %q", a.Description, want)
		}
	}
}

func TestExecute_ProjectDeleteTearsDownChildren(t *testing.T) {
	f := newFakeSemaphore(t)
	f.projects = []*models.Project{{ID: 1, Name: "proj"}}
	f.keys = []*models.AccessKey{{ID: 20, Name: "k"}}
	f.repos = []*models.Repository{{ID: 21, Name: "r"}}
	f.templates = []*models.Template{{ID: 22, Name: "tpl"}}
	f.schedules = []*models.Schedule{{ID: 23, Name: "s"}}

	// The config does NOT enumerate the children — the historical panic case:
	// executors used to index config slices for delete actions.
	cfg := &ApplyConfig{Project: "proj", ProjectState: "absent"}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	exec := NewExecutor(f.client(), cfg, recon)
	if errs := exec.Execute(context.Background(), plan); errs != 0 {
		t.Fatalf("Execute errors = %d, want 0", errs)
	}

	for _, key := range []string{
		"DELETE /api/project/1/schedules/23",
		"DELETE /api/project/1/templates/22",
		"DELETE /api/project/1/repositories/21",
		"DELETE /api/project/1/keys/20",
		"DELETE /api/project/1/",
	} {
		if !f.sawRequest(key) {
			t.Errorf("missing %s (captured: %v)", key, f.capturedKeys())
		}
	}
}

func TestBuildPlan_SchedulesEndpointMissing(t *testing.T) {
	f := newFakeSemaphore(t)
	f.projects = []*models.Project{{ID: 1, Name: "proj"}}
	// Pre-2.18 servers have no schedules-list endpoint.
	f.failOn["GET /api/project/1/schedules"] = http.StatusNotFound

	cfg := &ApplyConfig{
		Project:     "proj",
		Inventories: []InventoryEntry{{Name: "inv1", Type: "static"}},
		Schedules:   []ScheduleEntry{{Name: "sch1", TemplateID: 1, CronFormat: "* * * * *"}},
	}

	recon := NewReconciler(f.client(), cfg)
	plan, err := recon.BuildPlan()
	if err != nil {
		t.Fatalf("BuildPlan should tolerate a missing schedules API, got: %v", err)
	}

	// The rest of the plan still builds; schedules are left unmanaged.
	if got := findAction(t, plan, ResourceInventory, "inv1").Action; got != ActionCreate {
		t.Errorf("inventory action = %s, want create", got)
	}
	for _, a := range plan.Actions {
		if a.Type == ResourceSchedule {
			t.Errorf("unexpected schedule action %s %q — schedules must be unmanaged on 404", a.Action, a.Label)
		}
	}
}
