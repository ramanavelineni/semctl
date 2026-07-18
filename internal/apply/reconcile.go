package apply

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	apiclient "github.com/ramanavelineni/semctl/pkg/semapi/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
)

// Reconciler compares desired config with existing Semaphore state.
type Reconciler struct {
	client    *apiclient.Semapi
	config    *ApplyConfig
	projectID int64

	// Name-to-ID maps populated during reconciliation.
	// Keys are lowercased names (lookups are case-insensitive).
	KeyIDByName       map[string]int64
	VarGroupIDByName  map[string]int64
	RepoIDByName      map[string]int64
	InventoryIDByName map[string]int64
	TemplateIDByName  map[string]int64

	// Existing resources by ID, populated during reconciliation. The executor
	// merges config entries over these so that fields omitted from the config
	// keep their current server-side values on update.
	ExistingRepoByID      map[int64]*models.Repository
	ExistingInventoryByID map[int64]*models.Inventory
	ExistingTemplateByID  map[int64]*models.Template
	ExistingScheduleByID  map[int64]*models.Schedule
}

// NewReconciler creates a new reconciler.
func NewReconciler(client *apiclient.Semapi, config *ApplyConfig) *Reconciler {
	return &Reconciler{
		client:            client,
		config:            config,
		KeyIDByName:       make(map[string]int64),
		VarGroupIDByName:  make(map[string]int64),
		RepoIDByName:      make(map[string]int64),
		InventoryIDByName: make(map[string]int64),
		TemplateIDByName:  make(map[string]int64),

		ExistingRepoByID:      make(map[int64]*models.Repository),
		ExistingInventoryByID: make(map[int64]*models.Inventory),
		ExistingTemplateByID:  make(map[int64]*models.Template),
		ExistingScheduleByID:  make(map[int64]*models.Schedule),
	}
}

// ProjectID returns the resolved project ID.
func (r *Reconciler) ProjectID() int64 {
	return r.projectID
}

// SetProjectID sets the project ID (used after creation).
func (r *Reconciler) SetProjectID(id int64) {
	r.projectID = id
}

// BuildPlan resolves the project and builds a full reconciliation plan.
func (r *Reconciler) BuildPlan() (*Plan, error) {
	plan := &Plan{}

	// Step 1: Resolve project
	projectAction, err := r.resolveProject()
	if err != nil {
		return nil, fmt.Errorf("resolving project: %w", err)
	}
	plan.Actions = append(plan.Actions, projectAction)

	// If project is new, everything is a create
	if projectAction.Action == ActionCreate {
		r.buildAllAsCreate(plan)
		if err := r.validateNameRefs(); err != nil {
			return nil, err
		}
		return plan, nil
	}

	// If project is being deleted, fetch all children and mark them for deletion
	if projectAction.Action == ActionDelete {
		if err := r.buildAllAsDelete(plan); err != nil {
			return nil, fmt.Errorf("building delete plan: %w", err)
		}
		return plan, nil
	}

	// Step 2: Reconcile in dependency order
	if err := r.reconcileKeys(plan); err != nil {
		return nil, fmt.Errorf("reconciling keys: %w", err)
	}
	if err := r.reconcileVariableGroups(plan); err != nil {
		return nil, fmt.Errorf("reconciling variable groups: %w", err)
	}
	if err := r.reconcileRepositories(plan); err != nil {
		return nil, fmt.Errorf("reconciling repositories: %w", err)
	}
	if err := r.reconcileInventories(plan); err != nil {
		return nil, fmt.Errorf("reconciling inventories: %w", err)
	}
	if err := r.reconcileTemplates(plan); err != nil {
		return nil, fmt.Errorf("reconciling templates: %w", err)
	}

	if err := r.reconcileSchedules(plan); err != nil {
		return nil, fmt.Errorf("reconciling schedules: %w", err)
	}

	if err := r.validateNameRefs(); err != nil {
		return nil, err
	}

	return plan, nil
}

