package apply

import (
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
	client *apiclient.Semapi
	config *ApplyConfig
	recon  *Reconciler
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
func (e *Executor) Execute(plan *Plan) int {
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
				if err := e.executeAction(action); err != nil {
					style.Error(fmt.Sprintf("Failed to %s %s %q: %v", action.Action, action.Type, action.Label, err))
					errors++
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
				if err := e.executeAction(action); err != nil {
					style.Error(fmt.Sprintf("Failed to delete %s %q: %v", action.Type, action.Label, err))
					errors++
				}
			}
		}
	}

	return errors
}

func (e *Executor) executeAction(action ResourceAction) error {
	switch action.Type {
	case ResourceProject:
		return e.executeProject(action)
	case ResourceKey:
		return e.executeKey(action)
	case ResourceVariableGroup:
		return e.executeVariableGroup(action)
	case ResourceRepository:
		return e.executeRepository(action)
	case ResourceInventory:
		return e.executeInventory(action)
	case ResourceTemplate:
		return e.executeTemplate(action)
	case ResourceSchedule:
		return e.executeSchedule(action)
	default:
		return fmt.Errorf("unknown resource type: %s", action.Type)
	}
}

func (e *Executor) executeProject(action ResourceAction) error {
	switch action.Action {
	case ActionCreate:
		req := &models.ProjectRequest{
			Name: e.config.Project,
		}

		params := project.NewPostProjectsParams()
		params.Project = req

		resp, err := e.client.Project.PostProjects(params, nil)
		if err != nil {
			return err
		}

		p := resp.GetPayload()
		e.recon.SetProjectID(p.ID)
		style.Success(fmt.Sprintf("Created project %q (ID: %d)", p.Name, p.ID))

	case ActionDelete:
		params := project.NewDeleteProjectProjectIDParams()
		params.ProjectID = action.ExistingID

		_, err := e.client.Project.DeleteProjectProjectID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted project %q (ID: %d)", action.Label, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeKey(action ResourceAction) error {
	pid := e.recon.ProjectID()

	// Deletes must not touch the config: project-deletion plans reference
	// server-side resources that have no config entry (Index would be stale).
	if action.Action == ActionDelete {
		params := key_store.NewDeleteProjectProjectIDKeysKeyIDParams()
		params.ProjectID = pid
		params.KeyID = action.ExistingID

		_, err := e.client.KeyStore.DeleteProjectProjectIDKeysKeyID(params, nil)
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

		resp, err := e.client.KeyStore.PostProjectProjectIDKeys(params, nil)
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

		_, err := e.client.KeyStore.PutProjectProjectIDKeysKeyID(params, nil)
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

func (e *Executor) executeVariableGroup(action ResourceAction) error {
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

		resp, err := e.client.VariableGroup.PostProjectProjectIDEnvironment(params, nil)
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
		getResp, err := e.client.VariableGroup.GetProjectProjectIDEnvironmentEnvironmentID(getParams, nil)
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

		_, err = e.client.VariableGroup.PutProjectProjectIDEnvironmentEnvironmentID(putParams, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated variable group %q (ID: %d)", entry.Name, action.ExistingID))

	case ActionDelete:
		params := variable_group.NewDeleteProjectProjectIDEnvironmentEnvironmentIDParams()
		params.ProjectID = pid
		params.EnvironmentID = action.ExistingID

		_, err := e.client.VariableGroup.DeleteProjectProjectIDEnvironmentEnvironmentID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted variable group %q (ID: %d)", action.Label, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeRepository(action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := repository.NewDeleteProjectProjectIDRepositoriesRepositoryIDParams()
		params.ProjectID = pid
		params.RepositoryID = action.ExistingID

		_, err := e.client.Repository.DeleteProjectProjectIDRepositoriesRepositoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted repository %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Repositories[action.Index]
	sshKeyID := e.recon.resolveKeyID(entry.SSHKey, entry.SSHKeyID)

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

		resp, err := e.client.Repository.PostProjectProjectIDRepositories(params, nil)
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

		_, err := e.client.Repository.PutProjectProjectIDRepositoriesRepositoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated repository %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeInventory(action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := inventory.NewDeleteProjectProjectIDInventoryInventoryIDParams()
		params.ProjectID = pid
		params.InventoryID = action.ExistingID

		_, err := e.client.Inventory.DeleteProjectProjectIDInventoryInventoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted inventory %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Inventories[action.Index]
	sshKeyID := e.recon.resolveKeyID(entry.SSHKey, entry.SSHKeyID)
	becomeKeyID := e.recon.resolveKeyID(entry.BecomeKey, entry.BecomeKeyID)
	repoID := e.recon.resolveRepoID(entry.Repository, entry.RepositoryID)

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

		resp, err := e.client.Inventory.PostProjectProjectIDInventory(params, nil)
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

		_, err := e.client.Inventory.PutProjectProjectIDInventoryInventoryID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated inventory %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeTemplate(action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := template.NewDeleteProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = pid
		params.TemplateID = action.ExistingID

		_, err := e.client.Template.DeleteProjectProjectIDTemplatesTemplateID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted template %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Templates[action.Index]
	repoID := e.recon.resolveRepoID(entry.Repository, entry.RepositoryID)
	envID := e.recon.resolveVarGroupID(entry.VariableGroup, entry.EnvironmentID)
	invID := e.recon.resolveInventoryID(entry.Inventory, entry.InventoryID)
	buildTplID := e.recon.resolveTemplateID(entry.BuildTemplate, entry.BuildTemplateID)

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
			SurveyVars:              []*models.TemplateSurveyVar{},
			Vaults:                  []*models.TemplateVault{},
		}

		params := template.NewPostProjectProjectIDTemplatesParams()
		params.ProjectID = pid
		params.Template = req

		resp, err := e.client.Template.PostProjectProjectIDTemplates(params, nil)
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
			SurveyVars:              []*models.TemplateSurveyVar{},
			Vaults:                  []*models.TemplateVault{},
		}
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
			// Survey vars and vaults are not managed by apply configs yet —
			// preserve whatever is configured server-side instead of wiping it.
			if existing.SurveyVars != nil {
				req.SurveyVars = existing.SurveyVars
			}
			if existing.Vaults != nil {
				req.Vaults = existing.Vaults
			}
		}

		params := template.NewPutProjectProjectIDTemplatesTemplateIDParams()
		params.ProjectID = pid
		params.TemplateID = action.ExistingID
		params.Template = req

		_, err := e.client.Template.PutProjectProjectIDTemplatesTemplateID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Updated template %q (ID: %d)", entry.Name, action.ExistingID))
	}
	return nil
}

func (e *Executor) executeSchedule(action ResourceAction) error {
	pid := e.recon.ProjectID()

	if action.Action == ActionDelete {
		params := schedule.NewDeleteProjectProjectIDSchedulesScheduleIDParams()
		params.ProjectID = pid
		params.ScheduleID = action.ExistingID

		_, err := e.client.Schedule.DeleteProjectProjectIDSchedulesScheduleID(params, nil)
		if err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Deleted schedule %q (ID: %d)", action.Label, action.ExistingID))
		return nil
	}

	entry := e.config.Schedules[action.Index]
	tplID := e.recon.resolveTemplateID(entry.Template, entry.TemplateID)

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

		resp, err := e.client.Schedule.PostProjectProjectIDSchedules(params, nil)
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

		_, err := e.client.Schedule.PutProjectProjectIDSchedulesScheduleID(params, nil)
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
