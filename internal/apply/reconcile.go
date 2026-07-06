package apply

import (
	"fmt"
	"strings"

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

	return plan, nil
}

// reconcileSchedules diffs config schedules against the server by name.
// Must run after reconcileTemplates (template name refs resolve via the maps).
func (r *Reconciler) reconcileSchedules(plan *Plan) error {
	params := schedule.NewGetProjectProjectIDSchedulesParams()
	params.ProjectID = r.projectID
	resp, err := r.client.Schedule.GetProjectProjectIDSchedules(params, nil)
	if err != nil {
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

		if r.scheduleNeedsUpdate(entry, first) {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceSchedule,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: desc,
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
		} else if keyNeedsUpdate(entry, existingKey) {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:        ResourceKey,
				Action:      ActionUpdate,
				Label:       entry.Name,
				Description: "secrets always re-applied",
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
		} else if varGroupNeedsUpdate(entry, existingEnv) {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceVariableGroup,
				Action:     ActionUpdate,
				Label:      entry.Name,
				ExistingID: existingEnv.ID,
				Index:      i,
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
		} else if r.repoNeedsUpdate(entry, existingRepo) {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceRepository,
				Action:     ActionUpdate,
				Label:      entry.Name,
				ExistingID: existingRepo.ID,
				Index:      i,
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
		} else if r.inventoryNeedsUpdate(entry, existingInv) {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceInventory,
				Action:     ActionUpdate,
				Label:      entry.Name,
				ExistingID: existingInv.ID,
				Index:      i,
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
		} else if r.templateNeedsUpdate(entry, existingTpl) {
			plan.Actions = append(plan.Actions, ResourceAction{
				Type:       ResourceTemplate,
				Action:     ActionUpdate,
				Label:      entry.Name,
				ExistingID: existingTpl.ID,
				Index:      i,
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

// NeedsUpdate helpers

// keyNeedsUpdate returns true if the key type differs or secrets are specified.
// Secrets are never returned by the API, so we always update if SSH/login_password data is provided.
func keyNeedsUpdate(entry KeyEntry, existing *models.AccessKey) bool {
	if entry.Type != existing.Type {
		return true
	}
	if entry.SSH != nil && entry.SSH.PrivateKey != "" {
		return true
	}
	if entry.LoginPassword != nil && entry.LoginPassword.Password != "" {
		return true
	}
	return false
}

func varGroupNeedsUpdate(entry VariableGroupEntry, existing *models.Environment) bool {
	varsJSON := VarsToJSON(entry.Variables)
	if varsJSON != "{}" && varsJSON != existing.JSON {
		return true
	}
	envStr := EnvVarsToJSON(entry.EnvironmentVariables)
	if envStr != "{}" && envStr != existing.Env {
		return true
	}
	if len(entry.Secrets) > 0 || len(entry.SecretEnvironmentVariables) > 0 {
		return true // secret values not returned by API, always re-apply
	}
	return false
}

func (r *Reconciler) repoNeedsUpdate(entry RepoEntry, existing *models.Repository) bool {
	if entry.GitURL != "" && entry.GitURL != existing.GitURL {
		return true
	}
	if entry.GitBranch != "" && entry.GitBranch != existing.GitBranch {
		return true
	}
	resolvedKeyID := r.resolveKeyID(entry.SSHKey, entry.SSHKeyID)
	if resolvedKeyID != 0 && resolvedKeyID != existing.SSHKeyID {
		return true
	}
	return false
}

func (r *Reconciler) inventoryNeedsUpdate(entry InventoryEntry, existing *models.Inventory) bool {
	if entry.Type != "" && entry.Type != existing.Type {
		return true
	}
	if entry.Inventory != "" && entry.Inventory != existing.Inventory {
		return true
	}
	resolvedKeyID := r.resolveKeyID(entry.SSHKey, entry.SSHKeyID)
	if resolvedKeyID != 0 && resolvedKeyID != existing.SSHKeyID {
		return true
	}
	resolvedBecomeKeyID := r.resolveKeyID(entry.BecomeKey, entry.BecomeKeyID)
	if resolvedBecomeKeyID != 0 && resolvedBecomeKeyID != existing.BecomeKeyID {
		return true
	}
	resolvedRepoID := r.resolveRepoID(entry.Repository, entry.RepositoryID)
	if resolvedRepoID != 0 && resolvedRepoID != existing.RepositoryID {
		return true
	}
	return false
}

func (r *Reconciler) templateNeedsUpdate(entry TemplateEntry, existing *models.Template) bool {
	if entry.Type != "" && entry.Type != existing.Type {
		return true
	}
	if entry.App != "" && entry.App != existing.App {
		return true
	}
	if entry.Playbook != "" && entry.Playbook != existing.Playbook {
		return true
	}
	if entry.Description != "" && entry.Description != existing.Description {
		return true
	}
	if entry.GitBranch != "" && entry.GitBranch != existing.GitBranch {
		return true
	}
	if entry.Arguments != "" && entry.Arguments != existing.Arguments {
		return true
	}
	if entry.StartVersion != "" && entry.StartVersion != existing.StartVersion {
		return true
	}
	if entry.Autorun != nil && *entry.Autorun != existing.Autorun {
		return true
	}
	if entry.SuppressSuccessAlerts != nil && *entry.SuppressSuccessAlerts != existing.SuppressSuccessAlerts {
		return true
	}
	if entry.AllowOverrideArgsInTask != nil && *entry.AllowOverrideArgsInTask != existing.AllowOverrideArgsInTask {
		return true
	}

	resolvedRepoID := r.resolveRepoID(entry.Repository, entry.RepositoryID)
	if resolvedRepoID != 0 && resolvedRepoID != existing.RepositoryID {
		return true
	}
	resolvedEnvID := r.resolveVarGroupID(entry.VariableGroup, entry.EnvironmentID)
	if resolvedEnvID != 0 && resolvedEnvID != existing.EnvironmentID {
		return true
	}
	resolvedInvID := r.resolveInventoryID(entry.Inventory, entry.InventoryID)
	if resolvedInvID != 0 && resolvedInvID != existing.InventoryID {
		return true
	}
	resolvedBuildTplID := r.resolveTemplateID(entry.BuildTemplate, entry.BuildTemplateID)
	if resolvedBuildTplID != 0 && resolvedBuildTplID != existing.BuildTemplateID {
		return true
	}
	if entry.ViewID != 0 && entry.ViewID != existing.ViewID {
		return true
	}

	return false
}

// scheduleNeedsUpdate returns true if any specified schedule field differs
// from the existing server-side schedule.
func (r *Reconciler) scheduleNeedsUpdate(entry ScheduleEntry, existing *models.Schedule) bool {
	if entry.CronFormat != "" && entry.CronFormat != existing.CronFormat {
		return true
	}
	resolvedTplID := r.resolveTemplateID(entry.Template, entry.TemplateID)
	if resolvedTplID != 0 && resolvedTplID != existing.TemplateID {
		return true
	}
	if entry.Active != nil && *entry.Active != existing.Active {
		return true
	}
	return false
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