// reconcileSchedules diffs config schedules against the server by name.
// Must run after reconcileTemplates (template name refs resolve via the maps).
func (r *Reconciler) reconcileSchedules(plan *Plan) error {
	params := schedule.NewGetProjectProjectIDSchedulesParams()
	params.ProjectID = r.projectID
	resp, err := r.client.Schedule.GetProjectProjectIDSchedules(params, nil)
	if err != nil {
		// Older Semaphore servers have no schedules-list endpoint; leave
		// schedules unmanaged instead of failing the whole plan.
		if client.IsNotFound(err) {
			msg := "Server has no schedules API (Semaphore < 2.18?) — schedules left unmanaged."
			if len(r.config.Schedules) > 0 {
				msg = fmt.Sprintf("Server has no schedules API (Semaphore < 2.18?) — the %d schedule(s) in this config will NOT be applied.", len(r.config.Schedules))
			}
			style.Warning(msg)
			return nil
		}
		return err
	}

	existing := resp.GetPayload()
	for _, s := range existing {
		r.ExistingScheduleByID[s.ID] = s
	}

	for i, entry := range r.config.Schedules {
		matches := findSchedulesByName(existing, entry.Name)

		if entry.State == "absent" {
			// Delete every match: duplicates may exist from versions where
			// schedules could not be reconciled and re-applies created copies.
			for _, m := range matches {
				plan.Actions = append(plan.Actions, ResourceAction{
					Type:       ResourceSchedule,
					Action:     ActionDelete,
					Label:      entry.Name,
					ExistingID: m.ID,
					Index:      i,
				})
			}
			continue
		}

		if len(matches) == 0 {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:   ResourceSchedule,
				Action: ActionCreate,
				Label:  entry.Name,
				Index:  i,
			})
			continue
		}

		first := matches[0]
		desc := ""
		if len(matches) > 1 {
			desc = fmt.Sprintf("%d schedules share this name; managing ID %d — set state: absent once to delete all copies, then re-apply", len(matches), first.ID)
		}

		if fields := r.scheduleChangedFields(entry, first); len(fields) > 0 {
			fieldDesc := strings.Join(fields, ", ")
			if desc != "" {
				fieldDesc += "; " + desc
			}
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceSchedule,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: fieldDesc,
				ExistingID:  first.ID,
				Index:       i,
			})
		} else {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceSchedule,
				Action:      ActionSkip,
				Label:       entry.Name,
				Description: desc,
				ExistingID:  first.ID,
				Index:       i,
			})
		}
	}
	return nil
}

// resolveProject finds an existing project by name or plans a create/delete.
func (r *Reconciler) resolveProject() (ResourceAction, error) {
	resp, err := r.client.Project.GetProjects(project.NewGetProjectsParams(), nil)
	if err != nil {
		return ResourceAction{}, fmt.Errorf("listing projects: %w", err)
	}

	for _, p := range resp.GetPayload() {
		if strings.EqualFold(p.Name, r.config.Project) {
			r.projectID = p.ID

			if r.config.ProjectState == "absent" {
				return ResourceAction{
					Type:       ResourceProject,
					Action:     ActionDelete,
					Label:      p.Name,
					ExistingID: p.ID,
				}, nil
			}

			return ResourceAction{
				Type:       ResourceProject,
				Action:     ActionSkip,
				Label:      p.Name,
				ExistingID: p.ID,
			}, nil
		}
	}

	if r.config.ProjectState == "absent" {
		return ResourceAction{
			Type:        ResourceProject,
			Action:      ActionSkip,
			Label:       r.config.Project,
			Description: "already absent",
		}, nil
	}

	return ResourceAction{
		Type:   ResourceProject,
		Action: ActionCreate,
		Label:  r.config.Project,
	}, nil
}

