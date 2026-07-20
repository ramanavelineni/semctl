package apply

import (
	"context"
	"fmt"
	"strings"

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

// Executor applies planned changes to Semaphore.
type Executor struct {
	client          *apiclient.Semapi
	config          *ApplyConfig
	recon           *Reconciler
	failFast        bool
	warnedInterrupt bool
}

// SetFailFast makes Execute stop at the first error instead of continuing
// with the remaining actions.
func (e *Executor) SetFailFast(v bool) {
	e.failFast = v
}

// NewExecutor creates a new executor.
func NewExecutor(client *apiclient.Semapi, config *ApplyConfig, recon *Reconciler) *Executor {
	return &Executor{
		client: client,
		config: config,
		recon:  recon,
	}
}

// Execute applies all planned actions and returns the number of errors.
// Cancelling ctx (Ctrl-C) stops before the next action; the in-flight HTTP
// request is cancelled too. Already-applied changes remain — re-running
// apply resumes reconciliation.
func (e *Executor) Execute(ctx context.Context, plan *Plan) int {
	errors := 0

	// Create/update order: project → keys → envs → repos → inventories → templates → schedules
	createOrder := []ResourceType{
		ResourceProject,
		ResourceKey,
		ResourceVariableGroup,
		ResourceRepository,
		ResourceInventory,
		ResourceTemplate,
		ResourceSchedule,
	}

	for _, rt := range createOrder {
		for _, action := range plan.ActionsByType(rt) {
			if action.Action == ActionCreate || action.Action == ActionUpdate {
				if e.interrupted(ctx) {
					return errors
				}
				if err := e.executeAction(ctx, action); err != nil {
					if e.interrupted(ctx) {
						return errors
					}
					style.Error(fmt.Sprintf("Failed to %s %s %q: %v", action.Action, action.Type, action.Label, err))
					errors++
					if e.failFast {
						style.Warning("Stopping at first error (--fail-fast); re-running apply resumes reconciliation.")
						return errors
					}
				}
			}
		}
	}

	// Delete order (reverse): schedules → templates → inventories → repos → envs → keys → project
	deleteOrder := []ResourceType{
		ResourceSchedule,
		ResourceTemplate,
		ResourceInventory,
		ResourceRepository,
		ResourceVariableGroup,
		ResourceKey,
		ResourceProject,
	}

	for _, rt := range deleteOrder {
		for _, action := range plan.ActionsByType(rt) {
			if action.Action == ActionDelete {
				if e.interrupted(ctx) {
					return errors
				}
				if err := e.executeAction(ctx, action); err != nil {
					if e.interrupted(ctx) {
						return errors
					}
					style.Error(fmt.Sprintf("Failed to delete %s %q: %v", action.Type, action.Label, err))
					errors++
					if e.failFast {
						style.Warning("Stopping at first error (--fail-fast); re-running apply resumes reconciliation.")
						return errors
					}
				}
			}
		}
	}

	return errors
}

// interrupted reports whether ctx was cancelled, warning once about the
// partially reconciled state. An action failing because its HTTP request was
// cancelled is not counted as an apply error.
func (e *Executor) interrupted(ctx context.Context) bool {
	if ctx.Err() == nil {
		return false
	}
	if !e.warnedInterrupt {
		e.warnedInterrupt = true
		style.Warning("Interrupted — stopping apply. Changes already applied remain; re-running apply resumes reconciliation.")
	}
	return true
}

func (e *Executor) executeAction(ctx context.Context, action ResourceAction) error {
	switch action.Type {
	case ResourceProject:
		return e.executeProject(ctx, action)
	case ResourceKey:
		return e.executeKey(ctx, action)
	case ResourceVariableGroup:
		return e.executeVariableGroup(ctx, action)
	case ResourceRepository:
		return e.executeRepository(ctx, action)
	case ResourceInventory:
		return e.executeInventory(ctx, action)
	case ResourceTemplate:
		return e.executeTemplate(ctx, action)
	case ResourceSchedule:
		return e.executeSchedule(ctx, action)
	default:
		return fmt.Errorf("unknown resource type: %s", action.Type)
	}
}

func (e *Executor) executeProject(ctx context.Context, action ResourceAction) error {
	switch action.Action {
	case ActionCreate:
		req := &models.ProjectRequest{
			Name: e.config.Project,
		}

		params := project.NewPostProjectsParams()
		params.Project = req

		resp, err := e.client.Project.PostProjectsContext(ctx, params, nil)
		if err != nil {
			return err
		}

		p := resp.GetPayload()
		e.recon.SetProjectID(p.ID)
		style.Success(fmt.Sprintf("Created project %q (ID: %d)", p.Name, p.ID))

	case ActionDelete:
		params := project.NewDeleteProjectProjectIDParams()
		params.ProjectID = action.ExistingID

		_, err := e.client.Project.DeleteProjectProjectIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted project %q (ID: %d)", action.Label, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeKey(ctx context.Context, action ResourceAction) error {
	pid := e.recon.ProjectID()

	// Deletes must not touch the config: project-deletion plans reference
	// server-side resources that have no config entry (Index would be stale).
	if action.Action == ActionDelete {
		params := key_store.NewDeleteProjectProjectIDKeysKeyIDParams()
		params.ProjectID = pid
		params.KeyID = action.ExistingID

		_, err := e.client.KeyStore.DeleteProjectProjectIDKeysKeyIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted key %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Keys[action.Index]

	switch action.Action {
	case ActionCreate:
		req := &models.AccessKeyRequest{
			ProjectID: pid,
			Name:      entry.Name,
			Type:      entry.Type,
		}
		e.applyKeySecrets(req, entry)

		params := key_store.NewPostProjectProjectIDKeysParams()
		params.ProjectID = pid
		params.AccessKey = req

		resp, err := e.client.KeyStore.PostProjectProjectIDKeysContext(ctx, params, nil)
		if err != nil {
			return err
		}

		k := resp.GetPayload()
		e.recon.KeyIDByName[strings.ToLower(k.Name)] = k.ID
		style.Success(fmt.Sprintf("Created key %q (ID: %d)", k.Name, k.ID))

	case ActionUpdate:
		req := &models.AccessKeyRequest{
			ID:             action.ExistingID,
			ProjectID:      pid,
			Name:           entry.Name,
			Type:           entry.Type,
			OverrideSecret: true,
		}
		e.applyKeySecrets(req, entry)

		params := key_store.NewPutProjectProjectIDKeysKeyIDParams()
		params.ProjectID = pid
		params.KeyID = action.ExistingID
		params.AccessKey = req

		_, err := e.client.KeyStore.PutProjectProjectIDKeysKeyIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated key %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) applyKeySecrets(req *models.AccessKeyRequest, entry KeyEntry) {
	switch entry.Type {
	case "ssh":
		if entry.SSH != nil {
			req.SSH = &models.AccessKeyRequestSSH{
				Login:      entry.SSH.Login,
				PrivateKey: entry.SSH.PrivateKey,
				Passphrase: entry.SSH.Passphrase,
			}
		}
	case "login_password":
		if entry.LoginPassword != nil {
			req.LoginPassword = &models.AccessKeyRequestLoginPassword{
				Login:    entry.LoginPassword.Login,
				Password: entry.LoginPassword.Password,
			}
		}
	}
}

func (e *Executor) executeVariableGroup(ctx context.Context, action ResourceAction) error {
	pid := e.recon.ProjectID()

	switch action.Action {
	case ActionCreate:
		entry := e.config.VariableGroups[action.Index]
		req := &models.EnvironmentRequest{
			ProjectID: pid,
			Name:      entry.Name,
			JSON:      VarsToJSON(entry.Variables),
			Env:       EnvVarsToJSON(entry.EnvironmentVariables),
			Secrets:   buildAllSecretRequests(entry.Secrets, entry.SecretEnvironmentVariables, "create"),
		}

		params := variable_group.NewPostProjectProjectIDEnvironmentParams()
		params.ProjectID = pid
		params.Environment = req

		resp, err := e.client.VariableGroup.PostProjectProjectIDEnvironmentContext(ctx, params, nil)
		if err != nil {
			return err
		}

		env := resp.GetPayload()
		e.recon.VarGroupIDByName[strings.ToLower(env.Name)] = env.ID
		style.Success(fmt.Sprintf("Created variable group %q (ID: %d)", env.Name, env.ID))

	case ActionUpdate:
		entry := e.config.VariableGroups[action.Index]

		// Fetch existing to get current secret IDs
		getParams := variable_group.NewGetProjectProjectIDEnvironmentEnvironmentIDParams()
		getParams.ProjectID = pid
		getParams.EnvironmentID = action.ExistingID
		getResp, err := e.client.VariableGroup.GetProjectProjectIDEnvironmentEnvironmentIDContext(ctx, getParams, nil)
		if err != nil {
			return fmt.Errorf("fetching existing variable group: %w", err)
		}
		existing := getResp.GetPayload()

		req := &models.EnvironmentRequest{
			ID:        action.ExistingID,
			ProjectID: pid,
			Name:      entry.Name,
			JSON:      VarsToJSON(entry.Variables),
			Env:       EnvVarsToJSON(entry.EnvironmentVariables),
			Secrets:   buildAllSecretUpdateRequests(entry.Secrets, entry.SecretEnvironmentVariables, existing.Secrets),
		}

		putParams := variable_group.NewPutProjectProjectIDEnvironmentEnvironmentIDParams()
		putParams.ProjectID = pid
		putParams.EnvironmentID = action.ExistingID
		putParams.Environment = req

		_, err = e.client.VariableGroup.PutProjectProjectIDEnvironmentEnvironmentIDContext(ctx, putParams, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated variable group %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := variable_group.NewDeleteProjectProjectIDEnvironmentEnvironmentIDParams()
		params.ProjectID = pid
		params.EnvironmentID = action.ExistingID

		_, err := e.client.VariableGroup.DeleteProjectProjectIDEnvironmentEnvironmentIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted variable group %q (ID: %d)", action.Label, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeRepository(ctx context.Context, action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := repository.NewDeleteProjectProjectIDRepositoriesRepositoryIDParams()
		params.ProjectID = pid
		params.RepositoryID = action.ExistingID

		_, err := e.client.Repository.DeleteProjectProjectIDRepositoriesRepositoryIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted repository %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Repositories[action.Index]
	sshKeyID, err := mustResolve("key", entry.SSHKey, e.recon.resolveKeyID(entry.SSHKey, entry.SSHKeyID))
	if err != nil {
		return err
	}

	switch action.Action {
	case ActionCreate:
		req := &models.RepositoryRequest{
			ProjectID: pid,
			Name:      entry.Name,
			GitURL:    entry.GitURL,
			GitBranch: entry.GitBranch,
			SSHKeyID:  sshKeyID,
		}

		params := repository.NewPostProjectProjectIDRepositoriesParams()
		params.ProjectID = pid
		params.Repository = req

		resp, err := e.client.Repository.PostProjectProjectIDRepositoriesContext(ctx, params, nil)
		if err != nil {
			return err
		}

		r := resp.GetPayload()
		e.recon.RepoIDByName[strings.ToLower(r.Name)] = r.ID
		style.Success(fmt.Sprintf("Created repository %q (ID: %d)", r.Name, r.ID))

	case ActionUpdate:
		req := &models.RepositoryRequest{
			ID:        action.ExistingID,
			ProjectID: pid,
			Name:      entry.Name,
			GitURL:    entry.GitURL,
			GitBranch: entry.GitBranch,
			SSHKeyID:  sshKeyID,
		}
		// Merge over the existing resource: fields omitted from the config
		// keep their current server-side values instead of being zeroed.
		if existing := e.recon.ExistingRepoByID[action.ExistingID]; existing != nil {
			req.GitURL = mergeStr(entry.GitURL, existing.GitURL)
			req.GitBranch = mergeStr(entry.GitBranch, existing.GitBranch)
			req.SSHKeyID = mergeID(sshKeyID, existing.SSHKeyID)
		}

		params := repository.NewPutProjectProjectIDRepositoriesRepositoryIDParams()
		params.ProjectID = pid
		params.RepositoryID = action.ExistingID
		params.Repository = req

		_, err := e.client.Repository.PutProjectProjectIDRepositoriesRepositoryIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated repository %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeInventory(ctx context.Context, action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := inventory.NewDeleteProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = pid
		params.InventoryID = action.ExistingID

		_, err := e.client.Inventory.DeleteProjectProjectIDInventoryInventoryIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted inventory %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Inventories[action.Index]
	sshKeyID, err := mustResolve("key", entry.SSHKey, e.recon.resolveKeyID(entry.SSHKey, entry.SSHKeyID))
	if err != nil {
		return err
	}
	becomeKeyID, err := mustResolve("key", entry.BecomeKey, e.recon.resolveKeyID(entry.BecomeKey, entry.BecomeKeyID))
	if err != nil {
		return err
	}
	repoID, err := mustResolve("repository", entry.Repository, e.recon.resolveRepoID(entry.Repository, entry.RepositoryID))
	if err != nil {
		return err
	}

	switch action.Action {
	case ActionCreate:
		req := &models.InventoryRequest{
			ProjectID:    pid,
			Name:         entry.Name,
			Type:         entry.Type,
			Inventory:    entry.Inventory,
			SSHKeyID:     sshKeyID,
			BecomeKeyID:  becomeKeyID,
			RepositoryID: repoID,
		}

		params := inventory.NewPostProjectProjectIDInventoryParams()
		params.ProjectID = pid
		params.Inventory = req

		resp, err := e.client.Inventory.PostProjectProjectIDInventoryContext(ctx, params, nil)
		if err != nil {
			return err
		}

		inv := resp.GetPayload()
		e.recon.InventoryIDByName[strings.ToLower(inv.Name)] = inv.ID
		style.Success(fmt.Sprintf("Created inventory %q (ID: %d)", inv.Name, inv.ID))

	case ActionUpdate:
		req := &models.InventoryRequest{
			ID:           action.ExistingID,
			ProjectID:    pid,
			Name:         entry.Name,
			Type:         entry.Type,
			Inventory:    entry.Inventory,
			SSHKeyID:     sshKeyID,
			BecomeKeyID:  becomeKeyID,
			RepositoryID: repoID,
		}
		if existing := e.recon.ExistingInventoryByID[action.ExistingID]; existing != nil {
			req.Type = mergeStr(entry.Type, existing.Type)
			req.Inventory = mergeStr(entry.Inventory, existing.Inventory)
			req.SSHKeyID = mergeID(sshKeyID, existing.SSHKeyID)
			req.BecomeKeyID = mergeID(becomeKeyID, existing.BecomeKeyID)
			req.RepositoryID = mergeID(repoID, existing.RepositoryID)
		}

		params := inventory.NewPutProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = pid
		params.InventoryID = action.ExistingID
		params.Inventory = req

		_, err := e.client.Inventory.PutProjectProjectIDInventoryInventoryIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated inventory %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeTemplate(ctx context.Context, action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := template.NewDeleteProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = pid
		params.TemplateID = action.ExistingID

		_, err := e.client.Template.DeleteProjectProjectIDTemplatesTemplateIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted template %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Templates[action.Index]
	repoID, err := mustResolve("repository", entry.Repository, e.recon.resolveRepoID(entry.Repository, entry.RepositoryID))
	if err != nil {
		return err
	}
	envID, err := mustResolve("variable group", entry.VariableGroup, e.recon.resolveVarGroupID(entry.VariableGroup, entry.EnvironmentID))
	if err != nil {
		return err
	}
	invID, err := mustResolve("inventory", entry.Inventory, e.recon.resolveInventoryID(entry.Inventory, entry.InventoryID))
	if err != nil {
		return err
	}
	buildTplID, err := mustResolve("template", entry.BuildTemplate, e.recon.resolveTemplateID(entry.BuildTemplate, entry.BuildTemplateID))
	if err != nil {
		return err
	}

	switch action.Action {
	case ActionCreate:
		req := &models.TemplateRequest{
			ProjectID:               pid,
			Name:                    entry.Name,
			Type:                    entry.Type,
			App:                     entry.App,
			Playbook:                entry.Playbook,
			Description:             entry.Description,
			GitBranch:               entry.GitBranch,
			Arguments:               entry.Arguments,
			StartVersion:            entry.StartVersion,
			Autorun:                 entry.Autorun != nil && *entry.Autorun,
			SuppressSuccessAlerts:   entry.SuppressSuccessAlerts != nil && *entry.SuppressSuccessAlerts,
			AllowOverrideArgsInTask: entry.AllowOverrideArgsInTask != nil && *entry.AllowOverrideArgsInTask,
			RepositoryID:            repoID,
			EnvironmentID:           envID,
			InventoryID:             invID,
			BuildTemplateID:         buildTplID,
			ViewID:                  entry.ViewID,
		}
		PreserveUnmanagedTemplateFields(req, nil)

		params := template.NewPostProjectProjectIDTemplatesParams()
		params.ProjectID = pid
		params.Template = req

		resp, err := e.client.Template.PostProjectProjectIDTemplatesContext(ctx, params, nil)
		if err != nil {
			return err
		}

		t := resp.GetPayload()
		e.recon.TemplateIDByName[strings.ToLower(t.Name)] = t.ID
		style.Success(fmt.Sprintf("Created template %q (ID: %d)", t.Name, t.ID))

	case ActionUpdate:
		req := &models.TemplateRequest{
			ID:                      action.ExistingID,
			ProjectID:               pid,
			Name:                    entry.Name,
			Type:                    entry.Type,
			App:                     entry.App,
			Playbook:                entry.Playbook,
			Description:             entry.Description,
			GitBranch:               entry.GitBranch,
			Arguments:               entry.Arguments,
			StartVersion:            entry.StartVersion,
			Autorun:                 entry.Autorun != nil && *entry.Autorun,
			SuppressSuccessAlerts:   entry.SuppressSuccessAlerts != nil && *entry.SuppressSuccessAlerts,
			AllowOverrideArgsInTask: entry.AllowOverrideArgsInTask != nil && *entry.AllowOverrideArgsInTask,
			RepositoryID:            repoID,
			EnvironmentID:           envID,
			InventoryID:             invID,
			BuildTemplateID:         buildTplID,
			ViewID:                  entry.ViewID,
		}
		PreserveUnmanagedTemplateFields(req, e.recon.ExistingTemplateByID[action.ExistingID])
		if existing := e.recon.ExistingTemplateByID[action.ExistingID]; existing != nil {
			req.Type = mergeStr(entry.Type, existing.Type)
			req.App = mergeStr(entry.App, existing.App)
			req.Playbook = mergeStr(entry.Playbook, existing.Playbook)
			req.Description = mergeStr(entry.Description, existing.Description)
			req.GitBranch = mergeStr(entry.GitBranch, existing.GitBranch)
			req.Arguments = mergeStr(entry.Arguments, existing.Arguments)
			req.StartVersion = mergeStr(entry.StartVersion, existing.StartVersion)
			req.Autorun = mergeBool(entry.Autorun, existing.Autorun)
			req.SuppressSuccessAlerts = mergeBool(entry.SuppressSuccessAlerts, existing.SuppressSuccessAlerts)
			req.AllowOverrideArgsInTask = mergeBool(entry.AllowOverrideArgsInTask, existing.AllowOverrideArgsInTask)
			req.RepositoryID = mergeID(repoID, existing.RepositoryID)
			req.EnvironmentID = mergeID(envID, existing.EnvironmentID)
			req.InventoryID = mergeID(invID, existing.InventoryID)
			req.BuildTemplateID = mergeID(buildTplID, existing.BuildTemplateID)
			req.ViewID = mergeID(entry.ViewID, existing.ViewID)
		}

		params := template.NewPutProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = pid
		params.TemplateID = action.ExistingID
		params.Template = req

		_, err := e.client.Template.PutProjectProjectIDTemplatesTemplateIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated template %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeSchedule(ctx context.Context, action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := schedule.NewDeleteProjectProjectIDSchedulesScheduleIDParams()
		params.ProjectID = pid
		params.ScheduleID = action.ExistingID

		_, err := e.client.Schedule.DeleteProjectProjectIDSchedulesScheduleIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted schedule %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Schedules[action.Index]
	tplID, err := mustResolve("template", entry.Template, e.recon.resolveTemplateID(entry.Template, entry.TemplateID))
	if err != nil {
		return err
	}

	active := true
	if entry.Active != nil {
		active = *entry.Active
	}

	switch action.Action {
	case ActionCreate:
		req := &models.ScheduleRequest{
			ProjectID:  pid,
			TemplateID: tplID,
			Name:       entry.Name,
			CronFormat: entry.CronFormat,
			Active:     active,
		}

		params := schedule.NewPostProjectProjectIDSchedulesParams()
		params.ProjectID = pid
		params.Schedule = req

		resp, err := e.client.Schedule.PostProjectProjectIDSchedulesContext(ctx, params, nil)
		if err != nil {
			return err
		}

		s := resp.GetPayload()
		style.Success(fmt.Sprintf("Created schedule %q (ID: %d)", s.Name, s.ID))

	case ActionUpdate:
		req := &models.ScheduleRequest{
			ID:         action.ExistingID,
			ProjectID:  pid,
			TemplateID: tplID,
			Name:       entry.Name,
			CronFormat: entry.CronFormat,
			Active:     active,
		}
		if existing := e.recon.ExistingScheduleByID[action.ExistingID]; existing != nil {
			req.CronFormat = mergeStr(entry.CronFormat, existing.CronFormat)
			req.TemplateID = mergeID(tplID, existing.TemplateID)
			req.Active = mergeBool(entry.Active, existing.Active)
		}

		params := schedule.NewPutProjectProjectIDSchedulesScheduleIDParams()
		params.ProjectID = pid
		params.ScheduleID = action.ExistingID
		params.Schedule = req

		_, err := e.client.Schedule.PutProjectProjectIDSchedulesScheduleIDContext(ctx, params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated schedule %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

// Merge helpers: config fields left empty/nil fall back to the existing
// server-side value on update, so partial configs don't zero out fields.

func mergeStr(entry, existing string) string {
	if entry != "" {
		return entry
	}
	return existing
}

func mergeID(resolved, existing int64) int64 {
	if resolved != 0 {
		return resolved
	}
	return existing
}

func mergeBool(entry *bool, existing bool) bool {
	if entry != nil {
		return *entry
	}
	return existing
}

// buildAllSecretRequests combines both secret types (var + env) into a single slice for creation.
func buildAllSecretRequests(secrets, secretEnvVars map[string]string, operation string) []*models.EnvironmentSecretRequest {
	var result []*models.EnvironmentSecretRequest
	for name, value := range secrets {
		result = append(result, &models.EnvironmentSecretRequest{
			Name:      name,
			Secret:    value,
			Type:      "var",
			Operation: operation,
		})
	}
	for name, value := range secretEnvVars {
		result = append(result, &models.EnvironmentSecretRequest{
			Name:      name,
			Secret:    value,
			Type:      "env",
			Operation: operation,
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// buildAllSecretUpdateRequests builds secret requests for an update, matching existing secrets by name.
// Existing secrets get "update" with their ID; new secrets get "create".
func buildAllSecretUpdateRequests(secrets, secretEnvVars map[string]string, existing []*models.EnvironmentSecret) []*models.EnvironmentSecretRequest {
	if len(secrets) == 0 && len(secretEnvVars) == 0 {
		return nil
	}

	existingByName := make(map[string]*models.EnvironmentSecret)
	for _, s := range existing {
		existingByName[strings.ToLower(s.Name)] = s
	}

	var result []*models.EnvironmentSecretRequest
	for name, value := range secrets {
		if es, ok := existingByName[strings.ToLower(name)]; ok {
			result = append(result, &models.EnvironmentSecretRequest{
				ID:        es.ID,
				Name:      name,
				Secret:    value,
				Type:      "var",
				Operation: "update",
			})
		} else {
			result = append(result, &models.EnvironmentSecretRequest{
				Name:      name,
				Secret:    value,
				Type:      "var",
				Operation: "create",
			})
		}
	}
	for name, value := range secretEnvVars {
		if es, ok := existingByName[strings.ToLower(name)]; ok {
			result = append(result, &models.EnvironmentSecretRequest{
				ID:        es.ID,
				Name:      name,
				Secret:    value,
				Type:      "env",
				Operation: "update",
			})
		} else {
			result = append(result, &models.EnvironmentSecretRequest{
				Name:      name,
				Secret:    value,
				Type:      "env",
				Operation: "create",
			})
		}
	}
	return result
}