// buildAllAsCreate adds ActionCreate for all resources when project is new.
func (r *Reconciler) buildAllAsCreate(plan *Plan) {
	for i, k := range r.config.Keys {
		if k.State == "absent" {
			continue
		}
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:   ResourceKey,
			Action: ActionCreate,
			Label:  k.Name,
			Index:  i,
		})
	}
	for i, vg := range r.config.VariableGroups {
		if vg.State == "absent" {
			continue
		}
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:   ResourceVariableGroup,
			Action: ActionCreate,
			Label:  vg.Name,
			Index:  i,
		})
	}
	for i, repo := range r.config.Repositories {
		if repo.State == "absent" {
			continue
		}
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:   ResourceRepository,
			Action: ActionCreate,
			Label:  repo.Name,
			Index:  i,
		})
	}
	for i, inv := range r.config.Inventories {
		if inv.State == "absent" {
			continue
		}
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:   ResourceInventory,
			Action: ActionCreate,
			Label:  inv.Name,
			Index:  i,
		})
	}
	for i, t := range r.config.Templates {
		if t.State == "absent" {
			continue
		}
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:   ResourceTemplate,
			Action: ActionCreate,
			Label:  t.Name,
			Index:  i,
		})
	}
	for i, s := range r.config.Schedules {
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:   ResourceSchedule,
			Action: ActionCreate,
			Label:  s.Name,
			Index:  i,
		})
	}
}

// buildAllAsDelete fetches all existing child resources and marks them for deletion.
func (r *Reconciler) buildAllAsDelete(plan *Plan) error {
	pid := r.projectID

	// Schedules (deleted first — they reference templates)
	schedResp, err := r.client.Schedule.GetProjectProjectIDSchedules(
		schedule.NewGetProjectProjectIDSchedulesParams().WithProjectID(pid), nil)
	if err != nil {
		return fmt.Errorf("listing schedules: %w", err)
	}
	for _, s := range schedResp.GetPayload() {
		label := s.Name
		if label == "" {
			label = fmt.Sprintf("schedule %d", s.ID)
		}
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:       ResourceSchedule,
			Action:     ActionDelete,
			Label:      label,
			ExistingID: s.ID,
		})
	}

	// Templates
	tplResp, err := r.client.Template.GetProjectProjectIDTemplates(
		template.NewGetProjectProjectIDTemplatesParams().WithProjectID(pid), nil)
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}
	for _, t := range tplResp.GetPayload() {
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:       ResourceTemplate,
			Action:     ActionDelete,
			Label:      t.Name,
			ExistingID: t.ID,
		})
	}

	// Inventories
	invResp, err := r.client.Inventory.GetProjectProjectIDInventory(
		inventory.NewGetProjectProjectIDInventoryParams().WithProjectID(pid), nil)
	if err != nil {
		return fmt.Errorf("listing inventories: %w", err)
	}
	for _, inv := range invResp.GetPayload() {
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:       ResourceInventory,
			Action:     ActionDelete,
			Label:      inv.Name,
			ExistingID: inv.ID,
		})
	}

	// Repositories
	repoResp, err := r.client.Repository.GetProjectProjectIDRepositories(
		repository.NewGetProjectProjectIDRepositoriesParams().WithProjectID(pid), nil)
	if err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}
	for _, repo := range repoResp.GetPayload() {
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:       ResourceRepository,
			Action:     ActionDelete,
			Label:      repo.Name,
			ExistingID: repo.ID,
		})
	}

	// Environments
	envResp, err := r.client.VariableGroup.GetProjectProjectIDEnvironment(
		variable_group.NewGetProjectProjectIDEnvironmentParams().WithProjectID(pid), nil)
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}
	for _, env := range envResp.GetPayload() {
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:       ResourceVariableGroup,
			Action:     ActionDelete,
			Label:      env.Name,
			ExistingID: env.ID,
		})
	}

	// Keys
	keyResp, err := r.client.KeyStore.GetProjectProjectIDKeys(
		key_store.NewGetProjectProjectIDKeysParams().WithProjectID(pid), nil)
	if err != nil {
		return fmt.Errorf("listing keys: %w", err)
	}
	for _, k := range keyResp.GetPayload() {
		plan.Actions = append(plan.Actions, ResourceAction{
			Type:       ResourceKey,
			Action:     ActionDelete,
			Label:      k.Name,
			ExistingID: k.ID,
		})
	}

	return nil
}

func (r *Reconciler) reconcileKeys(plan *Plan) error {
	params := key_store.NewGetProjectProjectIDKeysParams()
	params.ProjectID = r.projectID
	resp, err := r.client.KeyStore.GetProjectProjectIDKeys(params, nil)
	if err != nil {
		return err
	}

	existing := resp.GetPayload()
	for _, k := range existing {
		r.KeyIDByName[strings.ToLower(k.Name)] = k.ID
	}

	for i, entry := range r.config.Keys {
		existingKey := findKeyByName(existing, entry.Name)

		if entry.State == "absent" {
			if existingKey != nil {
				plan.Actions = append(plan.Actions, ResourceAction{
					Type:       ResourceKey,
					Action:     ActionDelete,
					Label:      entry.Name,
					ExistingID: existingKey.ID,
					Index:      i,
				})
			}
			continue
		}

		if existingKey == nil {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:   ResourceKey,
				Action: ActionCreate,
				Label:  entry.Name,
				Index:  i,
			})
		} else if fields := keyChangedFields(entry, existingKey); len(fields) > 0 {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceKey,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: strings.Join(fields, ", "),
				ExistingID:  existingKey.ID,
				Index:       i,
			})
		} else {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceKey,
				Action:     ActionSkip,
				Label:      entry.Name,
				ExistingID: existingKey.ID,
				Index:      i,
			})
		}
	}
	return nil
}

func (r *Reconciler) reconcileVariableGroups(plan *Plan) error {
	params := variable_group.NewGetProjectProjectIDEnvironmentParams()
	params.ProjectID = r.projectID
	resp, err := r.client.VariableGroup.GetProjectProjectIDEnvironment(params, nil)
	if err != nil {
		return err
	}

	existing := resp.GetPayload()
	for _, e := range existing {
		r.VarGroupIDByName[strings.ToLower(e.Name)] = e.ID
	}

	for i, entry := range r.config.VariableGroups {
		existingEnv := findEnvByName(existing, entry.Name)

		if entry.State == "absent" {
			if existingEnv != nil {
				plan.Actions = append(plan.Actions, ResourceAction{
					Type:       ResourceVariableGroup,
					Action:     ActionDelete,
					Label:      entry.Name,
					ExistingID: existingEnv.ID,
					Index:      i,
				})
			}
			continue
		}

		if existingEnv == nil {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:   ResourceVariableGroup,
				Action: ActionCreate,
				Label:  entry.Name,
				Index:  i,
			})
		} else if fields := varGroupChangedFields(entry, existingEnv); len(fields) > 0 {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceVariableGroup,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: strings.Join(fields, ", "),
				ExistingID:  existingEnv.ID,
				Index:       i,
			})
		} else {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceVariableGroup,
				Action:     ActionSkip,
				Label:      entry.Name,
				ExistingID: existingEnv.ID,
				Index:      i,
			})
		}
	}
	return nil
}

func (r *Reconciler) reconcileRepositories(plan *Plan) error {
	params := repository.NewGetProjectProjectIDRepositoriesParams()
	params.ProjectID = r.projectID
	resp, err := r.client.Repository.GetProjectProjectIDRepositories(params, nil)
	if err != nil {
		return err
	}

	existing := resp.GetPayload()
	for _, repo := range existing {
		r.RepoIDByName[strings.ToLower(repo.Name)] = repo.ID
		r.ExistingRepoByID[repo.ID] = repo
	}

	for i, entry := range r.config.Repositories {
		existingRepo := findRepoByName(existing, entry.Name)

		if entry.State == "absent" {
			if existingRepo != nil {
				plan.Actions = append(plan.Actions, ResourceAction{
					Type:       ResourceRepository,
					Action:     ActionDelete,
					Label:      entry.Name,
					ExistingID: existingRepo.ID,
					Index:      i,
				})
			}
			continue
		}

		if existingRepo == nil {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:   ResourceRepository,
				Action: ActionCreate,
				Label:  entry.Name,
				Index:  i,
			})
		} else if fields := r.repoChangedFields(entry, existingRepo); len(fields) > 0 {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceRepository,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: strings.Join(fields, ", "),
				ExistingID:  existingRepo.ID,
				Index:       i,
			})
		} else {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceRepository,
				Action:     ActionSkip,
				Label:      entry.Name,
				ExistingID: existingRepo.ID,
				Index:      i,
			})
		}
	}
	return nil
}

func (r *Reconciler) reconcileInventories(plan *Plan) error {
	params := inventory.NewGetProjectProjectIDInventoryParams()
	params.ProjectID = r.projectID
	resp, err := r.client.Inventory.GetProjectProjectIDInventory(params, nil)
	if err != nil {
		return err
	}

	existing := resp.GetPayload()
	for _, inv := range existing {
		r.InventoryIDByName[strings.ToLower(inv.Name)] = inv.ID
		r.ExistingInventoryByID[inv.ID] = inv
	}

	for i, entry := range r.config.Inventories {
		existingInv := findInventoryByName(existing, entry.Name)

		if entry.State == "absent" {
			if existingInv != nil {
				plan.Actions = append(plan.Actions, ResourceAction{
					Type:       ResourceInventory,
					Action:     ActionDelete,
					Label:      entry.Name,
					ExistingID: existingInv.ID,
					Index:      i,
				})
			}
			continue
		}

		if existingInv == nil {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:   ResourceInventory,
				Action: ActionCreate,
				Label:  entry.Name,
				Index:  i,
			})
		} else if fields := r.inventoryChangedFields(entry, existingInv); len(fields) > 0 {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceInventory,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: strings.Join(fields, ", "),
				ExistingID:  existingInv.ID,
				Index:       i,
			})
		} else {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceInventory,
				Action:     ActionSkip,
				Label:      entry.Name,
				ExistingID: existingInv.ID,
				Index:      i,
			})
		}
	}
	return nil
}

func (r *Reconciler) reconcileTemplates(plan *Plan) error {
	params := template.NewGetProjectProjectIDTemplatesParams()
	params.ProjectID = r.projectID
	resp, err := r.client.Template.GetProjectProjectIDTemplates(params, nil)
	if err != nil {
		return err
	}

	existing := resp.GetPayload()
	for _, t := range existing {
		r.TemplateIDByName[strings.ToLower(t.Name)] = t.ID
		r.ExistingTemplateByID[t.ID] = t
	}

	for i, entry := range r.config.Templates {
		existingTpl := findTemplateByName(existing, entry.Name)

		if entry.State == "absent" {
			if existingTpl != nil {
				plan.Actions = append(plan.Actions, ResourceAction{
					Type:       ResourceTemplate,
					Action:     ActionDelete,
					Label:      entry.Name,
					ExistingID: existingTpl.ID,
					Index:      i,
				})
			}
			continue
		}

		if existingTpl == nil {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:   ResourceTemplate,
				Action: ActionCreate,
				Label:  entry.Name,
				Index:  i,
			})
		} else if fields := r.templateChangedFields(entry, existingTpl); len(fields) > 0 {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceTemplate,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: strings.Join(fields, ", "),
				ExistingID:  existingTpl.ID,
				Index:       i,
			})
		} else {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceTemplate,
				Action:     ActionSkip,
				Label:      entry.Name,
				ExistingID: existingTpl.ID,
				Index:      i,
			})
		}
	}
	return nil
}

// NeedsUpdate helpers: each xChangedFields returns the names of fields the
// config would change (shown in the plan); xNeedsUpdate wraps it as a bool.

// keyChangedFields: secrets are never returned by the API, so specifying
// them always re-applies.
func keyChangedFields(entry KeyEntry, existing *models.AccessKey) []string {
	var fields []string
	if entry.Type != existing.Type {
		fields = append(fields, "type")
	}
	if (entry.SSH != nil && entry.SSH.PrivateKey != "") ||
		(entry.LoginPassword != nil && entry.LoginPassword.Password != "") {
		fields = append(fields, "secrets (always re-applied)")
	}
	return fields
}

func keyNeedsUpdate(entry KeyEntry, existing *models.AccessKey) bool {
	return len(keyChangedFields(entry, existing)) > 0
}

// jsonVarsEqual compares a config variables map against the server-side JSON
// string semantically, so key order and whitespace differences don't produce
// phantom updates. Unparseable server JSON counts as different (safe: the
// PUT is idempotent).
func jsonVarsEqual(entryVars map[string]string, existingJSON string) bool {
	var existing map[string]any
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return false
	}
	if len(existing) != len(entryVars) {
		return false
	}
	for k, v := range entryVars {
		ev, ok := existing[k]
		if !ok {
			return false
		}
		es, ok := ev.(string)
		if !ok || es != v {
			return false
		}
	}
	return true
}

func varGroupChangedFields(entry VariableGroupEntry, existing *models.Environment) []string {
	var fields []string
	if len(entry.Variables) > 0 && !jsonVarsEqual(entry.Variables, existing.JSON) {
		fields = append(fields, "variables")
	}
	if len(entry.EnvironmentVariables) > 0 && !jsonVarsEqual(entry.EnvironmentVariables, existing.Env) {
		fields = append(fields, "environment_variables")
	}
	if len(entry.Secrets) > 0 || len(entry.SecretEnvironmentVariables) > 0 {
		fields = append(fields, "secrets (always re-applied)")
	}
	return fields
}

func varGroupNeedsUpdate(entry VariableGroupEntry, existing *models.Environment) bool {
	return len(varGroupChangedFields(entry, existing)) > 0
}

func (r *Reconciler) repoChangedFields(entry RepoEntry, existing *models.Repository) []string {
	var fields []string
	if entry.GitURL != "" && entry.GitURL != existing.GitURL {
		fields = append(fields, "git_url")
	}
	if entry.GitBranch != "" && entry.GitBranch != existing.GitBranch {
		fields = append(fields, "git_branch")
	}
	resolvedKeyID := r.resolveKeyID(entry.SSHKey, entry.SSHKeyID)
	if resolvedKeyID != 0 && resolvedKeyID != existing.SSHKeyID {
		fields = append(fields, "ssh_key")
	}
	return fields
}

func (r *Reconciler) repoNeedsUpdate(entry RepoEntry, existing *models.Repository) bool {
	return len(r.repoChangedFields(entry, existing)) > 0
}

func (r *Reconciler) inventoryChangedFields(entry InventoryEntry, existing *models.Inventory) []string {
	var fields []string
	if entry.Type != "" && entry.Type != existing.Type {
		fields = append(fields, "type")
	}
	if entry.Inventory != "" && entry.Inventory != existing.Inventory {
		fields = append(fields, "inventory")
	}
	if id := r.resolveKeyID(entry.SSHKey, entry.SSHKeyID); id != 0 && id != existing.SSHKeyID {
		fields = append(fields, "ssh_key")
	}
	if id := r.resolveKeyID(entry.BecomeKey, entry.BecomeKeyID); id != 0 && id != existing.BecomeKeyID {
		fields = append(fields, "become_key")
	}
	if id := r.resolveRepoID(entry.Repository, entry.RepositoryID); id != 0 && id != existing.RepositoryID {
		fields = append(fields, "repository")
	}
	return fields
}

func (r *Reconciler) templateChangedFields(entry TemplateEntry, existing *models.Template) []string {
	var fields []string
	strChanges := []struct {
		name          string
		entryVal, cur string
	}{
		{"type", entry.Type, existing.Type},
		{"app", entry.App, existing.App},
		{"playbook", entry.Playbook, existing.Playbook},
		{"description", entry.Description, existing.Description},
		{"git_branch", entry.GitBranch, existing.GitBranch},
		{"arguments", entry.Arguments, existing.Arguments},
		{"start_version", entry.StartVersion, existing.StartVersion},
	}
	for _, c := range strChanges {
		if c.entryVal != "" && c.entryVal != c.cur {
			fields = append(fields, c.name)
		}
	}
	boolChanges := []struct {
		name     string
		entryVal *bool
		cur      bool
	}{
		{"autorun", entry.Autorun, existing.Autorun},
		{"suppress_success_alerts", entry.SuppressSuccessAlerts, existing.SuppressSuccessAlerts},
		{"allow_override_args_in_task", entry.AllowOverrideArgsInTask, existing.AllowOverrideArgsInTask},
	}
	for _, c := range boolChanges {
		if c.entryVal != nil && *c.entryVal != c.cur {
			fields = append(fields, c.name)
		}
	}
	if id := r.resolveRepoID(entry.Repository, entry.RepositoryID); id != 0 && id != existing.RepositoryID {
		fields = append(fields, "repository")
	}
	if id := r.resolveVarGroupID(entry.VariableGroup, entry.EnvironmentID); id != 0 && id != existing.EnvironmentID {
		fields = append(fields, "variable_group")
	}
	if id := r.resolveInventoryID(entry.Inventory, entry.InventoryID); id != 0 && id != existing.InventoryID {
		fields = append(fields, "inventory")
	}
	if id := r.resolveTemplateID(entry.BuildTemplate, entry.BuildTemplateID); id != 0 && id != existing.BuildTemplateID {
		fields = append(fields, "build_template")
	}
	if entry.ViewID != 0 && entry.ViewID != existing.ViewID {
		fields = append(fields, "view_id")
	}
	return fields
}

func (r *Reconciler) templateNeedsUpdate(entry TemplateEntry, existing *models.Template) bool {
	return len(r.templateChangedFields(entry, existing)) > 0
}

// scheduleChangedFields returns the specified schedule fields that differ
// from the server-side schedule.
func (r *Reconciler) scheduleChangedFields(entry ScheduleEntry, existing *models.Schedule) []string {
	var fields []string
	if entry.CronFormat != "" && entry.CronFormat != existing.CronFormat {
		fields = append(fields, "cron_format")
	}
	if id := r.resolveTemplateID(entry.Template, entry.TemplateID); id != 0 && id != existing.TemplateID {
		fields = append(fields, "template")
	}
	if entry.Active != nil && *entry.Active != existing.Active {
		fields = append(fields, "active")
	}
	return fields
}

func (r *Reconciler) scheduleNeedsUpdate(entry ScheduleEntry, existing *models.Schedule) bool {
	return len(r.scheduleChangedFields(entry, existing)) > 0
}

// validateNameRefs fails the plan when a name reference points at a resource
// that neither exists server-side nor is defined in this config. Previously
// such refs silently resolved to ID 0 and created broken resources.
func (r *Reconciler) validateNameRefs() error {
	cfgNames := func(names []string) map[string]bool {
		set := make(map[string]bool, len(names))
		for _, n := range names {
			set[strings.ToLower(n)] = true
		}
		return set
	}
	var keyNames, repoNames, vgNames, invNames, tplNames []string
	for _, k := range r.config.Keys {
		if k.State != "absent" {
			keyNames = append(keyNames, k.Name)
		}
	}
	for _, rp := range r.config.Repositories {
		if rp.State != "absent" {
			repoNames = append(repoNames, rp.Name)
		}
	}
	for _, vg := range r.config.VariableGroups {
		if vg.State != "absent" {
			vgNames = append(vgNames, vg.Name)
		}
	}
	for _, inv := range r.config.Inventories {
		if inv.State != "absent" {
			invNames = append(invNames, inv.Name)
		}
	}
	for _, t := range r.config.Templates {
		if t.State != "absent" {
			tplNames = append(tplNames, t.Name)
		}
	}
	cfgKeys, cfgRepos, cfgVGs, cfgInvs, cfgTpls :=
		cfgNames(keyNames), cfgNames(repoNames), cfgNames(vgNames), cfgNames(invNames), cfgNames(tplNames)

	var errs []string
	check := func(kind, name, by string, serverMap map[string]int64, inConfig map[string]bool) {
		if name == "" {
			return
		}
		l := strings.ToLower(name)
		if _, ok := serverMap[l]; ok {
			return
		}
		if inConfig[l] {
			return
		}
		errs = append(errs, fmt.Sprintf("%s %q (referenced by %s) exists neither on the server nor in this config", kind, name, by))
	}

	for _, rp := range r.config.Repositories {
		if rp.State == "absent" {
			continue
		}
		check("key", rp.SSHKey, "repository "+rp.Name, r.KeyIDByName, cfgKeys)
	}
	for _, inv := range r.config.Inventories {
		if inv.State == "absent" {
			continue
		}
		check("key", inv.SSHKey, "inventory "+inv.Name, r.KeyIDByName, cfgKeys)
		check("key", inv.BecomeKey, "inventory "+inv.Name, r.KeyIDByName, cfgKeys)
		check("repository", inv.Repository, "inventory "+inv.Name, r.RepoIDByName, cfgRepos)
	}
	for _, t := range r.config.Templates {
		if t.State == "absent" {
			continue
		}
		by := "template " + t.Name
		check("repository", t.Repository, by, r.RepoIDByName, cfgRepos)
		check("variable group", t.VariableGroup, by, r.VarGroupIDByName, cfgVGs)
		check("inventory", t.Inventory, by, r.InventoryIDByName, cfgInvs)
		check("template", t.BuildTemplate, by, r.TemplateIDByName, cfgTpls)
	}
	for _, sc := range r.config.Schedules {
		if sc.State == "absent" {
			continue
		}
		check("template", sc.Template, "schedule "+sc.Name, r.TemplateIDByName, cfgTpls)
	}

	if len(errs) > 0 {
		return fmt.Errorf("unresolved references:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// mustResolve errors when a name reference failed to resolve to an ID at
// execution time (e.g. its creation failed earlier in this run) instead of
// letting a zero ID create a broken resource.
func mustResolve(kind, name string, id int64) (int64, error) {
	if name != "" && id == 0 {
		return 0, fmt.Errorf("%s %q did not resolve to an ID — its creation may have failed earlier in this run", kind, name)
	}
	return id, nil
}

// Cross-ref resolution helpers

func (r *Reconciler) resolveKeyID(name string, explicitID int64) int64 {
	if explicitID != 0 {
		return explicitID
	}
	if name != "" {
		if id, ok := r.KeyIDByName[strings.ToLower(name)]; ok {
			return id
		}
	}
	return 0
}

func (r *Reconciler) resolveVarGroupID(name string, explicitID int64) int64 {
	if explicitID != 0 {
		return explicitID
	}
	if name != "" {
		if id, ok := r.VarGroupIDByName[strings.ToLower(name)]; ok {
			return id
		}
	}
	return 0
}

func (r *Reconciler) resolveRepoID(name string, explicitID int64) int64 {
	if explicitID != 0 {
		return explicitID
	}
	if name != "" {
		if id, ok := r.RepoIDByName[strings.ToLower(name)]; ok {
			return id
		}
	}
	return 0
}

func (r *Reconciler) resolveInventoryID(name string, explicitID int64) int64 {
	if explicitID != 0 {
		return explicitID
	}
	if name != "" {
		if id, ok := r.InventoryIDByName[strings.ToLower(name)]; ok {
			return id
		}
	}
	return 0
}

func (r *Reconciler) resolveTemplateID(name string, explicitID int64) int64 {
	if explicitID != 0 {
		return explicitID
	}
	if name != "" {
		if id, ok := r.TemplateIDByName[strings.ToLower(name)]; ok {
			return id
		}
	}
	return 0
}

// Find helpers (case-insensitive name match)

func findKeyByName(keys []*models.AccessKey, name string) *models.AccessKey {
	for _, k := range keys {
		if strings.EqualFold(k.Name, name) {
			return k
		}
	}
	return nil
}

func findEnvByName(envs []*models.Environment, name string) *models.Environment {
	for _, e := range envs {
		if strings.EqualFold(e.Name, name) {
			return e
		}
	}
	return nil
}

func findRepoByName(repos []*models.Repository, name string) *models.Repository {
	for _, r := range repos {
		if strings.EqualFold(r.Name, name) {
			return r
		}
	}
	return nil
}

func findInventoryByName(invs []*models.Inventory, name string) *models.Inventory {
	for _, inv := range invs {
		if strings.EqualFold(inv.Name, name) {
			return inv
		}
	}
	return nil
}

func findTemplateByName(templates []*models.Template, name string) *models.Template {
	for _, t := range templates {
		if strings.EqualFold(t.Name, name) {
			return t
		}
	}
	return nil
}

// findSchedulesByName returns ALL schedules matching the name: duplicates can
// exist server-side (schedule names are not unique in Semaphore).
func findSchedulesByName(schedules []*models.Schedule, name string) []*models.Schedule {
	var result []*models.Schedule
	for _, s := range schedules {
		if strings.EqualFold(s.Name, name) {
			result = append(result, s)
		}
	}
	return result
}
